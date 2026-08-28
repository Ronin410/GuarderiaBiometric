package server

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"biometrico/internal/middleware"
)

type VinculacionRequest struct {
	PadreID int `json:"padre_id"`
	HijoID  int `json:"hijo_id"`
}

func (s *Server) registrarRutasHijos(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)
	// Todas estas son operaciones del directorio de tutores (pestaña
	// "Familia" en el panel) -- salvo /padre/:id/hijos, que un padre
	// también llama para su propio expediente y por eso se queda solo con
	// auth, sin exigir el área. Antes de "permisos personalizados por
	// docente" ninguna de estas (salvo regenerar-token) pedía ni siquiera
	// RequireStaff: cualquier JWT válido, incluido uno de rol "papa", podía
	// llamarlas directo sin pasar por la UI. familia cierra ese hueco de
	// paso, no solo agrega la personalización.
	familia := middleware.RequireArea("familia")

	r.POST("/registrar-hijo", auth, familia, s.handleRegistrarHijo)
	r.GET("/padre/:id/hijos", auth, s.handleHijosDePadre)
	r.POST("/vincular-tutor", auth, familia, s.handleVincularTutor)
	r.GET("/buscar-hijos", auth, familia, s.handleBuscarHijos)
	r.GET("/buscar-padres", auth, familia, s.handleBuscarPadres)
	r.POST("/desvincular-hijo", auth, familia, s.handleDesvincularHijo)
	r.POST("/actualizar-padre", auth, familia, s.handleActualizarPadre)
	r.PATCH("/hijos/:id/desactivar", auth, familia, s.handleDesactivarHijo)
	r.PUT("/hijos/:id", auth, familia, s.handleEditarNombreHijo)
	r.PATCH("/hijos/:id/activar", auth, familia, s.handleActivarHijo)
	// El enlace público de bitácora es permanente (el papá lo revisita cada día), así
	// que no expira solo por tiempo. Esto le da al staff una forma de invalidar un
	// link comprometido/reenviado de más al instante.
	r.POST("/hijos/:id/regenerar-token", auth, familia, s.handleRegenerarToken)
	// Para un tutor cuyo rostro ya se registró (en el kiosco, sin marcar
	// "crear cuenta" en su momento) -- da de alta el acceso al portal
	// después, sin volver a pasar por Rekognition.
	r.POST("/padres/:id/crear-cuenta", auth, familia, s.handleCrearCuentaPadre)
	// Para cuando "Crear cuenta" choca porque el tutor YA tiene una (la
	// olvidaron, o alguien la creó antes) -- en vez de un callejón sin
	// salida, esto la resetea directo. Ver el comentario largo en
	// handleCrearCuentaPadre sobre por qué SOLO puede tocar cuentas rol
	// "papa".
	r.PUT("/padres/:id/restablecer-password", auth, familia, s.handleRestablecerPasswordPadre)
}

func (s *Server) handleRegistrarHijo(c *gin.Context) {
	gID, exists := c.Get("guarderia_id")
	if !exists {
		// No debería pasar nunca si Auth() corrió antes -- lo pone siempre
		// en el contexto. Se loguea igual porque, si algún día SÍ pasa, es
		// justo el tipo de cosa que sin rastro es imposible de diagnosticar.
		log.Println("handleRegistrarHijo: guarderia_id no estaba en el contexto (¿Auth() no corrió antes?)")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo identificar la guardería"})
		return
	}

	var input struct {
		Nombre string `json:"nombre_niño"`
	}
	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nombre requerido"})
		return
	}

	var hijoID int
	var urlToken string

	query := `
        INSERT INTO hijos (nombre_niño, guarderia_id, url_token)
        VALUES ($1, $2, gen_random_uuid())
        RETURNING id, url_token`

	err := s.DB.QueryRow(query, input.Nombre, gID).Scan(&hijoID, &urlToken)
	if err != nil {
		fmt.Printf("Error al insertar hijo: %v\n", err)
		log.Printf("Error al crear niño en la base de datos: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear niño en la base de datos"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           hijoID,
		"nombre":       input.Nombre,
		"guarderia_id": gID,
		"url_token":    urlToken,
	})
}

func (s *Server) handleHijosDePadre(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	tokenUsuarioID, _ := c.Get("user_id")
	rol, _ := c.Get("rol")

	padreID := c.Param("id")

	// --- LÓGICA DE COMODÍN PARA EL PAPÁ ---
	// Si el ID es "0", usamos el ID que viene dentro del Token
	if padreID == "0" {
		padreID = fmt.Sprintf("%v", tokenUsuarioID)
	} else {
		// SEGURIDAD: si no es "0", quien consulta debe ser ADMIN o STAFF — evita que
		// un papá cambie el "0" por el ID de otro papá en la URL.
		if rol != "admin" && rol != "staff" {
			c.JSON(http.StatusForbidden, gin.H{"error": "No tienes permiso para consultar otros IDs"})
			return
		}
	}

	// Filtramos por padre_id Y por guarderia_id para asegurar que pertenezcan a la
	// misma sede. Incluimos el expediente extendido para que el portal del papá no
	// necesite otra llamada.
	query := `
        SELECT h.id, h.nombre_niño, h.activo, h.fecha_nacimiento, h.direccion,
               h.contacto_emergencia_nombre, h.contacto_emergencia_telefono, h.colegiatura_mensual
		FROM hijos h
		JOIN tutor_hijos th ON h.id = th.hijo_id
		WHERE th.padre_id = $1 AND th.guarderia_id = $2`

	rows, err := s.DB.Query(query, padreID, gID)
	if err != nil {
		log.Printf("Error al consultar hijos: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar hijos"})
		return
	}
	defer rows.Close()

	listaHijos := []NinoPerfil{}
	for rows.Next() {
		var h NinoPerfil
		if err := rows.Scan(
			&h.ID, &h.Nombre, &h.Activo, &h.FechaNacimiento, &h.Direccion,
			&h.ContactoEmergenciaNombre, &h.ContactoEmergenciaTelefono, &h.ColegiaturaMensual,
		); err != nil {
			continue
		}
		h.FechaNacimiento = soloFecha(h.FechaNacimiento)
		listaHijos = append(listaHijos, h)
	}

	c.JSON(http.StatusOK, listaHijos)
}

func (s *Server) handleVincularTutor(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")

	var req VinculacionRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	// El ON CONFLICT evita errores si intentan vincular al mismo padre/hijo dos veces
	query := `
        INSERT INTO tutor_hijos (padre_id, hijo_id, guarderia_id)
        VALUES ($1, $2, $3)
        ON CONFLICT (padre_id, hijo_id) DO NOTHING`

	_, err := s.DB.Exec(query, req.PadreID, req.HijoID, gID)
	if err != nil {
		log.Printf("Error en vinculación: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo realizar la vinculación"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "Vinculación exitosa",
		"padre_id": req.PadreID,
		"hijo_id":  req.HijoID,
	})
}

func (s *Server) handleBuscarHijos(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	queryParam := c.Query("q")

	query := `
        SELECT id, nombre_niño
        FROM hijos
        WHERE nombre_niño ILIKE $1 AND guarderia_id = $2
		AND activo = true
        LIMIT 5`

	rows, err := s.DB.Query(query, "%"+queryParam+"%", gID)

	lista := []gin.H{}

	if err != nil {
		fmt.Printf("Error en buscar-hijos: %v\n", err)
		log.Printf("Error al consultar la base de datos: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar la base de datos"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var nombre string
		if err := rows.Scan(&id, &nombre); err != nil {
			continue
		}
		lista = append(lista, gin.H{
			"id":          id,
			"nombre_niño": nombre,
		})
	}

	c.JSON(http.StatusOK, lista)
}

func (s *Server) handleBuscarPadres(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	queryParam := c.Query("q")

	var rows *sql.Rows
	var err error

	// Si queryParam está vacío, traemos todos los de la guardería
	if queryParam == "" {
		query := `
		SELECT p.id, p.nombre, COALESCE(h.id, 0), COALESCE(h.nombre_niño, '')
		FROM padres p
		LEFT JOIN tutor_hijos th ON p.id = th.padre_id
		LEFT JOIN hijos h ON th.hijo_id = h.id
		WHERE p.guarderia_id = $1
		ORDER BY p.nombre ASC`
		rows, err = s.DB.Query(query, gID)
	} else {
		query := `
		SELECT p.id, p.nombre, COALESCE(h.id, 0), COALESCE(h.nombre_niño, '')
		FROM padres p
		LEFT JOIN tutor_hijos th ON p.id = th.padre_id
		LEFT JOIN hijos h ON th.hijo_id = h.id
		WHERE p.nombre ILIKE $1 AND p.guarderia_id = $2
		ORDER BY p.nombre ASC`
		rows, err = s.DB.Query(query, "%"+queryParam+"%", gID)
	}

	if err != nil {
		log.Printf("Error al consultar la base de datos: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar la base de datos"})
		return
	}
	defer rows.Close()

	type PadreData struct {
		ID     int    `json:"id"`
		Nombre string `json:"nombre"`
		Hijos  []any  `json:"hijos"`
	}

	padresMap := make(map[int]*PadreData)

	for rows.Next() {
		var pID int
		var pNombre string
		var hID int
		var hNombre string

		if err := rows.Scan(&pID, &pNombre, &hID, &hNombre); err != nil {
			continue
		}

		if _, ok := padresMap[pID]; !ok {
			padresMap[pID] = &PadreData{
				ID:     pID,
				Nombre: pNombre,
				Hijos:  []any{},
			}
		}

		if hID != 0 {
			padresMap[pID].Hijos = append(padresMap[pID].Hijos, gin.H{
				"id":          hID,
				"nombre_niño": hNombre,
			})
		}
	}

	resultado := []PadreData{}
	for _, p := range padresMap {
		resultado = append(resultado, *p)
	}

	c.JSON(http.StatusOK, resultado)
}

func (s *Server) handleDesvincularHijo(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")

	var input struct {
		PadreID int `json:"padre_id"`
		HijoID  int `json:"hijo_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	// Aseguramos que solo se borre si la relación pertenece a esta guardería
	query := `
        DELETE FROM tutor_hijos
        WHERE padre_id = $1 AND hijo_id = $2 AND guarderia_id = $3`

	result, err := s.DB.Exec(query, input.PadreID, input.HijoID, gID)
	if err != nil {
		log.Printf("No se pudo realizar la desvinculación: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo realizar la desvinculación"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"mensaje": "No se encontró la relación o no pertenece a esta guardería"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"mensaje": "Desvinculación exitosa"})
}

func (s *Server) handleActualizarPadre(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")

	var req struct {
		ID     int    `json:"id"`
		Nombre string `json:"nombre"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	// Doble filtro (ID + Guardería): solo se actualiza si el padre pertenece a la
	// guardería del admin.
	query := "UPDATE padres SET nombre = $1 WHERE id = $2 AND guarderia_id = $3"

	result, err := s.DB.Exec(query, req.Nombre, req.ID, gID)
	if err != nil {
		log.Printf("Error interno al actualizar: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error interno al actualizar"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Padre no encontrado o no pertenece a esta guardería"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "Nombre actualizado correctamente"})
}

// handleCrearCuentaPadre da de alta la cuenta del portal (rol "papa") para
// un tutor que YA existe -- para cuando su rostro se registró sin marcar
// "crear cuenta" en el kiosco, o cuando esa cuenta falló en su momento (ver
// el comentario largo en asistencia.go/handleRegistrar sobre el choque de
// id). No vuelve a tocar Rekognition: el padre ya tiene su face_id, esto
// solo agrega credenciales de acceso al portal.
func (s *Server) handleCrearCuentaPadre(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	padreID := c.Param("id")

	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}
	input.Username = strings.TrimSpace(input.Username)
	if len(input.Username) < 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El usuario debe tener al menos 3 caracteres"})
		return
	}
	if len(input.Password) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "La contraseña debe tener al menos 8 caracteres"})
		return
	}

	var nombrePadre string
	err := s.DB.QueryRow("SELECT nombre FROM padres WHERE id = $1 AND guarderia_id = $2", padreID, gID).Scan(&nombrePadre)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tutor no encontrado"})
		return
	} else if err != nil {
		log.Printf("Error al consultar el padre %s para crear su cuenta: %v", padreID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo consultar al tutor"})
		return
	}

	// Misma convención que al crear la cuenta junto con el rostro: el
	// portal resuelve "quién soy" asumiendo usuarios.id == padres.id (ver
	// el comentario largo en asistencia.go). Acá no se puede elegir un id
	// libre distinto -- el padre ya existe con un id fijo, referenciado
	// por tutor_hijos/consentimientos/asistencia -- así que si ESE id
	// específico ya lo tiene otra cuenta, no hay forma de dar de alta esta
	// cuenta sin cambiar esa convención en todo el backend. En vez de un
	// mensaje genérico, se identifica de quién es esa cuenta: si es del
	// propio padre (rol "papa" -- lo más común: ya tenía cuenta, o alguien
	// la creó antes y no se acuerdan) se ofrece restablecer su contraseña
	// en vez de un callejón sin salida; si es de alguien más (coincidencia
	// rara con admin/staff) sí hace falta ayuda técnica de verdad.
	var usernameExistente, rolExistente string
	err = s.DBAuth.QueryRow("SELECT username, rol FROM usuarios WHERE id = $1", padreID).Scan(&usernameExistente, &rolExistente)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error al verificar si el padre %s ya tiene cuenta: %v", padreID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo verificar la cuenta"})
		return
	}
	if err == nil {
		if rolExistente == "papa" {
			c.JSON(http.StatusConflict, gin.H{
				"error":              fmt.Sprintf("Este tutor ya tiene una cuenta del portal (usuario \"%s\"). Si no recuerda la contraseña, puedes restablecerla en vez de crear una cuenta nueva.", usernameExistente),
				"username_existente": usernameExistente,
				"puede_restablecer":  true,
			})
			return
		}
		c.JSON(http.StatusConflict, gin.H{
			"error":             fmt.Sprintf("El id interno de este tutor choca con la cuenta de %s \"%s\" -- no se puede crear su cuenta del portal sin ayuda técnica. Contacta soporte.", rolExistente, usernameExistente),
			"puede_restablecer": false,
		})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error al procesar la contraseña: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al procesar la contraseña"})
		return
	}

	_, err = s.DBAuth.Exec(
		`INSERT INTO usuarios (id, guarderia_id, username, password_hash, pin_admin, rol, nombre, activo)
         VALUES ($1, $2, $3, $4, '0000', 'papa', $5, true)`,
		padreID, gID, input.Username, string(hash), nombrePadre,
	)
	if err != nil {
		if esUsernameDuplicado(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "Ese nombre de usuario ya existe"})
			return
		}
		log.Printf("No se pudo crear la cuenta de portal para el padre %s: %v", padreID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo crear la cuenta"})
		return
	}
	if _, err := s.DBAuth.Exec(`SELECT setval('usuarios_id_seq', (SELECT MAX(id) FROM usuarios))`); err != nil {
		log.Printf("No se pudo reacomodar la secuencia de usuarios tras vincular la cuenta del padre %s: %v", padreID, err)
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Cuenta creada"})
}

func (s *Server) handleDesactivarHijo(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	hijoID := c.Param("id")

	query := "UPDATE hijos SET activo = false WHERE id = $1 AND guarderia_id = $2"

	result, err := s.DB.Exec(query, hijoID, gID)
	if err != nil {
		log.Printf("Error al dar de baja: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al dar de baja"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Niño no encontrado"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"mensaje": "Alumno dado de baja correctamente"})
}

func (s *Server) handleEditarNombreHijo(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	hijoID := c.Param("id")

	var input struct {
		Nombre string `json:"nombre"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Nombre == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nombre es requerido"})
		return
	}

	query := "UPDATE hijos SET nombre_niño = $1 WHERE id = $2 AND guarderia_id = $3"
	_, err := s.DB.Exec(query, input.Nombre, hijoID, gID)
	if err != nil {
		log.Printf("Error al actualizar: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "Nombre actualizado"})
}

func (s *Server) handleActivarHijo(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	hijoID := c.Param("id")

	query := "UPDATE hijos SET activo = true WHERE id = $1 AND guarderia_id = $2"

	_, err := s.DB.Exec(query, hijoID, gID)
	if err != nil {
		log.Printf("Error al reactivar: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al reactivar"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"mensaje": "Alumno reactivado correctamente"})
}

func (s *Server) handleRegenerarToken(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	hijoID := c.Param("id")

	var nuevoToken string
	query := `UPDATE hijos SET url_token = gen_random_uuid()
		          WHERE id = $1 AND guarderia_id = $2
		          RETURNING url_token`
	err := s.DB.QueryRow(query, hijoID, gID).Scan(&nuevoToken)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Niño no encontrado"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url_token": nuevoToken})
}

// handleRestablecerPasswordPadre resetea la contraseña del portal de un
// tutor que YA tiene cuenta -- la olvidó, o alguien intentó "Crear cuenta"
// sin saber que ya existía (ver el 409 de handleCrearCuentaPadre, que
// apunta para acá cuando el id ocupado es justo del propio padre). Espejo
// de handleResetPasswordPersonal en personal.go, pero con rol = 'papa' fijo
// en el WHERE -- a propósito, para que este endpoint JAMÁS pueda tocar una
// cuenta de admin/staff aunque su id choque por la convención
// usuarios.id == padres.id (ver el comentario largo en
// handleCrearCuentaPadre).
func (s *Server) handleRestablecerPasswordPadre(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	adminID, _ := c.Get("user_id")
	padreID := c.Param("id")

	var input struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || len(input.Password) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "La contraseña debe tener al menos 8 caracteres"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error al procesar la contraseña: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al procesar la contraseña"})
		return
	}

	res, err := s.DBAuth.Exec(
		`UPDATE usuarios SET password_hash = $1
         WHERE id = $2 AND guarderia_id = $3 AND rol = 'papa'`,
		string(hash), padreID, gID,
	)
	if err != nil {
		log.Printf("No se pudo restablecer la contraseña del padre %s: %v", padreID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo restablecer la contraseña"})
		return
	}
	if filas, _ := res.RowsAffected(); filas == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Este tutor no tiene una cuenta del portal"})
		return
	}

	s.registrarAcceso("padre_password_reset", gID, adminID, "contraseña del portal reseteada para el padre "+padreID, c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"message": "Contraseña actualizada"})
}
