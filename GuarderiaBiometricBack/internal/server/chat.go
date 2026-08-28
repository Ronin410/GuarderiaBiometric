package server

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"biometrico/internal/middleware"
)

// maxTamanoAdjuntoChat -- mismo límite que documentos.go (maxTamanoDocumento):
// no hay razón para que una foto o PDF mandado por chat pese más que uno de
// inscripción.
const maxTamanoAdjuntoChat = maxTamanoDocumento

// ConversacionResumen es una fila del inbox de staff: una conversación por
// familia (no hay asignación de un maestro específico por niño en el modelo
// actual, así que cualquier staff/admin puede leer y responder a cualquiera).
type ConversacionResumen struct {
	PadreID       int    `json:"padre_id"`
	Nombre        string `json:"nombre"`
	UltimoMensaje string `json:"ultimo_mensaje"`
	UltimoEn      string `json:"ultimo_en"`
	NoLeidos      int    `json:"no_leidos"`
}

// MensajeChat es un mensaje del hilo. EsMio se calcula del lado del backend
// según el rol de quien pide la conversación (papá ve sus propios mensajes
// como "míos"; staff ve los suyos como "míos") para que el frontend solo
// tenga que alinear burbujas sin comparar ids/roles. Los campos de adjunto
// van en puntero porque la mayoría de los mensajes no traen uno -- si
// AdjuntoURL es nil, el frontend no muestra nada de adjunto.
type MensajeChat struct {
	ID            int     `json:"id"`
	AutorRol      string  `json:"autor_rol"`
	Contenido     string  `json:"contenido"`
	CreadoEn      string  `json:"creado_en"`
	EsMio         bool    `json:"es_mio"`
	AdjuntoURL    *string `json:"adjunto_url,omitempty"`
	AdjuntoNombre *string `json:"adjunto_nombre,omitempty"`
	AdjuntoTipo   *string `json:"adjunto_tipo,omitempty"` // "imagen" | "archivo"
}

func (s *Server) registrarRutasChat(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)
	staff := middleware.RequireStaff()

	r.GET("/chat/conversaciones", auth, staff, s.handleListarConversaciones)
	r.GET("/chat/:padreId/mensajes", auth, staff, s.handleObtenerMensajesStaff)
	r.POST("/chat/:padreId/mensajes", auth, staff, s.handleEnviarMensajeStaff)

	r.GET("/padre/chat", auth, s.handleObtenerMensajesPadre)
	r.POST("/padre/chat", auth, s.handleEnviarMensajePadre)
}

func (s *Server) padrePerteneceAGuarderia(padreID string, gID any) bool {
	var existe bool
	err := s.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM padres WHERE id = $1 AND guarderia_id = $2)`, padreID, gID).Scan(&existe)
	return err == nil && existe
}

// handleListarConversaciones regresa una fila por familia que ya escribió al
// menos un mensaje, con el último mensaje y cuántos de esa familia siguen
// sin leer -- ordenado por actividad más reciente primero.
func (s *Server) handleListarConversaciones(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")

	rows, err := s.DB.Query(`
        SELECT DISTINCT ON (m.padre_id)
            m.padre_id, COALESCE(pa.nombre, 'Familia'), m.contenido, m.creado_en
        FROM mensajes_chat m
        LEFT JOIN padres pa ON pa.id = m.padre_id
        WHERE m.guarderia_id = $1
        ORDER BY m.padre_id, m.creado_en DESC`,
		gID,
	)
	if err != nil {
		log.Printf("Error al consultar conversaciones: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar conversaciones"})
		return
	}
	conversaciones := []ConversacionResumen{}
	for rows.Next() {
		var conv ConversacionResumen
		var creadoEn *string
		if err := rows.Scan(&conv.PadreID, &conv.Nombre, &conv.UltimoMensaje, &creadoEn); err != nil {
			continue
		}
		if creadoEn != nil {
			conv.UltimoEn = *creadoEn
		}
		conversaciones = append(conversaciones, conv)
	}
	rows.Close()

	noLeidos := map[int]int{}
	filas, err := s.DB.Query(`
        SELECT padre_id, COUNT(*) FROM mensajes_chat
        WHERE guarderia_id = $1 AND autor_rol = 'papa' AND NOT leido
        GROUP BY padre_id`,
		gID,
	)
	if err == nil {
		for filas.Next() {
			var padreID, n int
			if err := filas.Scan(&padreID, &n); err == nil {
				noLeidos[padreID] = n
			}
		}
		filas.Close()
	}

	for i := range conversaciones {
		conversaciones[i].NoLeidos = noLeidos[conversaciones[i].PadreID]
	}
	// DISTINCT ON exige que el ORDER BY empiece por padre_id, así que el
	// orden final por actividad reciente se resuelve aquí en vez de en SQL.
	sortConversacionesPorRecencia(conversaciones)

	c.JSON(http.StatusOK, conversaciones)
}

func sortConversacionesPorRecencia(conversaciones []ConversacionResumen) {
	for i := 1; i < len(conversaciones); i++ {
		j := i
		for j > 0 && conversaciones[j-1].UltimoEn < conversaciones[j].UltimoEn {
			conversaciones[j-1], conversaciones[j] = conversaciones[j], conversaciones[j-1]
			j--
		}
	}
}

func (s *Server) handleObtenerMensajesStaff(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	padreID := c.Param("padreId")

	if !s.padrePerteneceAGuarderia(padreID, gID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Familia no encontrada"})
		return
	}

	mensajes, err := s.obtenerHiloChat(gID, padreID)
	if err != nil {
		log.Printf("Error al consultar los mensajes: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar los mensajes"})
		return
	}
	for i := range mensajes {
		mensajes[i].EsMio = mensajes[i].AutorRol != "papa"
	}

	s.DB.Exec(`UPDATE mensajes_chat SET leido = true WHERE guarderia_id = $1 AND padre_id = $2 AND autor_rol = 'papa' AND NOT leido`, gID, padreID)

	c.JSON(http.StatusOK, mensajes)
}

func (s *Server) handleEnviarMensajeStaff(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	userID, _ := c.Get("user_id")
	rol, _ := c.Get("rol")
	padreID := c.Param("padreId")

	if !s.padrePerteneceAGuarderia(padreID, gID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Familia no encontrada"})
		return
	}

	msj, ok := s.leerMensajeConAdjunto(c, gID, fmt.Sprintf("padre_%v", padreID))
	if !ok {
		return
	}

	if _, err := s.DB.Exec(
		`INSERT INTO mensajes_chat (guarderia_id, padre_id, autor_id, autor_rol, contenido, adjunto_s3_key, adjunto_nombre, adjunto_tipo)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		gID, padreID, userID, rol, msj.contenido, msj.s3Key, msj.nombreArchivo, msj.tipoAdjunto,
	); err != nil {
		if msj.s3Key != nil {
			go s.borrarDeS3(*msj.s3Key) // el mensaje no se guardó, no dejamos el adjunto huérfano
		}
		log.Printf("No se pudo enviar el mensaje de chat (guardería %v, padre %v): %v", gID, padreID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo enviar el mensaje"})
		return
	}

	if padreIDInt, err := strconv.Atoi(padreID); err == nil {
		go s.notificarMensajeChat(padreIDInt)
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Mensaje enviado"})
}

func (s *Server) handleObtenerMensajesPadre(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	userID, _ := c.Get("user_id")

	mensajes, err := s.obtenerHiloChat(gID, userID)
	if err != nil {
		log.Printf("Error al consultar los mensajes: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar los mensajes"})
		return
	}
	for i := range mensajes {
		mensajes[i].EsMio = mensajes[i].AutorRol == "papa"
	}

	s.DB.Exec(`UPDATE mensajes_chat SET leido = true WHERE guarderia_id = $1 AND padre_id = $2 AND autor_rol != 'papa' AND NOT leido`, gID, userID)

	c.JSON(http.StatusOK, mensajes)
}

func (s *Server) handleEnviarMensajePadre(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	userID, _ := c.Get("user_id")

	msj, ok := s.leerMensajeConAdjunto(c, gID, fmt.Sprintf("padre_%v", userID))
	if !ok {
		return
	}

	if _, err := s.DB.Exec(
		`INSERT INTO mensajes_chat (guarderia_id, padre_id, autor_id, autor_rol, contenido, adjunto_s3_key, adjunto_nombre, adjunto_tipo)
         VALUES ($1, $2, $3, 'papa', $4, $5, $6, $7)`,
		gID, userID, userID, msj.contenido, msj.s3Key, msj.nombreArchivo, msj.tipoAdjunto,
	); err != nil {
		if msj.s3Key != nil {
			go s.borrarDeS3(*msj.s3Key)
		}
		log.Printf("No se pudo enviar el mensaje de chat (guardería %v, padre %v): %v", gID, userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo enviar el mensaje"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Mensaje enviado"})
}

// mensajeConAdjunto es lo que ya se validó y (si traía archivo) ya se subió
// a S3, listo para insertarse -- s3Key/nombreArchivo/tipoAdjunto van en
// puntero porque la mayoría de los mensajes no traen adjunto (se insertan
// como NULL).
type mensajeConAdjunto struct {
	contenido     string
	s3Key         *string
	nombreArchivo *string
	tipoAdjunto   *string
}

// leerMensajeConAdjunto centraliza lo que comparten handleEnviarMensajeStaff
// y handleEnviarMensajePadre: el mensaje viaja como multipart/form-data
// (antes era JSON) para poder traer un archivo opcional junto con el texto
// -- un mensaje válido trae AL MENOS uno de los dos (texto o adjunto), igual
// que WhatsApp permite mandar una foto sin caption. rutaKey identifica la
// conversación dentro de la key de S3 (ya trae "padre_<id>" armado por el
// caller, porque el staff usa el padre_id de la URL y el papá su propio
// user_id).
func (s *Server) leerMensajeConAdjunto(c *gin.Context, gID any, rutaKey string) (mensajeConAdjunto, bool) {
	contenido := strings.TrimSpace(c.PostForm("contenido"))
	if len(contenido) > 2000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El mensaje es demasiado largo (máximo 2000 caracteres)"})
		return mensajeConAdjunto{}, false
	}

	fileHeader, errArchivo := c.FormFile("archivo")
	if errArchivo != nil {
		// Sin archivo: mismo criterio de antes, el mensaje necesita texto.
		if contenido == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "El mensaje no puede estar vacío"})
			return mensajeConAdjunto{}, false
		}
		return mensajeConAdjunto{contenido: contenido}, true
	}

	if fileHeader.Size > maxTamanoAdjuntoChat {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El archivo no puede pesar más de 10 MB"})
		return mensajeConAdjunto{}, false
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	tipoAdjunto := "archivo"
	if strings.HasPrefix(contentType, "image/") {
		tipoAdjunto = "imagen"
	}

	key := fmt.Sprintf("chat/guarderia_%v/%s/%d_%s", gID, rutaKey, time.Now().UnixNano(), fileHeader.Filename)
	if _, err := s.uploadToS3(fileHeader, key, contentType); err != nil {
		log.Printf("leerMensajeConAdjunto: fallo al subir el adjunto a S3 (%s): %v", key, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo subir el archivo"})
		return mensajeConAdjunto{}, false
	}

	nombreArchivo := fileHeader.Filename
	return mensajeConAdjunto{
		contenido:     contenido,
		s3Key:         &key,
		nombreArchivo: &nombreArchivo,
		tipoAdjunto:   &tipoAdjunto,
	}, true
}

func (s *Server) obtenerHiloChat(gID, padreID any) ([]MensajeChat, error) {
	rows, err := s.DB.Query(
		`SELECT id, autor_rol, contenido, creado_en, adjunto_s3_key, adjunto_nombre, adjunto_tipo FROM mensajes_chat
         WHERE guarderia_id = $1 AND padre_id = $2
         ORDER BY creado_en ASC`,
		gID, padreID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	mensajes := []MensajeChat{}
	for rows.Next() {
		var m MensajeChat
		var adjuntoKey, adjuntoNombre, adjuntoTipo *string
		if err := rows.Scan(&m.ID, &m.AutorRol, &m.Contenido, &m.CreadoEn, &adjuntoKey, &adjuntoNombre, &adjuntoTipo); err != nil {
			continue
		}
		if adjuntoKey != nil {
			if url, err := s.firmarURLFoto(*adjuntoKey); err == nil {
				m.AdjuntoURL = &url
				m.AdjuntoNombre = adjuntoNombre
				m.AdjuntoTipo = adjuntoTipo
			} else {
				log.Printf("obtenerHiloChat: no se pudo firmar la URL del adjunto %q (mensaje %d): %v", *adjuntoKey, m.ID, err)
			}
		}
		mensajes = append(mensajes, m)
	}
	return mensajes, nil
}
