package server

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"biometrico/internal/middleware"
)

// solicitudes.go es el alta de una guardería nueva: autoservicio en la
// solicitud (cualquiera llena el formulario público), pero CON aprobación
// manual antes de que exista un tenant de verdad -- crear una guardería
// consume Rekognition/S3 reales que paga el dueño de la plataforma, así que
// no queda abierto a que cualquiera cree tenants gratis sin más.
//
// Por eso esto NUNCA inserta directo en `guarderias`/`usuarios`: solo en
// `solicitudes_guarderia`, con estado "pendiente". La aprobación (protegida
// por PLATFORM_ADMIN_KEY, ver middleware.RequirePlatformKey) es lo único que
// de verdad crea el tenant.

// SolicitudGuarderia es lo que ve /plataforma/solicitudes -- nunca incluye
// password_hash, mismo criterio que Personal en personal.go.
type SolicitudGuarderia struct {
	ID               int     `json:"id"`
	NombreGuarderia  string  `json:"nombre_guarderia"`
	Direccion        *string `json:"direccion"`
	NombreContacto   string  `json:"nombre_contacto"`
	EmailContacto    string  `json:"email_contacto"`
	TelefonoContacto *string `json:"telefono_contacto"`
	UsernameDeseado  string  `json:"username_deseado"`
	Estado           string  `json:"estado"`
	NotaRevision     *string `json:"nota_revision"`
	CreadoEn         string  `json:"creado_en"`
}

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

func (s *Server) registrarRutasSolicitudes(r *gin.Engine) {
	platform := middleware.RequirePlatformKey(s.PlatformAdminKey)

	// Pública a propósito: es el formulario "quiero dar de alta mi
	// guardería", nadie tiene sesión todavía.
	r.POST("/solicitudes-guarderia", s.handleCrearSolicitud)

	r.GET("/plataforma/solicitudes", platform, s.handleListarSolicitudes)
	r.POST("/plataforma/solicitudes/:id/aprobar", platform, s.handleAprobarSolicitud)
	r.POST("/plataforma/solicitudes/:id/rechazar", platform, s.handleRechazarSolicitud)
}

func (s *Server) handleCrearSolicitud(c *gin.Context) {
	var input struct {
		NombreGuarderia  string `json:"nombre_guarderia"`
		Direccion        string `json:"direccion"`
		NombreContacto   string `json:"nombre_contacto"`
		EmailContacto    string `json:"email_contacto"`
		TelefonoContacto string `json:"telefono_contacto"`
		UsernameDeseado  string `json:"username_deseado"`
		Password         string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	input.NombreGuarderia = strings.TrimSpace(input.NombreGuarderia)
	input.Direccion = strings.TrimSpace(input.Direccion)
	input.NombreContacto = strings.TrimSpace(input.NombreContacto)
	input.EmailContacto = strings.TrimSpace(strings.ToLower(input.EmailContacto))
	input.TelefonoContacto = strings.TrimSpace(input.TelefonoContacto)
	input.UsernameDeseado = strings.TrimSpace(input.UsernameDeseado)

	if input.NombreGuarderia == "" || input.NombreContacto == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El nombre de la guardería y del contacto son obligatorios"})
		return
	}
	if !emailRegex.MatchString(input.EmailContacto) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Correo de contacto inválido"})
		return
	}
	if len(input.UsernameDeseado) < 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El usuario administrador debe tener al menos 3 caracteres"})
		return
	}
	if len(input.Password) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "La contraseña debe tener al menos 8 caracteres"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error al procesar la contraseña: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al procesar la contraseña"})
		return
	}

	var direccion, telefono sql.NullString
	if input.Direccion != "" {
		direccion = sql.NullString{String: input.Direccion, Valid: true}
	}
	if input.TelefonoContacto != "" {
		telefono = sql.NullString{String: input.TelefonoContacto, Valid: true}
	}

	_, err = s.DBAuth.Exec(
		`INSERT INTO solicitudes_guarderia
         (nombre_guarderia, direccion, nombre_contacto, email_contacto, telefono_contacto, username_deseado, password_hash)
         VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		input.NombreGuarderia, direccion, input.NombreContacto, input.EmailContacto, telefono, input.UsernameDeseado, string(hash),
	)
	if err != nil {
		log.Printf("No se pudo guardar la solicitud de guardería de %s: %v", input.EmailContacto, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo registrar tu solicitud, intenta de nuevo"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Solicitud recibida. Te contactaremos para confirmar el alta de tu guardería."})
}

func (s *Server) handleListarSolicitudes(c *gin.Context) {
	estado := c.DefaultQuery("estado", "pendiente")

	rows, err := s.DBAuth.Query(
		`SELECT id, nombre_guarderia, direccion, nombre_contacto, email_contacto, telefono_contacto,
                username_deseado, estado, nota_revision, creado_en
         FROM solicitudes_guarderia
         WHERE estado = $1
         ORDER BY creado_en ASC`,
		estado,
	)
	if err != nil {
		log.Printf("Error al consultar solicitudes: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar solicitudes"})
		return
	}
	defer rows.Close()

	solicitudes := []SolicitudGuarderia{}
	for rows.Next() {
		var sg SolicitudGuarderia
		if err := rows.Scan(&sg.ID, &sg.NombreGuarderia, &sg.Direccion, &sg.NombreContacto, &sg.EmailContacto,
			&sg.TelefonoContacto, &sg.UsernameDeseado, &sg.Estado, &sg.NotaRevision, &sg.CreadoEn); err != nil {
			continue
		}
		solicitudes = append(solicitudes, sg)
	}
	c.JSON(http.StatusOK, solicitudes)
}

// handleAprobarSolicitud crea de verdad la guardería y su primera cuenta
// admin (usando el username/password_hash ya capturados en la solicitud —
// quien la llenó ya sabe su propia contraseña, no hace falta pedirla otra
// vez ni regresarla). El PIN administrativo (obligatorio en el esquema,
// nunca lo pide este formulario) queda en "0000" -- el admin lo cambia desde
// Gestión de Personal como cualquier otra cuenta.
func (s *Server) handleAprobarSolicitud(c *gin.Context) {
	id := c.Param("id")

	var nombreGuarderia, direccion, username, passwordHash, estado string
	err := s.DBAuth.QueryRow(
		`SELECT nombre_guarderia, COALESCE(direccion, ''), username_deseado, password_hash, estado
         FROM solicitudes_guarderia WHERE id = $1`,
		id,
	).Scan(&nombreGuarderia, &direccion, &username, &passwordHash, &estado)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Solicitud no encontrada"})
		return
	} else if err != nil {
		log.Printf("Error al consultar la solicitud: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar la solicitud"})
		return
	}
	if estado != "pendiente" {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Esta solicitud ya fue %s", estado)})
		return
	}

	tx, err := s.DBAuth.Begin()
	if err != nil {
		log.Printf("No se pudo iniciar la operación: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo iniciar la operación"})
		return
	}
	defer tx.Rollback()

	slug, err := s.slugGuarderiaDisponible(tx, nombreGuarderia)
	if err != nil {
		log.Printf("No se pudo generar el identificador de la guardería: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo generar el identificador de la guardería"})
		return
	}

	var direccionParam sql.NullString
	if direccion != "" {
		direccionParam = sql.NullString{String: direccion, Valid: true}
	}

	var nuevaGuarderiaID int
	if err := tx.QueryRow(
		`INSERT INTO guarderias (nombre, slug, direccion) VALUES ($1, $2, $3) RETURNING id`,
		nombreGuarderia, slug, direccionParam,
	).Scan(&nuevaGuarderiaID); err != nil {
		log.Printf("No se pudo crear la guardería '%s' (solicitud #%s): %v", nombreGuarderia, id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo crear la guardería"})
		return
	}

	if _, err := tx.Exec(
		`INSERT INTO usuarios (guarderia_id, username, password_hash, pin_admin, rol, activo)
         VALUES ($1, $2, $3, '0000', 'admin', true)`,
		nuevaGuarderiaID, username, passwordHash,
	); err != nil {
		if esUsernameDuplicado(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "Ese nombre de usuario ya lo usa otra cuenta -- pide al solicitante uno distinto y créala a mano, o recházala."})
			return
		}
		log.Printf("No se pudo crear la cuenta administrador de la guardería %d (solicitud #%s): %v", nuevaGuarderiaID, id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo crear la cuenta administrador"})
		return
	}

	if _, err := tx.Exec(
		`UPDATE solicitudes_guarderia SET estado = 'aprobada', guarderia_id = $1, revisado_en = now() WHERE id = $2`,
		nuevaGuarderiaID, id,
	); err != nil {
		log.Printf("No se pudo marcar la solicitud #%s como aprobada (guardería %d ya creada): %v", id, nuevaGuarderiaID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo marcar la solicitud como aprobada"})
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("No se pudo completar la aprobación de la solicitud #%s (guardería %d): %v", id, nuevaGuarderiaID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo completar la aprobación"})
		return
	}

	s.registrarAcceso("guarderia_aprobada", nuevaGuarderiaID, nil,
		fmt.Sprintf("guardería '%s' (slug %s) creada desde la solicitud #%s", nombreGuarderia, slug, id), c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"message":      "Guardería creada. Ya puede iniciar sesión con el usuario que registró.",
		"guarderia_id": nuevaGuarderiaID,
		"slug":         slug,
	})
}

func (s *Server) handleRechazarSolicitud(c *gin.Context) {
	id := c.Param("id")
	var input struct {
		Nota string `json:"nota"`
	}
	c.ShouldBindJSON(&input)

	var nota sql.NullString
	if strings.TrimSpace(input.Nota) != "" {
		nota = sql.NullString{String: strings.TrimSpace(input.Nota), Valid: true}
	}

	res, err := s.DBAuth.Exec(
		`UPDATE solicitudes_guarderia SET estado = 'rechazada', nota_revision = $1, revisado_en = now()
         WHERE id = $2 AND estado = 'pendiente'`,
		nota, id,
	)
	if err != nil {
		log.Printf("No se pudo rechazar la solicitud: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo rechazar la solicitud"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Solicitud no encontrada o ya revisada"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Solicitud rechazada"})
}

var slugCaracteresInvalidos = regexp.MustCompile(`[^a-z0-9]+`)

// slugGuarderiaDisponible genera un slug legible ("mi-guardería-2" ->
// "mi-guarderia-2") y le agrega un sufijo numérico si ya existe -- se
// consulta dentro de la misma transacción que lo va a usar para no chocar
// con otra aprobación concurrente.
func (s *Server) slugGuarderiaDisponible(tx *sql.Tx, nombre string) (string, error) {
	base := strings.ToLower(nombre)
	base = strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ñ", "n", "ü", "u",
	).Replace(base)
	base = slugCaracteresInvalidos.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	if base == "" {
		base = "guarderia"
	}

	candidato := base
	for intento := 2; ; intento++ {
		var existe bool
		if err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM guarderias WHERE slug = $1)", candidato).Scan(&existe); err != nil {
			return "", err
		}
		if !existe {
			return candidato, nil
		}
		candidato = fmt.Sprintf("%s-%d", base, intento)
	}
}
