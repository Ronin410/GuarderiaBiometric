package server

import (
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"

	"biometrico/internal/middleware"
)

// Personal es la fila que ve el panel "Gestión de personal" — nunca incluye
// password_hash ni pin_admin (el PIN se cambia por un endpoint aparte, y
// nunca se regresa su valor actual, mismo criterio que /verificar-pin).
// Permisos va como slice normal (no *[]string): un valor nil se serializa
// como JSON null ("sin personalizar, acceso completo") y un slice vacío
// no-nil como [] ("sin acceso a nada") -- exactamente la distinción que
// necesita el frontend, sin tener que inventar un envoltorio aparte.
type Personal struct {
	ID        int      `json:"id"`
	Username  string   `json:"username"`
	Nombre    *string  `json:"nombre"`
	Rol       string   `json:"rol"`
	Activo    bool     `json:"activo"`
	CreatedAt string   `json:"created_at"`
	Permisos  []string `json:"permisos"`
}

var pinRegex = regexp.MustCompile(`^\d{4}$`)

// AreasPermiso enumera las secciones que un admin puede conceder o quitar
// individualmente a una cuenta de staff -- las mismas 9 pestañas que hoy
// exige el PIN por igual (TABS_PROTEGIDAS en App.jsx). "familia" es la única
// que no comparte literal con el nombre de su pestaña en la URL (que es
// "admin", por herencia histórica) -- se usa el nombre visible en la UI
// ("Familia", el directorio de tutores) para no confundirla con el rol admin.
var AreasPermiso = []string{
	"familia", "bitacora", "reportes", "perfiles", "pagos",
	"estadisticas", "configuracion", "menu", "circulares",
}

func areaPermisoValida(area string) bool {
	for _, a := range AreasPermiso {
		if a == area {
			return true
		}
	}
	return false
}

func (s *Server) registrarRutasPersonal(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)
	admin := middleware.RequireAdmin()

	r.GET("/admin/personal", auth, admin, s.handleListarPersonal)
	r.POST("/admin/personal", auth, admin, s.handleCrearPersonal)
	r.PUT("/admin/personal/:id", auth, admin, s.handleActualizarPersonal)
	r.PUT("/admin/personal/:id/pin", auth, admin, s.handleActualizarPinPersonal)
	r.PUT("/admin/personal/:id/password", auth, admin, s.handleResetPasswordPersonal)
	r.PUT("/admin/personal/:id/permisos", auth, admin, s.handleActualizarPermisosPersonal)
}

// handleListarPersonal regresa las cuentas de personal (admin/staff) de la
// guardería del usuario en sesión — nunca las de "papa" (esas se gestionan
// implícitamente al dar de alta un tutor, no desde aquí) ni las de otra
// guardería.
func (s *Server) handleListarPersonal(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")

	rows, err := s.DBAuth.Query(
		`SELECT id, username, nombre, rol, activo, created_at, permisos
         FROM usuarios
         WHERE guarderia_id = $1 AND rol IN ('admin', 'staff')
         ORDER BY created_at ASC`,
		gID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar personal"})
		return
	}
	defer rows.Close()

	personal := []Personal{}
	for rows.Next() {
		var p Personal
		if err := rows.Scan(&p.ID, &p.Username, &p.Nombre, &p.Rol, &p.Activo, &p.CreatedAt, pq.Array(&p.Permisos)); err != nil {
			continue
		}
		personal = append(personal, p)
	}
	c.JSON(http.StatusOK, personal)
}

// handleCrearPersonal da de alta una nueva cuenta de staff o admin. A
// diferencia del viejo /usuarios/registro (público, sin login, y que
// aceptaba guarderia_id del body), guarderia_id sale siempre del JWT de
// quien hace la petición — nunca del body.
func (s *Server) handleCrearPersonal(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	adminID, _ := c.Get("user_id")

	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Nombre   string `json:"nombre"`
		Rol      string `json:"rol"`
		Pin      string `json:"pin"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	input.Username = strings.TrimSpace(input.Username)
	input.Nombre = strings.TrimSpace(input.Nombre)

	if input.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El usuario es obligatorio"})
		return
	}
	if len(input.Password) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "La contraseña debe tener al menos 8 caracteres"})
		return
	}
	if input.Rol != "admin" && input.Rol != "staff" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El rol debe ser admin o staff"})
		return
	}
	if !pinRegex.MatchString(input.Pin) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El PIN debe ser de exactamente 4 dígitos"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al procesar la contraseña"})
		return
	}

	var nombre sql.NullString
	if input.Nombre != "" {
		nombre = sql.NullString{String: input.Nombre, Valid: true}
	}

	var nuevoID int
	err = s.DBAuth.QueryRow(
		`INSERT INTO usuarios (guarderia_id, username, password_hash, pin_admin, rol, nombre, activo)
         VALUES ($1, $2, $3, $4, $5, $6, true) RETURNING id`,
		gID, input.Username, string(hash), input.Pin, input.Rol, nombre,
	).Scan(&nuevoID)
	if err != nil {
		if esUsernameDuplicado(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "Ese nombre de usuario ya existe"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo crear la cuenta"})
		return
	}

	s.registrarAcceso("personal_creado", gID, adminID,
		fmt.Sprintf("cuenta creada: %s (%s)", input.Username, input.Rol), c.ClientIP())

	c.JSON(http.StatusCreated, gin.H{"id": nuevoID, "message": "Cuenta creada"})
}

// handleActualizarPersonal edita nombre/rol/activo de una cuenta existente.
// No toca username, password ni PIN — esos tienen sus propios endpoints
// porque son acciones con su propia validación e implicaciones de seguridad.
func (s *Server) handleActualizarPersonal(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	adminID, _ := c.Get("user_id")
	targetID := c.Param("id")

	var input struct {
		Nombre string `json:"nombre"`
		Rol    string `json:"rol"`
		Activo bool   `json:"activo"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}
	if input.Rol != "admin" && input.Rol != "staff" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El rol debe ser admin o staff"})
		return
	}

	// Un admin no puede desactivarse ni degradarse a sí mismo desde aquí —
	// evita que la guardería se quede sin ningún admin activo por accidente
	// (o que alguien se autolimite sin querer a media edición).
	if fmt.Sprintf("%v", adminID) == targetID && (!input.Activo || input.Rol != "admin") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No puedes desactivarte ni quitarte el rol admin a ti mismo"})
		return
	}

	var nombre sql.NullString
	nombreLimpio := strings.TrimSpace(input.Nombre)
	if nombreLimpio != "" {
		nombre = sql.NullString{String: nombreLimpio, Valid: true}
	}

	res, err := s.DBAuth.Exec(
		`UPDATE usuarios SET nombre = $1, rol = $2, activo = $3
         WHERE id = $4 AND guarderia_id = $5 AND rol IN ('admin', 'staff')`,
		nombre, input.Rol, input.Activo, targetID, gID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo actualizar la cuenta"})
		return
	}
	if filas, _ := res.RowsAffected(); filas == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cuenta no encontrada"})
		return
	}

	s.registrarAcceso("personal_actualizado", gID, adminID,
		fmt.Sprintf("cuenta %s editada (rol=%s, activo=%v)", targetID, input.Rol, input.Activo), c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "Cuenta actualizada"})
}

// handleActualizarPinPersonal es el "administrador pueda configurar el PIN
// de los staff" pedido explícitamente: cambia el pin_admin de una cuenta sin
// tocar nada más. Reutiliza el mismo formato de 4 dígitos que valida
// /verificar-pin.
func (s *Server) handleActualizarPinPersonal(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	adminID, _ := c.Get("user_id")
	targetID := c.Param("id")

	var input struct {
		Pin string `json:"pin"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || !pinRegex.MatchString(input.Pin) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El PIN debe ser de exactamente 4 dígitos"})
		return
	}

	res, err := s.DBAuth.Exec(
		`UPDATE usuarios SET pin_admin = $1
         WHERE id = $2 AND guarderia_id = $3 AND rol IN ('admin', 'staff')`,
		input.Pin, targetID, gID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo cambiar el PIN"})
		return
	}
	if filas, _ := res.RowsAffected(); filas == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cuenta no encontrada"})
		return
	}

	s.registrarAcceso("personal_pin_cambiado", gID, adminID, "PIN cambiado para cuenta "+targetID, c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"message": "PIN actualizado"})
}

// handleResetPasswordPersonal deja que el admin resetee la contraseña de
// alguien de su personal (ej. la olvidó) sin necesitar la contraseña vieja.
func (s *Server) handleResetPasswordPersonal(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	adminID, _ := c.Get("user_id")
	targetID := c.Param("id")

	var input struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || len(input.Password) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "La contraseña debe tener al menos 8 caracteres"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al procesar la contraseña"})
		return
	}

	res, err := s.DBAuth.Exec(
		`UPDATE usuarios SET password_hash = $1
         WHERE id = $2 AND guarderia_id = $3 AND rol IN ('admin', 'staff')`,
		string(hash), targetID, gID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo cambiar la contraseña"})
		return
	}
	if filas, _ := res.RowsAffected(); filas == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cuenta no encontrada"})
		return
	}

	s.registrarAcceso("personal_password_reset", gID, adminID, "contraseña reseteada para cuenta "+targetID, c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"message": "Contraseña actualizada"})
}

// handleActualizarPermisosPersonal es "permisos personalizados por
// docente": reemplaza el candado de un PIN compartido -- que, una vez
// conocido, desbloqueaba TODAS las secciones sensibles por igual y sin que
// el backend volviera a revisar nada -- por una lista explícita de qué
// secciones puede tocar cada cuenta de staff, exigida por RequireArea en
// cada request, no solo escondida del menú.
//
// permisos: null (u omitido en el body) regresa la cuenta al
// comportamiento de siempre: acceso completo, para no obligar a configurar
// cada cuenta existente de golpe al desplegar esto. permisos: [] (array
// vacío, pero presente) dado explícito dejar a la cuenta sin acceso a
// ninguna sección protegida.
func (s *Server) handleActualizarPermisosPersonal(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	adminID, _ := c.Get("user_id")
	targetID := c.Param("id")

	var input struct {
		Permisos *[]string `json:"permisos"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	var res sql.Result
	var err error
	if input.Permisos == nil {
		res, err = s.DBAuth.Exec(
			`UPDATE usuarios SET permisos = NULL WHERE id = $1 AND guarderia_id = $2 AND rol IN ('admin', 'staff')`,
			targetID, gID,
		)
	} else {
		for _, area := range *input.Permisos {
			if !areaPermisoValida(area) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Área de permiso desconocida: " + area})
				return
			}
		}
		res, err = s.DBAuth.Exec(
			`UPDATE usuarios SET permisos = $1 WHERE id = $2 AND guarderia_id = $3 AND rol IN ('admin', 'staff')`,
			pq.Array(*input.Permisos), targetID, gID,
		)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo actualizar los permisos"})
		return
	}
	if filas, _ := res.RowsAffected(); filas == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cuenta no encontrada"})
		return
	}

	s.registrarAcceso("personal_permisos_actualizados", gID, adminID,
		"permisos actualizados para cuenta "+targetID, c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"message": "Permisos actualizados"})
}

// esUsernameDuplicado detecta el error de unique_violation de Postgres
// (código 23505) específicamente sobre usuarios.username, para devolver un
// mensaje legible en vez de uno genérico. Revisa el nombre de la
// restricción (no solo el código 23505) a propósito: ese mismo código
// también salta por chocar contra la PRIMARY KEY -- asistencia.go fuerza un
// id explícito al crear la cuenta del portal junto con el rostro, así que
// un choque de id ahí es una posibilidad real, y reportarlo como "usuario
// ya existe" sería engañoso (el usuario pedido podía estar libre; lo que
// chocó fue el id).
func esUsernameDuplicado(err error) bool {
	pqErr, ok := err.(*pq.Error)
	return ok && pqErr.Code == "23505" && pqErr.Constraint == "usuarios_username_key"
}
