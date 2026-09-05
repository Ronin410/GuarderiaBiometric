package server

import (
	"encoding/json"
	"net/http"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"

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
		rol, _ := c.Get("rol")

		var input pushSubscriptionInput
		if err := c.ShouldBindJSON(&input); err != nil || input.Endpoint == "" || input.Keys.P256dh == "" || input.Keys.Auth == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Suscripción inválida"})
			return
		}

		// "A la guardería también le llegarán notificaciones" -- ahora
		// también se suscribe staff/admin, no solo papás. Cada fila trae UNO
		// de los dos ids (nunca ambos): se decide aquí según el rol de quien
		// se suscribe, no según qué campos mandó el cliente.
		var padreID, personalID any
		if rol == "papa" {
			padreID = userID
		} else {
			personalID = userID
		}

		query := `
        INSERT INTO push_subscripciones (padre_id, personal_id, guarderia_id, endpoint, p256dh, auth)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (endpoint) DO UPDATE SET
            padre_id = EXCLUDED.padre_id,
            personal_id = EXCLUDED.personal_id,
            p256dh = EXCLUDED.p256dh,
            auth = EXCLUDED.auth`

		if _, err := s.DB.Exec(query, padreID, personalID, gID, input.Endpoint, input.Keys.P256dh, input.Keys.Auth); err != nil {
			s.logError(c, "No se pudo guardar la suscripción push", err)
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
//
// Manda tanto por Web Push (push_subscripciones, PWA del navegador) como
// por Expo (push_tokens_expo, app nativa de papás) -- son mecanismos
// independientes, cada uno con su propia tabla de destinos y su propio
// "si no hay nada que mandar, no truena", así que un papá puede tener
// activado uno, el otro, los dos, o ninguno.
func (s *Server) notificarEvento(hijoID int, evento string, detalle string) {
	var nombre string
	if err := s.DB.QueryRow("SELECT nombre_niño FROM hijos WHERE id = $1", hijoID).Scan(&nombre); err != nil {
		s.logError(nil, "notificarEvento: no se pudo obtener el nombre del niño", err, "hijo_id", hijoID)
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
		titulo = "Pasitos"
		cuerpo = nombre + ": " + detalle
	}

	s.enviarWebPushYExpo(
		`SELECT DISTINCT ps.id, ps.endpoint, ps.p256dh, ps.auth FROM push_subscripciones ps
         INNER JOIN tutor_hijos th ON th.padre_id = ps.padre_id WHERE th.hijo_id = $1`,
		`SELECT DISTINCT pte.id, pte.token FROM push_tokens_expo pte
         INNER JOIN tutor_hijos th ON th.padre_id = pte.padre_id WHERE th.hijo_id = $1`,
		titulo, cuerpo, "notificarEvento", hijoID,
	)
}

// notificarCircular avisa a TODOS los tutores de la guardería (no solo a los
// de un niño en particular, a diferencia de notificarEvento) cuando se
// publica una circular nueva. Igual que notificarEvento, debe llamarse como
// "go s.notificarCircular(...)" -- nunca debe frenar la respuesta al panel.
// grupos vacío = la publicación va para toda la guardería, que es el
// comportamiento de siempre. Con grupos, la notificación solo sale hacia los
// tutores que tienen un hijo activo en alguno de ellos -- si el aviso no se
// le muestra a una familia, tampoco tiene por qué sonarle el teléfono.
func (s *Server) notificarCircular(guarderiaID any, titulo, contenido string, grupos []int) {
	// Recorte por runas, no por bytes: el contenido es texto en español con
	// acentos/ñ (multi-byte en UTF-8) -- cortar por índice de byte podría
	// partir un carácter a la mitad y mandar texto corrupto en la notificación.
	cuerpo := contenido
	runas := []rune(contenido)
	if len(runas) > 120 {
		cuerpo = string(runas[:120]) + "…"
	}

	if len(grupos) == 0 {
		s.enviarWebPushYExpo(
			`SELECT id, endpoint, p256dh, auth FROM push_subscripciones WHERE guarderia_id = $1`,
			`SELECT id, token FROM push_tokens_expo WHERE guarderia_id = $1`,
			"📢 "+titulo, cuerpo, "notificarCircular", guarderiaID,
		)
		return
	}

	// El mismo criterio que condicionVisibleParaPadre en destinatarios.go,
	// pero contra el padre_id de la suscripción en vez del del token.
	const filtroGrupos = ` AND EXISTS (
        SELECT 1 FROM tutor_hijos th
        JOIN hijos h ON h.id = th.hijo_id AND h.activo
        WHERE th.padre_id = sub.padre_id AND h.grupo_id = ANY($2)
    )`

	s.enviarWebPushYExpo(
		`SELECT sub.id, sub.endpoint, sub.p256dh, sub.auth FROM push_subscripciones sub WHERE sub.guarderia_id = $1`+filtroGrupos,
		`SELECT sub.id, sub.token FROM push_tokens_expo sub WHERE sub.guarderia_id = $1`+filtroGrupos,
		"📢 "+titulo, cuerpo, "notificarCircular", guarderiaID, pq.Array(grupos),
	)
}

// notificarMensajeChat avisa a UN tutor (no a toda la guardería, a
// diferencia de notificarCircular) cuando staff le responde en el chat
// privado. No se manda contenido del mensaje en la notificación a propósito
// -- push va sobre HTTPS pero el payload queda en el log del navegador/SO,
// y son datos de una conversación privada.
func (s *Server) notificarMensajeChat(padreID int) {
	s.enviarWebPushYExpo(
		`SELECT id, endpoint, p256dh, auth FROM push_subscripciones WHERE padre_id = $1`,
		`SELECT id, token FROM push_tokens_expo WHERE padre_id = $1`,
		"💬 Nuevo mensaje", "La guardería te envió un mensaje.", "notificarMensajeChat", padreID,
	)
}

// notificarStaffDeGuarderia avisa a TODO el staff/admin de una guardería
// (no a un papá) -- "a la guardería también le llegarán notificaciones...
// de pedidos de comedor y otras cosas". Igual que notificarCircular, debe
// llamarse como "go s.notificarStaffDeGuarderia(...)".
func (s *Server) notificarStaffDeGuarderia(guarderiaID any, titulo, cuerpo string) {
	s.enviarWebPushYExpo(
		`SELECT id, endpoint, p256dh, auth FROM push_subscripciones WHERE guarderia_id = $1 AND personal_id IS NOT NULL`,
		`SELECT id, token FROM push_tokens_expo WHERE guarderia_id = $1 AND personal_id IS NOT NULL`,
		titulo, cuerpo, "notificarStaffDeGuarderia", guarderiaID,
	)
}

// notificarStaffEspecifico avisa a UN miembro del staff (no a toda la
// guardería, a diferencia de notificarStaffDeGuarderia) -- para cuando un
// papá le escribe a alguien en concreto en el chat. No se manda el
// contenido del mensaje, mismo criterio que notificarMensajeChat.
func (s *Server) notificarStaffEspecifico(personalID any, titulo, cuerpo string) {
	s.enviarWebPushYExpo(
		`SELECT id, endpoint, p256dh, auth FROM push_subscripciones WHERE personal_id = $1`,
		`SELECT id, token FROM push_tokens_expo WHERE personal_id = $1`,
		titulo, cuerpo, "notificarStaffEspecifico", personalID,
	)
}

// enviarWebPushYExpo es el punto único que usan las 5 funciones
// notificar* de arriba: junta Web Push (push_subscripciones, PWA del
// navegador) y Expo (push_tokens_expo, app nativa de papás en
// GuarderiaBiometricMobile) para no repetir en cada una el mismo
// Query+Scan+Close dos veces. queryWebPush y queryExpo deben regresar,
// en ese orden, (id, endpoint, p256dh, auth) e (id, token)
// respectivamente, y aceptar los mismos args -- en los 5 casos de arriba
// el filtro es siempre uno solo (hijo_id/guarderia_id/padre_id/
// personal_id), así que esto no pierde flexibilidad real.
//
// Web Push se salta por completo si el servidor no tiene llaves VAPID
// (PushConfigurado()); Expo no necesita configuración del lado del
// servidor, así que ese envío nunca se salta -- un papá puede tener
// activado un canal, el otro, los dos, o ninguno (si no hay suscripción/
// token en la tabla correspondiente, simplemente no hay destinos y no
// pasa nada).
func (s *Server) enviarWebPushYExpo(queryWebPush, queryExpo, titulo, cuerpo, origen string, args ...any) {
	if s.PushConfigurado() {
		rows, err := s.DB.Query(queryWebPush, args...)
		if err != nil {
			s.logError(nil, origen+": error consultando suscripciones", err)
		} else {
			var destinos []destinoPush
			for rows.Next() {
				var d destinoPush
				if err := rows.Scan(&d.id, &d.sub.Endpoint, &d.sub.Keys.P256dh, &d.sub.Keys.Auth); err != nil {
					continue
				}
				destinos = append(destinos, d)
			}
			rows.Close()

			if len(destinos) > 0 {
				if payload, err := json.Marshal(pushPayload{Titulo: titulo, Cuerpo: cuerpo, URL: "/"}); err != nil {
					s.logError(nil, origen+": error serializando payload", err)
				} else {
					s.enviarPushATodos(destinos, payload, "push_subscripciones")
				}
			}
		}
	}

	rowsExpo, err := s.DB.Query(queryExpo, args...)
	if err != nil {
		s.logError(nil, origen+": error consultando tokens Expo", err)
		return
	}
	var destinosExpo []destinoExpo
	for rowsExpo.Next() {
		var d destinoExpo
		if err := rowsExpo.Scan(&d.id, &d.token); err != nil {
			continue
		}
		destinosExpo = append(destinosExpo, d)
	}
	rowsExpo.Close()
	s.enviarPushExpoATodos(destinosExpo, titulo, cuerpo)
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
// notificarCircular. tabla es SIEMPRE un literal fijo que decide el
// caller (nunca dato de entrada) -- "push_subscripciones" para
// papás/staff de una guardería, "push_suscripciones_plataforma" para el
// dueño de la plataforma (ver push_plataforma.go); hace falta porque cada
// una vive en su propia tabla con sus propios ids autonuméricos, así que
// no basta con un DELETE ... WHERE id = $1 fijo a una sola tabla -- borraría
// (o fallaría en borrar) la fila equivocada según de cuál suscripción se
// trate.
func (s *Server) enviarPushATodos(destinos []destinoPush, payload []byte, tabla string) {
	for _, d := range destinos {
		resp, err := webpush.SendNotification(payload, &d.sub, &webpush.Options{
			VAPIDPublicKey:  s.VapidPublicKey,
			VAPIDPrivateKey: s.VapidPrivateKey,
			Subscriber:      s.VapidSubject,
			TTL:             30,
		})
		if err != nil {
			s.logError(nil, "enviarPushATodos: error enviando a suscripción", err, "suscripcion_id", d.id)
			continue
		}
		resp.Body.Close()

		// 404/410: la suscripción ya no existe del lado del navegador. La limpiamos.
		if resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound {
			s.DB.Exec("DELETE FROM "+tabla+" WHERE id = $1", d.id)
		}
	}
}
