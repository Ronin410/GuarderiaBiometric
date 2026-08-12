package server

import (
	"encoding/json"
	"log"
	"net/http"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/gin-gonic/gin"

	"biometrico/internal/middleware"
)

type pushSubscriptionInput struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

type pushPayload struct {
	Titulo string `json:"titulo"`
	Cuerpo string `json:"cuerpo"`
	URL    string `json:"url"`
}

func (s *Server) registrarRutasPush(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)

	// --- GUARDAR SUSCRIPCIÓN (cualquier usuario autenticado, típicamente rol "papa") ---
	r.POST("/push/suscribir", auth, func(c *gin.Context) {
		if !s.PushConfigurado() {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Notificaciones push no configuradas en el servidor"})
			return
		}

		gID, _ := c.Get("guarderia_id")
		userID, _ := c.Get("user_id")

		var input pushSubscriptionInput
		if err := c.ShouldBindJSON(&input); err != nil || input.Endpoint == "" || input.Keys.P256dh == "" || input.Keys.Auth == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Suscripción inválida"})
			return
		}

		query := `
        INSERT INTO push_subscripciones (padre_id, guarderia_id, endpoint, p256dh, auth)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (endpoint) DO UPDATE SET
            padre_id = EXCLUDED.padre_id,
            p256dh = EXCLUDED.p256dh,
            auth = EXCLUDED.auth`

		if _, err := s.DB.Exec(query, userID, gID, input.Endpoint, input.Keys.P256dh, input.Keys.Auth); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo guardar la suscripción"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "Suscripción guardada"})
	})

	// --- ELIMINAR SUSCRIPCIÓN (cuando el usuario desactiva notificaciones) ---
	r.DELETE("/push/suscribir", auth, func(c *gin.Context) {
		var input struct {
			Endpoint string `json:"endpoint"`
		}
		if err := c.ShouldBindJSON(&input); err != nil || input.Endpoint == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "endpoint requerido"})
			return
		}

		s.DB.Exec("DELETE FROM push_subscripciones WHERE endpoint = $1", input.Endpoint)
		c.JSON(http.StatusOK, gin.H{"status": "Suscripción eliminada"})
	})
}

// notificarEvento avisa a TODOS los tutores vinculados a un niño (no solo a
// quien disparó el movimiento). Debe llamarse siempre como "go s.notificarEvento(...)"
// desde el handler: nunca debe frenar la respuesta al kiosco/staff.
func (s *Server) notificarEvento(hijoID int, evento string, detalle string) {
	if !s.PushConfigurado() {
		return
	}

	var nombre string
	if err := s.DB.QueryRow("SELECT nombre_niño FROM hijos WHERE id = $1", hijoID).Scan(&nombre); err != nil {
		log.Printf("notificarEvento: no se pudo obtener el nombre del niño %d: %v", hijoID, err)
		return
	}

	var titulo, cuerpo string
	switch evento {
	case "ENTRADA":
		titulo = "🟢 Entrada registrada"
		cuerpo = nombre + " llegó a la guardería."
	case "SALIDA":
		titulo = "🟠 Salida registrada"
		cuerpo = nombre + " salió de la guardería."
	case "BITACORA":
		titulo = "📋 Bitácora actualizada"
		cuerpo = nombre + ": " + detalle
	case "RECORDATORIO_PAGO":
		titulo = "💰 Recordatorio de pago"
		cuerpo = "La colegiatura de " + nombre + " del periodo " + detalle + " sigue pendiente."
	default:
		titulo = "BioSafe"
		cuerpo = nombre + ": " + detalle
	}

	rows, err := s.DB.Query(`
        SELECT DISTINCT ps.id, ps.endpoint, ps.p256dh, ps.auth
        FROM push_subscripciones ps
        INNER JOIN tutor_hijos th ON th.padre_id = ps.padre_id
        WHERE th.hijo_id = $1`, hijoID)
	if err != nil {
		log.Printf("notificarEvento: error consultando suscripciones: %v", err)
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

	payload, err := json.Marshal(pushPayload{Titulo: titulo, Cuerpo: cuerpo, URL: "/"})
	if err != nil {
		log.Printf("notificarEvento: error serializando payload: %v", err)
		return
	}

	s.enviarPushATodos(destinos, payload)
}

// notificarCircular avisa a TODOS los tutores de la guardería (no solo a los
// de un niño en particular, a diferencia de notificarEvento) cuando se
// publica una circular nueva. Igual que notificarEvento, debe llamarse como
// "go s.notificarCircular(...)" -- nunca debe frenar la respuesta al panel.
func (s *Server) notificarCircular(guarderiaID any, titulo, contenido string) {
	if !s.PushConfigurado() {
		return
	}

	rows, err := s.DB.Query(`SELECT id, endpoint, p256dh, auth FROM push_subscripciones WHERE guarderia_id = $1`, guarderiaID)
	if err != nil {
		log.Printf("notificarCircular: error consultando suscripciones: %v", err)
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

	// Recorte por runas, no por bytes: el contenido es texto en español con
	// acentos/ñ (multi-byte en UTF-8) -- cortar por índice de byte podría
	// partir un carácter a la mitad y mandar texto corrupto en la notificación.
	cuerpo := contenido
	runas := []rune(contenido)
	if len(runas) > 120 {
		cuerpo = string(runas[:120]) + "…"
	}

	payload, err := json.Marshal(pushPayload{Titulo: "📢 " + titulo, Cuerpo: cuerpo, URL: "/"})
	if err != nil {
		log.Printf("notificarCircular: error serializando payload: %v", err)
		return
	}

	s.enviarPushATodos(destinos, payload)
}

// destinoPush es una suscripción push resuelta, lista para mandarle una
// notificación -- compartido entre notificarEvento (por niño) y
// notificarCircular (por guardería completa).
type destinoPush struct {
	id  int
	sub webpush.Subscription
}

// enviarPushATodos manda el mismo payload a cada destino y limpia del lado
// de la BD cualquier suscripción que el navegador ya invalidó (404/410) --
// lógica de envío común para no duplicarla entre notificarEvento y
// notificarCircular.
func (s *Server) enviarPushATodos(destinos []destinoPush, payload []byte) {
	for _, d := range destinos {
		resp, err := webpush.SendNotification(payload, &d.sub, &webpush.Options{
			VAPIDPublicKey:  s.VapidPublicKey,
			VAPIDPrivateKey: s.VapidPrivateKey,
			Subscriber:      s.VapidSubject,
			TTL:             30,
		})
		if err != nil {
			log.Printf("enviarPushATodos: error enviando a suscripción %d: %v", d.id, err)
			continue
		}
		resp.Body.Close()

		// 404/410: la suscripción ya no existe del lado del navegador. La limpiamos.
		if resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound {
			s.DB.Exec("DELETE FROM push_subscripciones WHERE id = $1", d.id)
		}
	}
}
