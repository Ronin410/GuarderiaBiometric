package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"biometrico/internal/middleware"
)

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
// tenga que alinear burbujas sin comparar ids/roles.
type MensajeChat struct {
	ID        int    `json:"id"`
	AutorRol  string `json:"autor_rol"`
	Contenido string `json:"contenido"`
	CreadoEn  string `json:"creado_en"`
	EsMio     bool   `json:"es_mio"`
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

	contenido, ok := leerContenidoMensaje(c)
	if !ok {
		return
	}

	if _, err := s.DB.Exec(
		`INSERT INTO mensajes_chat (guarderia_id, padre_id, autor_id, autor_rol, contenido) VALUES ($1, $2, $3, $4, $5)`,
		gID, padreID, userID, rol, contenido,
	); err != nil {
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

	contenido, ok := leerContenidoMensaje(c)
	if !ok {
		return
	}

	if _, err := s.DB.Exec(
		`INSERT INTO mensajes_chat (guarderia_id, padre_id, autor_id, autor_rol, contenido) VALUES ($1, $2, $3, 'papa', $4)`,
		gID, userID, userID, contenido,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo enviar el mensaje"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Mensaje enviado"})
}

// leerContenidoMensaje centraliza la validación que comparten
// handleEnviarMensajeStaff y handleEnviarMensajePadre: solo cambia quién es
// el autor y a qué conversación va, no qué hace válido un mensaje.
func leerContenidoMensaje(c *gin.Context) (string, bool) {
	var input struct {
		Contenido string `json:"contenido"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return "", false
	}
	contenido := strings.TrimSpace(input.Contenido)
	if contenido == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El mensaje no puede estar vacío"})
		return "", false
	}
	if len(contenido) > 2000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El mensaje es demasiado largo (máximo 2000 caracteres)"})
		return "", false
	}
	return contenido, true
}

func (s *Server) obtenerHiloChat(gID, padreID any) ([]MensajeChat, error) {
	rows, err := s.DB.Query(
		`SELECT id, autor_rol, contenido, creado_en FROM mensajes_chat
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
		if err := rows.Scan(&m.ID, &m.AutorRol, &m.Contenido, &m.CreadoEn); err != nil {
			continue
		}
		mensajes = append(mensajes, m)
	}
	return mensajes, nil
}
