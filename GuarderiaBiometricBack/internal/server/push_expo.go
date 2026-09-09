package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"biometrico/internal/middleware"
)

// Notificaciones push para GuarderiaBiometricMobile (la app nativa de
// papás en React Native/Expo) -- distintas de push.go, que es Web Push/
// VAPID para la PWA del navegador. Expo tiene su propio servicio de push
// (https://exp.host/--/api/v2/push/send) que enruta a FCM/APNs por debajo,
// así que aquí no hacen falta llaves VAPID ni PushConfigurado(): basta con
// el "Expo push token" que la app obtiene al arrancar y manda a
// /push/expo/registrar.
const urlPushExpo = "https://exp.host/--/api/v2/push/send"

func (s *Server) registrarRutasPushExpo(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)

	// --- GUARDAR TOKEN (cualquier usuario autenticado, típicamente "papa") ---
	r.POST("/push/expo/registrar", auth, func(c *gin.Context) {
		gID, _ := c.Get("guarderia_id")
		userID, _ := c.Get("user_id")
		rol, _ := c.Get("rol")

		var input struct {
			Token string `json:"token"`
		}
		if err := c.ShouldBindJSON(&input); err != nil || input.Token == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Token inválido"})
			return
		}

		// Mismo criterio que POST /push/suscribir en push.go: cada fila trae
		// UNO de los dos ids, según el rol de quien se registra.
		var padreID, personalID any
		if rol == "papa" {
			padreID = userID
		} else {
			personalID = userID
		}

		query := `
        INSERT INTO push_tokens_expo (padre_id, personal_id, guarderia_id, token)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (token) DO UPDATE SET
            padre_id = EXCLUDED.padre_id,
            personal_id = EXCLUDED.personal_id,
            guarderia_id = EXCLUDED.guarderia_id`

		if _, err := s.DB.Exec(query, padreID, personalID, gID, input.Token); err != nil {
			s.logError(c, "No se pudo guardar el token push de Expo", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo guardar el token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "Token guardado"})
	})

	// --- ELIMINAR TOKEN (al cerrar sesión o desactivar notificaciones) ---
	r.DELETE("/push/expo/registrar", auth, func(c *gin.Context) {
		var input struct {
			Token string `json:"token"`
		}
		if err := c.ShouldBindJSON(&input); err != nil || input.Token == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "token requerido"})
			return
		}

		s.DB.Exec("DELETE FROM push_tokens_expo WHERE token = $1", input.Token)
		c.JSON(http.StatusOK, gin.H{"status": "Token eliminado"})
	})
}

// destinoExpo es un token push de Expo resuelto, listo para mandarle una
// notificación -- equivalente de destinoPush (push.go) para la app nativa.
type destinoExpo struct {
	id    int
	token string
}

type mensajeExpo struct {
	To    string            `json:"to"`
	Title string            `json:"title"`
	Body  string            `json:"body"`
	Sound string            `json:"sound,omitempty"`
	Data  map[string]string `json:"data,omitempty"`
}

type respuestaExpoItem struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Details struct {
		Error string `json:"error,omitempty"`
	} `json:"details,omitempty"`
}

type respuestaExpo struct {
	Data []respuestaExpoItem `json:"data"`
}

// enviarPushExpoATodos manda el mismo título/cuerpo a cada token, en lotes
// de 100 (límite de la API de Expo por petición), y limpia de la BD
// cualquier token que Expo ya reporte como "DeviceNotRegistered" (el
// usuario desinstaló la app o el sistema operativo invalidó el token) --
// mismo criterio de limpieza que enviarPushATodos en push.go para 404/410.
func (s *Server) enviarPushExpoATodos(destinos []destinoExpo, titulo, cuerpo string) {
	if len(destinos) == 0 {
		return
	}

	cliente := &http.Client{Timeout: 10 * time.Second}

	for inicio := 0; inicio < len(destinos); inicio += 100 {
		fin := inicio + 100
		if fin > len(destinos) {
			fin = len(destinos)
		}
		lote := destinos[inicio:fin]

		mensajes := make([]mensajeExpo, len(lote))
		for i, d := range lote {
			mensajes[i] = mensajeExpo{To: d.token, Title: titulo, Body: cuerpo, Sound: "default"}
		}

		cuerpoJSON, err := json.Marshal(mensajes)
		if err != nil {
			s.logError(nil, "enviarPushExpoATodos: error serializando el lote", err)
			continue
		}

		req, err := http.NewRequest(http.MethodPost, urlPushExpo, bytes.NewReader(cuerpoJSON))
		if err != nil {
			s.logError(nil, "enviarPushExpoATodos: error creando la petición", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := cliente.Do(req)
		if err != nil {
			s.logError(nil, "enviarPushExpoATodos: error llamando a la API de Expo", err)
			continue
		}

		var respuesta respuestaExpo
		if err := json.NewDecoder(resp.Body).Decode(&respuesta); err != nil {
			s.logError(nil, "enviarPushExpoATodos: error leyendo la respuesta de Expo", err)
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		// La respuesta trae un elemento por cada mensaje del lote, EN EL
		// MISMO ORDEN -- así se relaciona cada resultado con su destino sin
		// que Expo tenga que regresar el token de vuelta.
		for i, item := range respuesta.Data {
			if i >= len(lote) {
				break
			}
			if item.Status == "error" && item.Details.Error == "DeviceNotRegistered" {
				s.DB.Exec("DELETE FROM push_tokens_expo WHERE id = $1", lote[i].id)
			}
		}
	}
}
