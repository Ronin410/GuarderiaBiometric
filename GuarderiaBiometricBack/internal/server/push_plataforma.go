package server

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"biometrico/internal/middleware"
)

// push_plataforma.go -- suscripción push del DUEÑO de la plataforma
// (Alejandro), para las notificaciones del chat de soporte ("quiero que
// las notificaciones de los chats sean notificaciones push... que lleguen
// al iniciar como mi cuenta de admin"). Reutiliza el MISMO par de llaves
// VAPID que ya usa el resto de la app (padres/staff, ver push.go) -- VAPID
// es una credencial del SERVIDOR, no de cada suscriptor, así que no hace
// falta ninguna variable de entorno nueva. La suscripción vive en su
// propia tabla (push_suscripciones_plataforma) porque no hay guarderia_id/
// padre_id/personal_id que amarrarle: esta sesión se autentica con
// PLATFORM_ADMIN_KEY, no con el JWT normal.
func (s *Server) registrarRutasPushPlataforma(r *gin.Engine) {
	platform := middleware.RequirePlatformKey(s.PlatformAdminKey)

	r.POST("/plataforma/push/suscribir", platform, func(c *gin.Context) {
		if !s.PushConfigurado() {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Notificaciones push no configuradas en el servidor"})
			return
		}

		var input pushSubscriptionInput
		if err := c.ShouldBindJSON(&input); err != nil || input.Endpoint == "" || input.Keys.P256dh == "" || input.Keys.Auth == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Suscripción inválida"})
			return
		}

		if _, err := s.DB.Exec(
			`INSERT INTO push_suscripciones_plataforma (endpoint, p256dh, auth)
             VALUES ($1, $2, $3)
             ON CONFLICT (endpoint) DO UPDATE SET p256dh = EXCLUDED.p256dh, auth = EXCLUDED.auth`,
			input.Endpoint, input.Keys.P256dh, input.Keys.Auth,
		); err != nil {
			s.logError(c, "No se pudo guardar la suscripción push de plataforma", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo guardar la suscripción"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "Suscripción guardada"})
	})

	r.DELETE("/plataforma/push/suscribir", platform, func(c *gin.Context) {
		var input struct {
			Endpoint string `json:"endpoint"`
		}
		if err := c.ShouldBindJSON(&input); err != nil || input.Endpoint == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "endpoint requerido"})
			return
		}

		s.DB.Exec(`DELETE FROM push_suscripciones_plataforma WHERE endpoint = $1`, input.Endpoint)
		c.JSON(http.StatusOK, gin.H{"status": "Suscripción eliminada"})
	})
}

// notificarPlataformaPush manda el mismo push a TODAS las suscripciones del
// dueño de la plataforma -- puede tener varias (celular, computadora) igual
// que un papá o un miembro del staff. URL "/plataforma" en vez de "/": al
// tocar la notificación debe abrir el inbox de soporte, no la página de
// presentación pública.
func (s *Server) notificarPlataformaPush(titulo, cuerpo string) {
	if !s.PushConfigurado() {
		return
	}

	rows, err := s.DB.Query(`SELECT id, endpoint, p256dh, auth FROM push_suscripciones_plataforma`)
	if err != nil {
		s.logError(nil, "notificarPlataformaPush: error consultando suscripciones", err)
		return
	}
	var destinos []destinoPush
	for rows.Next() {
		var d destinoPush
		if err := rows.Scan(&d.id, &d.sub.Endpoint, &d.sub.Keys.P256dh, &d.sub.Keys.Auth); err != nil {
			continue
		}
		destinos = append(destinos, d)
	}
	rows.Close()

	payload, err := json.Marshal(pushPayload{Titulo: titulo, Cuerpo: cuerpo, URL: "/plataforma"})
	if err != nil {
		s.logError(nil, "notificarPlataformaPush: error serializando payload", err)
		return
	}

	s.enviarPushATodos(destinos, payload, "push_suscripciones_plataforma")
}
