package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/gin-gonic/gin"
)

var vapidPublicKey = os.Getenv("VAPID_PUBLIC_KEY")
var vapidPrivateKey = os.Getenv("VAPID_PRIVATE_KEY")
var vapidSubject = os.Getenv("VAPID_SUBJECT")

// pushConfigurado indica si el servidor tiene claves VAPID propias. Sin ellas,
// las notificaciones simplemente se omiten (no es un requisito para operar el
// resto de la app, a diferencia de JWT_SECRET).
func pushConfigurado() bool {
	return vapidPublicKey != "" && vapidPrivateKey != ""
}

type pushSubscriptionInput struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

func registrarRutasPush(r *gin.Engine) {
	// --- GUARDAR SUSCRIPCIÓN (cualquier usuario autenticado, típicamente rol "papa") ---
	r.POST("/push/suscribir", AuthMiddleware(), func(c *gin.Context) {
		if !pushConfigurado() {
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

		if _, err := db.Exec(query, userID, gID, input.Endpoint, input.Keys.P256dh, input.Keys.Auth); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo guardar la suscripción"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "Suscripción guardada"})
	})

	// --- ELIMINAR SUSCRIPCIÓN (cuando el usuario desactiva notificaciones) ---
	r.DELETE("/push/suscribir", AuthMiddleware(), func(c *gin.Context) {
		var input struct {
			Endpoint string `json:"endpoint"`
		}
		if err := c.ShouldBindJSON(&input); err != nil || input.Endpoint == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "endpoint requerido"})
			return
		}

		db.Exec("DELETE FROM push_subscripciones WHERE endpoint = $1", input.Endpoint)
		c.JSON(http.StatusOK, gin.H{"status": "Suscripción eliminada"})
	})
}

type pushPayload struct {
	Titulo string `json:"titulo"`
	Cuerpo string `json:"cuerpo"`
	URL    string `json:"url"`
}

// notificarEvento avisa a TODOS los tutores vinculados a un niño (no solo a
// quien disparó el movimiento). Debe llamarse siempre como "go notificarEvento(...)"
// desde el handler: nunca debe frenar la respuesta al kiosco/staff.
func notificarEvento(hijoID int, evento string, detalle string) {
	if !pushConfigurado() {
		return
	}

	var nombre string
	if err := db.QueryRow("SELECT nombre_niño FROM hijos WHERE id = $1", hijoID).Scan(&nombre); err != nil {
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
	default:
		titulo = "BioSafe"
		cuerpo = nombre + ": " + detalle
	}

	rows, err := db.Query(`
        SELECT DISTINCT ps.id, ps.endpoint, ps.p256dh, ps.auth
        FROM push_subscripciones ps
        INNER JOIN tutor_hijos th ON th.padre_id = ps.padre_id
        WHERE th.hijo_id = $1`, hijoID)
	if err != nil {
		log.Printf("notificarEvento: error consultando suscripciones: %v", err)
		return
	}

	type destino struct {
		id  int
		sub webpush.Subscription
	}
	var destinos []destino
	for rows.Next() {
		var d destino
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

	for _, d := range destinos {
		resp, err := webpush.SendNotification(payload, &d.sub, &webpush.Options{
			VAPIDPublicKey:  vapidPublicKey,
			VAPIDPrivateKey: vapidPrivateKey,
			Subscriber:      vapidSubject,
			TTL:             30,
		})
		if err != nil {
			log.Printf("notificarEvento: error enviando a suscripción %d: %v", d.id, err)
			continue
		}
		resp.Body.Close()

		// 404/410: la suscripción ya no existe del lado del navegador. La limpiamos.
		if resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound {
			db.Exec("DELETE FROM push_subscripciones WHERE id = $1", d.id)
		}
	}
}
