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
// (familia, miembro del staff) -- "quiero que al papá le aparezcan los
// staff o administradores... para escoger con quién hablar", así que ya no
// es "una por familia" como antes: un mismo papá puede tener varias
// conversaciones abiertas, una por cada persona del staff que eligió.
// PersonalNombre solo viene lleno cuando quien pide la lista es admin (ve
// las de todos, así que necesita saber de quién es cada una) -- un staff
// normal ya sabe que todas las suyas son con él mismo.
type ConversacionResumen struct {
	PadreID        int    `json:"padre_id"`
	Nombre         string `json:"nombre"`
	PersonalID     int    `json:"personal_id"`
	PersonalNombre string `json:"personal_nombre,omitempty"`
	UltimoMensaje  string `json:"ultimo_mensaje"`
	UltimoEn       string `json:"ultimo_en"`
	NoLeidos       int    `json:"no_leidos"`
}

// ContactoChat es una fila del selector "con quién quieres hablar" del
// portal del papá -- el staff/admin activo de su guardería.
type ContactoChat struct {
	ID     int    `json:"id"`
	Nombre string `json:"nombre"`
	Rol    string `json:"rol"`
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
	r.GET("/chat/:padreId/:personalId/mensajes", auth, staff, s.handleObtenerMensajesStaff)
	r.POST("/chat/:padreId/:personalId/mensajes", auth, staff, s.handleEnviarMensajeStaff)

	r.GET("/padre/chat/contactos", auth, s.handleListarContactosChatPadre)
	r.GET("/padre/chat/:personalId", auth, s.handleObtenerMensajesPadre)
	r.POST("/padre/chat/:personalId", auth, s.handleEnviarMensajePadre)
}

func (s *Server) padrePerteneceAGuarderia(padreID string, gID any) bool {
	var existe bool
	err := s.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM padres WHERE id = $1 AND guarderia_id = $2)`, padreID, gID).Scan(&existe)
	return err == nil && existe
}

// handleListarContactosChatPadre -- "quiero que al papá le aparezcan los
// staff o administradores de la guardería... para escoger con quién
// hablar": el directorio que arma el selector antes de entrar a una
// conversación.
func (s *Server) handleListarContactosChatPadre(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")

	rows, err := s.DBAuth.Query(`
        SELECT id, COALESCE(nombre, username), rol
        FROM usuarios
        WHERE guarderia_id = $1 AND rol IN ('admin', 'staff') AND activo = true
        ORDER BY (rol = 'admin') DESC, COALESCE(nombre, username) ASC`,
		gID,
	)
	if err != nil {
		log.Printf("Error al consultar el staff para el chat: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar el staff"})
		return
	}
	defer rows.Close()

	contactos := []ContactoChat{}
	for rows.Next() {
		var ct ContactoChat
		if err := rows.Scan(&ct.ID, &ct.Nombre, &ct.Rol); err != nil {
			continue
		}
		contactos = append(contactos, ct)
	}
	c.JSON(http.StatusOK, contactos)
}

// handleListarConversaciones regresa una fila por (familia, staff) que ya
// tiene al menos un mensaje, con el último mensaje y cuántos siguen sin
// leer -- ordenado por actividad más reciente primero. Staff normal solo
// ve las conversaciones dirigidas a él mismo; admin ve todas (supervisión).
func (s *Server) handleListarConversaciones(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	rol, _ := c.Get("rol")
	userID, _ := c.Get("user_id")
	esAdmin := rol == "admin"

	rows, err := s.DB.Query(`
        SELECT DISTINCT ON (m.padre_id, m.personal_id)
            m.padre_id, COALESCE(pa.nombre, 'Familia'), m.personal_id, m.contenido, m.creado_en
        FROM mensajes_chat m
        LEFT JOIN padres pa ON pa.id = m.padre_id
        WHERE m.guarderia_id = $1 AND m.personal_id IS NOT NULL AND ($2 OR m.personal_id = $3)
        ORDER BY m.padre_id, m.personal_id, m.creado_en DESC`,
		gID, esAdmin, userID,
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
		if err := rows.Scan(&conv.PadreID, &conv.Nombre, &conv.PersonalID, &conv.UltimoMensaje, &creadoEn); err != nil {
			continue
		}
		if creadoEn != nil {
			conv.UltimoEn = *creadoEn
		}
		conversaciones = append(conversaciones, conv)
	}
	rows.Close()

	type parClave struct{ padreID, personalID int }
	noLeidos := map[parClave]int{}
	filas, err := s.DB.Query(`
        SELECT padre_id, personal_id, COUNT(*) FROM mensajes_chat
        WHERE guarderia_id = $1 AND autor_rol = 'papa' AND NOT leido AND personal_id IS NOT NULL
          AND ($2 OR personal_id = $3)
        GROUP BY padre_id, personal_id`,
		gID, esAdmin, userID,
	)
	if err == nil {
		for filas.Next() {
			var padreID, personalID, n int
			if err := filas.Scan(&padreID, &personalID, &n); err == nil {
				noLeidos[parClave{padreID, personalID}] = n
			}
		}
		filas.Close()
	}

	// Nombre de cada miembro del staff -- solo hace falta resolverlo para
	// admin (ve conversaciones de todos, necesita saber de quién es cada
	// una); un staff normal ya sabe que todas las suyas son con él mismo.
	// Consulta aparte porque usuarios vive en DBAuth, no en DB.
	if esAdmin && len(conversaciones) > 0 {
		nombresPersonal := map[int]string{}
		filasPersonal, err := s.DBAuth.Query(`SELECT id, COALESCE(nombre, username) FROM usuarios WHERE guarderia_id = $1`, gID)
		if err == nil {
			for filasPersonal.Next() {
				var id int
				var nombre string
				if err := filasPersonal.Scan(&id, &nombre); err == nil {
					nombresPersonal[id] = nombre
				}
			}
			filasPersonal.Close()
			for i := range conversaciones {
				conversaciones[i].PersonalNombre = nombresPersonal[conversaciones[i].PersonalID]
			}
		}
	}

	for i := range conversaciones {
		conversaciones[i].NoLeidos = noLeidos[parClave{conversaciones[i].PadreID, conversaciones[i].PersonalID}]
	}
	// DISTINCT ON exige que el ORDER BY empiece por esas columnas, así que el
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

// puedeAccederAlHiloDeStaff decide si quien hace la petición (rol/userID del
// token) puede ver/escribir en el hilo dirigido a personalID -- solo ese
// mismo miembro del staff, o cualquier admin (supervisión).
func puedeAccederAlHiloDeStaff(rol string, userID any, personalID string) bool {
	return rol == "admin" || fmt.Sprintf("%v", userID) == personalID
}

func (s *Server) handleObtenerMensajesStaff(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	rol, _ := c.Get("rol")
	userID, _ := c.Get("user_id")
	padreID := c.Param("padreId")
	personalID := c.Param("personalId")

	if !puedeAccederAlHiloDeStaff(fmt.Sprintf("%v", rol), userID, personalID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "No puedes ver conversaciones de otro miembro del staff"})
		return
	}
	if !s.padrePerteneceAGuarderia(padreID, gID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Familia no encontrada"})
		return
	}

	mensajes, err := s.obtenerHiloChat(gID, padreID, personalID)
	if err != nil {
		log.Printf("Error al consultar los mensajes: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar los mensajes"})
		return
	}
	for i := range mensajes {
		mensajes[i].EsMio = mensajes[i].AutorRol != "papa"
	}

	s.DB.Exec(`UPDATE mensajes_chat SET leido = true WHERE guarderia_id = $1 AND padre_id = $2 AND personal_id = $3 AND autor_rol = 'papa' AND NOT leido`, gID, padreID, personalID)

	c.JSON(http.StatusOK, mensajes)
}

func (s *Server) handleEnviarMensajeStaff(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	userID, _ := c.Get("user_id")
	rol, _ := c.Get("rol")
	padreID := c.Param("padreId")
	personalID := c.Param("personalId")

	if !puedeAccederAlHiloDeStaff(fmt.Sprintf("%v", rol), userID, personalID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "No puedes escribir en conversaciones de otro miembro del staff"})
		return
	}
	if !s.padrePerteneceAGuarderia(padreID, gID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Familia no encontrada"})
		return
	}

	msj, ok := s.leerMensajeConAdjunto(c, gID, fmt.Sprintf("padre_%v", padreID))
	if !ok {
		return
	}

	// personal_id queda con el DUEÑO del hilo (el de la URL), no
	// necesariamente con quien escribe -- así, si un admin responde dentro
	// de la conversación de un staff (cubriéndolo), el mensaje se queda en
	// ESE hilo en vez de crear uno nuevo a nombre del admin.
	if _, err := s.DB.Exec(
		`INSERT INTO mensajes_chat (guarderia_id, padre_id, personal_id, autor_id, autor_rol, contenido, adjunto_s3_key, adjunto_nombre, adjunto_tipo)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		gID, padreID, personalID, userID, rol, msj.contenido, msj.s3Key, msj.nombreArchivo, msj.tipoAdjunto,
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
	personalID := c.Param("personalId")

	if !s.personalPerteneceAGuarderia(personalID, gID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ese contacto no existe en tu guardería"})
		return
	}

	mensajes, err := s.obtenerHiloChat(gID, userID, personalID)
	if err != nil {
		log.Printf("Error al consultar los mensajes: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar los mensajes"})
		return
	}
	for i := range mensajes {
		mensajes[i].EsMio = mensajes[i].AutorRol == "papa"
	}

	s.DB.Exec(`UPDATE mensajes_chat SET leido = true WHERE guarderia_id = $1 AND padre_id = $2 AND personal_id = $3 AND autor_rol != 'papa' AND NOT leido`, gID, userID, personalID)

	c.JSON(http.StatusOK, mensajes)
}

func (s *Server) handleEnviarMensajePadre(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	userID, _ := c.Get("user_id")
	personalID := c.Param("personalId")

	if !s.personalPerteneceAGuarderia(personalID, gID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ese contacto no existe en tu guardería"})
		return
	}

	msj, ok := s.leerMensajeConAdjunto(c, gID, fmt.Sprintf("padre_%v", userID))
	if !ok {
		return
	}

	if _, err := s.DB.Exec(
		`INSERT INTO mensajes_chat (guarderia_id, padre_id, personal_id, autor_id, autor_rol, contenido, adjunto_s3_key, adjunto_nombre, adjunto_tipo)
         VALUES ($1, $2, $3, $4, 'papa', $5, $6, $7, $8)`,
		gID, userID, personalID, userID, msj.contenido, msj.s3Key, msj.nombreArchivo, msj.tipoAdjunto,
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

func (s *Server) obtenerHiloChat(gID, padreID, personalID any) ([]MensajeChat, error) {
	rows, err := s.DB.Query(
		`SELECT id, autor_rol, contenido, creado_en, adjunto_s3_key, adjunto_nombre, adjunto_tipo FROM mensajes_chat
         WHERE guarderia_id = $1 AND padre_id = $2 AND personal_id = $3
         ORDER BY creado_en ASC`,
		gID, padreID, personalID,
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
