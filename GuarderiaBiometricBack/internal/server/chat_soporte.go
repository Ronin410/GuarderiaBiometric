package server

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"biometrico/internal/middleware"
)

// chat_soporte.go -- chat de soporte con el dueño de la plataforma, DISTINTO
// del chat de chat.go (que es papás<->staff DENTRO de una misma guardería).
// Aquí escriben tres tipos de gente: papás y staff/admin ya con cuenta (de
// CUALQUIER guardería), y prospectos sin cuenta (visitantes de la página de
// presentación, "posibles nuevos clientes"). El dueño de la plataforma ve y
// responde todo desde /plataforma, protegido por PLATFORM_ADMIN_KEY -- igual
// que las solicitudes de alta y el panorama de guarderías.

const maxTamanoMensajeSoporte = 2000

// ConversacionSoporte es una fila del inbox de la plataforma.
// GuarderiaNombre solo viene lleno para tipo papa/staff (se resuelve aparte,
// ver resolverNombresGuarderiasSoporte); un prospecto no tiene guardería
// todavía.
type ConversacionSoporte struct {
	ID              int     `json:"id"`
	Tipo            string  `json:"tipo"`
	GuarderiaID     *int    `json:"guarderia_id,omitempty"`
	GuarderiaNombre string  `json:"guarderia_nombre,omitempty"`
	Nombre          string  `json:"nombre"`
	Email           *string `json:"email,omitempty"`
	Cerrada         bool    `json:"cerrada"`
	UltimoMensaje   string  `json:"ultimo_mensaje"`
	ActualizadoEn   string  `json:"actualizado_en"`
	NoLeidos        int     `json:"no_leidos"`
}

// MensajeSoporte es un mensaje del hilo. EsMio se calcula del lado del
// backend según quién pide la conversación -- mismo criterio que
// MensajeChat en chat.go.
type MensajeSoporte struct {
	ID        int    `json:"id"`
	AutorRol  string `json:"autor_rol"`
	Contenido string `json:"contenido"`
	CreadoEn  string `json:"creado_en"`
	EsMio     bool   `json:"es_mio"`
}

func (s *Server) registrarRutasSoporte(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)
	platform := middleware.RequirePlatformKey(s.PlatformAdminKey)

	// Prospecto: público a propósito -- gente que todavía no tiene cuenta
	// (visitantes de la página de presentación). Limitado por IP para que
	// el formulario no se pueda usar para mandar spam masivo.
	r.POST("/soporte/prospecto", s.soporteLimiter.Middleware(), s.handleCrearConversacionProspecto)
	r.GET("/soporte/prospecto/:token/mensajes", s.handleObtenerMensajesProspecto)
	r.POST("/soporte/prospecto/:token/mensajes", s.soporteLimiter.Middleware(), s.handleEnviarMensajeProspecto)

	// Papá / staff / admin ya autenticados -- una sola conversación
	// continua con la plataforma por cuenta (no un hilo nuevo cada vez).
	r.GET("/soporte/mis-mensajes", auth, s.handleObtenerMisMensajesSoporte)
	r.POST("/soporte/mis-mensajes", auth, s.handleEnviarMensajeSoporte)
	r.GET("/soporte/no-leidos", auth, s.handleContarNoLeidosSoporte)

	// Dueño de la plataforma -- ve y responde TODAS las conversaciones.
	r.GET("/plataforma/soporte/conversaciones", platform, s.handleListarConversacionesSoporte)
	r.GET("/plataforma/soporte/:id/mensajes", platform, s.handleObtenerMensajesSoportePlataforma)
	r.POST("/plataforma/soporte/:id/mensajes", platform, s.handleResponderSoportePlataforma)
	r.GET("/plataforma/soporte/no-leidos", platform, s.handleContarNoLeidosSoportePlataforma)
}

func generarTokenProspecto() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ---------- Prospecto (público, sin cuenta) ----------

func (s *Server) handleCrearConversacionProspecto(c *gin.Context) {
	var input struct {
		Nombre  string `json:"nombre"`
		Email   string `json:"email"`
		Mensaje string `json:"mensaje"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}
	input.Nombre = strings.TrimSpace(input.Nombre)
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	input.Mensaje = strings.TrimSpace(input.Mensaje)

	if input.Nombre == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cuéntanos tu nombre para poder ayudarte"})
		return
	}
	if !emailRegex.MatchString(input.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Correo inválido"})
		return
	}
	if input.Mensaje == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Escribe tu mensaje"})
		return
	}
	if len(input.Mensaje) > maxTamanoMensajeSoporte {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El mensaje es demasiado largo (máximo 2000 caracteres)"})
		return
	}

	token, err := generarTokenProspecto()
	if err != nil {
		s.logError(c, "No se pudo generar el token de la conversación de soporte", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo iniciar la conversación"})
		return
	}

	tx, err := s.DB.Begin()
	if err != nil {
		s.logError(c, "No se pudo iniciar la conversación de soporte", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo iniciar la conversación"})
		return
	}
	defer tx.Rollback()

	var convID int
	if err := tx.QueryRow(
		`INSERT INTO conversaciones_soporte (tipo, nombre, email, token) VALUES ('prospecto', $1, $2, $3) RETURNING id`,
		input.Nombre, input.Email, token,
	).Scan(&convID); err != nil {
		s.logError(c, "No se pudo crear la conversación de soporte (prospecto)", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo iniciar la conversación"})
		return
	}

	if _, err := tx.Exec(
		`INSERT INTO mensajes_soporte (conversacion_id, autor_rol, contenido) VALUES ($1, $2, $3)`,
		convID, "prospecto", input.Mensaje,
	); err != nil {
		s.logError(c, "No se pudo guardar el primer mensaje de soporte (prospecto)", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo iniciar la conversación"})
		return
	}

	if err := tx.Commit(); err != nil {
		s.logError(c, "No se pudo completar el alta de la conversación de soporte (prospecto)", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo iniciar la conversación"})
		return
	}

	go s.notificarPlataformaNuevoMensajeSoporte("Prospecto nuevo: " + input.Nombre)

	c.JSON(http.StatusCreated, gin.H{"token": token})
}

func (s *Server) conversacionProspectoPorToken(token string) (int, bool, error) {
	var id int
	err := s.DB.QueryRow(`SELECT id FROM conversaciones_soporte WHERE token = $1 AND tipo = 'prospecto'`, token).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

func (s *Server) handleObtenerMensajesProspecto(c *gin.Context) {
	token := c.Param("token")
	convID, existe, err := s.conversacionProspectoPorToken(token)
	if err != nil {
		s.logError(c, "Error al consultar la conversación de soporte (prospecto)", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar la conversación"})
		return
	}
	if !existe {
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversación no encontrada"})
		return
	}

	mensajes, err := s.obtenerHiloSoporte(convID, "prospecto")
	if err != nil {
		s.logError(c, "Error al consultar los mensajes de soporte (prospecto)", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar los mensajes"})
		return
	}

	s.DB.Exec(`UPDATE mensajes_soporte SET leido = true WHERE conversacion_id = $1 AND autor_rol = 'plataforma' AND NOT leido`, convID)

	c.JSON(http.StatusOK, mensajes)
}

func (s *Server) handleEnviarMensajeProspecto(c *gin.Context) {
	token := c.Param("token")
	convID, existe, err := s.conversacionProspectoPorToken(token)
	if err != nil {
		s.logError(c, "Error al consultar la conversación de soporte (prospecto)", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar la conversación"})
		return
	}
	if !existe {
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversación no encontrada"})
		return
	}

	contenido, ok := s.leerMensajeSoporte(c)
	if !ok {
		return
	}

	if err := s.insertarMensajeSoporte(convID, "prospecto", contenido); err != nil {
		s.logError(c, "No se pudo enviar el mensaje de soporte (prospecto)", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo enviar el mensaje"})
		return
	}

	go s.notificarPlataformaNuevoMensajeSoporteDeConversacion(convID, "Prospecto")

	c.JSON(http.StatusCreated, gin.H{"message": "Mensaje enviado"})
}

// ---------- Papá / staff / admin autenticados ----------

// buscarConversacionPropia busca (sin crear) la conversación de soporte de
// quien hace la petición -- para GET: abrir el widget de soporte sin haber
// escrito nunca no debe dejar una conversación vacía en el inbox de la
// plataforma.
func (s *Server) buscarConversacionPropia(gID, userID any, tipo string) (int, bool, error) {
	var convID int
	err := s.DB.QueryRow(
		`SELECT id FROM conversaciones_soporte WHERE tipo = $1 AND guarderia_id = $2 AND usuario_id = $3`,
		tipo, gID, userID,
	).Scan(&convID)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return convID, true, nil
}

// obtenerOCrearConversacionPropia regresa la conversación de soporte de
// quien hace la petición, creándola (con nombre resuelto) la primera vez
// que escribe.
func (s *Server) obtenerOCrearConversacionPropia(gID, userID any, rol string) (int, error) {
	tipo := "staff"
	if rol == "papa" {
		tipo = "papa"
	}

	if convID, existe, err := s.buscarConversacionPropia(gID, userID, tipo); err != nil {
		return 0, err
	} else if existe {
		return convID, nil
	}

	nombre := s.resolverNombreParaSoporte(gID, userID, tipo)
	var convID int
	err := s.DB.QueryRow(
		`INSERT INTO conversaciones_soporte (tipo, guarderia_id, usuario_id, nombre) VALUES ($1, $2, $3, $4) RETURNING id`,
		tipo, gID, userID, nombre,
	).Scan(&convID)
	return convID, err
}

// resolverNombreParaSoporte consulta el nombre de quien escribe -- padres
// vive en DB, usuarios en DBAuth (mismo cruce ya resuelto con dos queries
// separadas en chat.go). Si falla, se guarda un nombre genérico en vez de
// tumbar la creación de la conversación.
func (s *Server) resolverNombreParaSoporte(gID, userID any, tipo string) string {
	var nombre string
	var err error
	if tipo == "papa" {
		err = s.DB.QueryRow(`SELECT COALESCE(nombre, 'Familia') FROM padres WHERE id = $1 AND guarderia_id = $2`, userID, gID).Scan(&nombre)
	} else {
		err = s.DBAuth.QueryRow(`SELECT COALESCE(nombre, username) FROM usuarios WHERE id = $1 AND guarderia_id = $2`, userID, gID).Scan(&nombre)
	}
	if err != nil || nombre == "" {
		return "Sin nombre"
	}
	return nombre
}

func (s *Server) handleObtenerMisMensajesSoporte(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	userID, _ := c.Get("user_id")
	rol, _ := c.Get("rol")
	tipo := "staff"
	if fmtRol(rol) == "papa" {
		tipo = "papa"
	}

	convID, existe, err := s.buscarConversacionPropia(gID, userID, tipo)
	if err != nil {
		s.logError(c, "Error al obtener la conversación de soporte", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener la conversación"})
		return
	}
	if !existe {
		// Todavía no ha escrito nada -- hilo vacío, sin crear la fila.
		c.JSON(http.StatusOK, []MensajeSoporte{})
		return
	}

	mensajes, err := s.obtenerHiloSoporte(convID, "plataforma")
	if err != nil {
		s.logError(c, "Error al consultar los mensajes de soporte", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar los mensajes"})
		return
	}

	s.DB.Exec(`UPDATE mensajes_soporte SET leido = true WHERE conversacion_id = $1 AND autor_rol = 'plataforma' AND NOT leido`, convID)

	c.JSON(http.StatusOK, mensajes)
}

func (s *Server) handleEnviarMensajeSoporte(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	userID, _ := c.Get("user_id")
	rol, _ := c.Get("rol")

	convID, err := s.obtenerOCrearConversacionPropia(gID, userID, fmtRol(rol))
	if err != nil {
		s.logError(c, "Error al obtener la conversación de soporte", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener la conversación"})
		return
	}

	contenido, ok := s.leerMensajeSoporte(c)
	if !ok {
		return
	}

	if err := s.insertarMensajeSoporte(convID, fmtRol(rol), contenido); err != nil {
		s.logError(c, "No se pudo enviar el mensaje de soporte", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo enviar el mensaje"})
		return
	}

	etiquetaRol := "Staff/Admin"
	if fmtRol(rol) == "papa" {
		etiquetaRol = "Papá"
	}
	go s.notificarPlataformaNuevoMensajeSoporteDeConversacion(convID, etiquetaRol)

	c.JSON(http.StatusCreated, gin.H{"message": "Mensaje enviado"})
}

// handleContarNoLeidosSoporte -- badge del botón de soporte para un papá o
// staff/admin ya logueado: cuántas respuestas de la plataforma le faltan
// por ver en SU propia conversación.
func (s *Server) handleContarNoLeidosSoporte(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	userID, _ := c.Get("user_id")
	rol, _ := c.Get("rol")
	tipo := "staff"
	if fmtRol(rol) == "papa" {
		tipo = "papa"
	}

	var noLeidos int
	err := s.DB.QueryRow(`
        SELECT COUNT(*) FROM mensajes_soporte m
        JOIN conversaciones_soporte cs ON cs.id = m.conversacion_id
        WHERE cs.tipo = $1 AND cs.guarderia_id = $2 AND cs.usuario_id = $3
          AND m.autor_rol = 'plataforma' AND NOT m.leido`,
		tipo, gID, userID,
	).Scan(&noLeidos)
	if err != nil {
		s.logError(c, "Error al contar los mensajes de soporte sin leer", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al contar los mensajes sin leer"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"no_leidos": noLeidos})
}

// fmtRol normaliza el valor que trae el contexto de gin (any) a string --
// mismo patrón que puedeAccederAlHiloDeStaff en chat.go usa con fmt.Sprintf,
// pero rol siempre llega como string desde middleware.Auth así que un type
// assertion directo basta.
func fmtRol(rol any) string {
	r, _ := rol.(string)
	return r
}

// ---------- Dueño de la plataforma ----------

// handleListarConversacionesSoporte regresa el inbox completo, más
// reciente primero -- papás, staff/admin y prospectos mezclados, cada uno
// con su tipo para que el frontend los distinga.
func (s *Server) handleListarConversacionesSoporte(c *gin.Context) {
	rows, err := s.DB.Query(`
        SELECT cs.id, cs.tipo, cs.guarderia_id, cs.nombre, cs.email, cs.cerrada, cs.actualizado_en,
               COALESCE((SELECT contenido FROM mensajes_soporte WHERE conversacion_id = cs.id ORDER BY creado_en DESC LIMIT 1), ''),
               (SELECT COUNT(*) FROM mensajes_soporte WHERE conversacion_id = cs.id AND autor_rol != 'plataforma' AND NOT leido)
        FROM conversaciones_soporte cs
        ORDER BY cs.actualizado_en DESC`)
	if err != nil {
		s.logError(c, "Error al consultar las conversaciones de soporte", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar las conversaciones"})
		return
	}
	defer rows.Close()

	conversaciones := []ConversacionSoporte{}
	for rows.Next() {
		var conv ConversacionSoporte
		if err := rows.Scan(&conv.ID, &conv.Tipo, &conv.GuarderiaID, &conv.Nombre, &conv.Email, &conv.Cerrada,
			&conv.ActualizadoEn, &conv.UltimoMensaje, &conv.NoLeidos); err != nil {
			continue
		}
		conversaciones = append(conversaciones, conv)
	}

	s.resolverNombresGuarderiasSoporte(conversaciones)

	c.JSON(http.StatusOK, conversaciones)
}

// resolverNombresGuarderiasSoporte llena GuarderiaNombre en una sola query
// extra a DBAuth (donde vive guarderias) -- mismo patrón que
// handleListarConversaciones en chat.go usa para el nombre del staff.
func (s *Server) resolverNombresGuarderiasSoporte(conversaciones []ConversacionSoporte) {
	ids := map[int]bool{}
	for _, conv := range conversaciones {
		if conv.GuarderiaID != nil {
			ids[*conv.GuarderiaID] = true
		}
	}
	if len(ids) == 0 {
		return
	}

	nombres := map[int]string{}
	rows, err := s.DBAuth.Query(`SELECT id, nombre FROM guarderias`)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var nombre string
		if err := rows.Scan(&id, &nombre); err == nil {
			nombres[id] = nombre
		}
	}

	for i := range conversaciones {
		if conversaciones[i].GuarderiaID != nil {
			conversaciones[i].GuarderiaNombre = nombres[*conversaciones[i].GuarderiaID]
		}
	}
}

func (s *Server) handleObtenerMensajesSoportePlataforma(c *gin.Context) {
	convID := c.Param("id")

	// Un id inexistente simplemente regresa un hilo vacío -- no vale la
	// pena una consulta aparte solo para distinguirlo de una conversación
	// real todavía sin mensajes, ninguno de los dos rompe nada del lado
	// del frontend.
	mensajes, err := s.obtenerHiloSoporte(convID, "papa_staff_o_prospecto")
	if err != nil {
		s.logError(c, "Error al consultar los mensajes de soporte", err, "conversacion_id", convID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar los mensajes"})
		return
	}

	s.DB.Exec(`UPDATE mensajes_soporte SET leido = true WHERE conversacion_id = $1 AND autor_rol != 'plataforma' AND NOT leido`, convID)

	c.JSON(http.StatusOK, mensajes)
}

func (s *Server) handleResponderSoportePlataforma(c *gin.Context) {
	convID := c.Param("id")

	contenido, ok := s.leerMensajeSoporte(c)
	if !ok {
		return
	}

	var existe bool
	if err := s.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM conversaciones_soporte WHERE id = $1)`, convID).Scan(&existe); err != nil {
		s.logError(c, "Error al verificar la conversación de soporte", err, "conversacion_id", convID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al enviar la respuesta"})
		return
	}
	if !existe {
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversación no encontrada"})
		return
	}

	if err := s.insertarMensajeSoporte(convID, "plataforma", contenido); err != nil {
		s.logError(c, "No se pudo enviar la respuesta de soporte", err, "conversacion_id", convID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo enviar la respuesta"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Respuesta enviada"})
}

func (s *Server) handleContarNoLeidosSoportePlataforma(c *gin.Context) {
	var noLeidos int
	err := s.DB.QueryRow(`
        SELECT COUNT(DISTINCT conversacion_id) FROM mensajes_soporte
        WHERE autor_rol != 'plataforma' AND NOT leido`,
	).Scan(&noLeidos)
	if err != nil {
		s.logError(c, "Error al contar las conversaciones de soporte sin leer", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al contar las conversaciones sin leer"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"no_leidos": noLeidos})
}

// ---------- compartido ----------

// notificarPlataformaNuevoMensajeSoporte -- "quiero que las notificaciones
// de los chats sean notificaciones push... que lleguen al iniciar como mi
// cuenta de admin". Manda un push a las suscripciones del dueño de la
// plataforma (ver push_plataforma.go) con quién escribió, ya resuelto por
// el caller (ver notificarPlataformaNuevoMensajeSoporteDeConversacion para
// cuando hace falta resolver el nombre/guardería primero). No lleva el
// contenido del mensaje -- mismo criterio de privacidad que
// notificarMensajeChat/notificarStaffEspecifico en push.go: el papá/staff/
// prospecto que escribió no dio su consentimiento para que su mensaje viaje
// en el payload de una notificación del sistema operativo.
func (s *Server) notificarPlataformaNuevoMensajeSoporte(quien string) {
	s.notificarPlataformaPush("💬 Nuevo mensaje de soporte", quien+" te escribió. Responde desde /plataforma.")
}

// notificarPlataformaNuevoMensajeSoporteDeConversacion -- misma idea, pero
// para las respuestas de una conversación YA existente (papá/staff/admin
// autenticados, o un prospecto que sigue escribiendo): el caller solo tiene
// el conversacion_id, así que esto resuelve el nombre y, si aplica, la
// guardería ("quiero que los chats con administradores y los chats con los
// papás me digan a qué guardería pertenecen") guardados ahí. Pensada para
// llamarse SIEMPRE con "go" (ver los tres call sites en este archivo) -- la
// consulta a la base y el envío del push quedan fuera del camino de la
// respuesta HTTP, igual que notificarMensajeChat.
func (s *Server) notificarPlataformaNuevoMensajeSoporteDeConversacion(convID any, etiquetaRol string) {
	var nombre string
	var guarderiaID *int
	if err := s.DB.QueryRow(`SELECT nombre, guarderia_id FROM conversaciones_soporte WHERE id = $1`, convID).Scan(&nombre, &guarderiaID); err != nil {
		nombre = "alguien"
	}
	quien := etiquetaRol + ": " + nombre
	if guarderiaID != nil {
		if gNombre := s.nombreGuarderiaParaSoporte(*guarderiaID); gNombre != "" {
			quien += " (" + gNombre + ")"
		}
	}
	s.notificarPlataformaNuevoMensajeSoporte(quien)
}

// nombreGuarderiaParaSoporte resuelve el nombre de una guardería por id --
// guarderias vive en DBAuth (mismo cruce que ya usa
// resolverNombresGuarderiasSoporte para el listado completo; aquí es una
// sola fila porque esto corre para UNA conversación en la goroutine de
// notificación). Un error se traga en vez de propagarse: como en
// resolverNombreParaSoporte, esto solo alimenta el texto de una
// notificación, no debe tumbar nada.
func (s *Server) nombreGuarderiaParaSoporte(guarderiaID int) string {
	var nombre string
	if err := s.DBAuth.QueryRow(`SELECT nombre FROM guarderias WHERE id = $1`, guarderiaID).Scan(&nombre); err != nil {
		return ""
	}
	return nombre
}

// leerMensajeSoporte valida el cuerpo JSON {"contenido": "..."} que
// comparten los cinco endpoints de "enviar mensaje" de este archivo.
func (s *Server) leerMensajeSoporte(c *gin.Context) (string, bool) {
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
	if len(contenido) > maxTamanoMensajeSoporte {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El mensaje es demasiado largo (máximo 2000 caracteres)"})
		return "", false
	}
	return contenido, true
}

func (s *Server) insertarMensajeSoporte(convID any, autorRol, contenido string) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`INSERT INTO mensajes_soporte (conversacion_id, autor_rol, contenido) VALUES ($1, $2, $3)`,
		convID, autorRol, contenido,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE conversaciones_soporte SET actualizado_en = now() WHERE id = $1`, convID); err != nil {
		return err
	}
	return tx.Commit()
}

// obtenerHiloSoporte trae los mensajes de una conversación y marca EsMio
// según quién los está leyendo: "plataforma" ve como suyos los que ella
// escribió, cualquier otro lector (papá, staff, prospecto) ve como suyos
// los que NO son de la plataforma.
func (s *Server) obtenerHiloSoporte(convID any, lector string) ([]MensajeSoporte, error) {
	rows, err := s.DB.Query(
		`SELECT id, autor_rol, contenido, creado_en FROM mensajes_soporte WHERE conversacion_id = $1 ORDER BY creado_en ASC`,
		convID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	mensajes := []MensajeSoporte{}
	for rows.Next() {
		var m MensajeSoporte
		if err := rows.Scan(&m.ID, &m.AutorRol, &m.Contenido, &m.CreadoEn); err != nil {
			continue
		}
		if lector == "plataforma" {
			m.EsMio = m.AutorRol == "plataforma"
		} else {
			m.EsMio = m.AutorRol != "plataforma"
		}
		mensajes = append(mensajes, m)
	}
	return mensajes, nil
}
