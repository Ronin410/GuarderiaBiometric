package server

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"

	"biometrico/internal/applog"
	"biometrico/internal/middleware"
)

// PinRequest es el cuerpo esperado por /verificar-pin.
type PinRequest struct {
	Pin string `json:"pin"`
}

// Vida de las cookies de sesión (biosafe_token y biosafe_csrf) -- coincide
// con la expiración del JWT para que ambas cosas caduquen juntas.
//
// Distinta según el rol a propósito: admin/staff manejan datos sensibles de
// TODAS las familias de la guardería (domicilios, pagos, expedientes) y se
// quedan con la sesión corta de siempre -- se re-loguean cada día. Un papá
// solo ve lo de sus propios hijos y en su mayoría es lectura (bitácora,
// pagos, circulares); pedirle credenciales seguido no suma seguridad real
// aquí y sí rompe el caso de uso (que la PWA se quede abierta en su
// celular, entrando rápido cuando quiera ver algo). handleMe además
// renueva la sesión del papá en cada visita (ver más abajo) -- mientras
// siga abriendo la app de vez en cuando, nunca la ve expirar.
const (
	duracionCookieSegundosStaff = 24 * 60 * 60
	duracionCookieSegundosPapa  = 90 * 24 * 60 * 60
)

func duracionSesion(rol string) int {
	if rol == "papa" {
		return duracionCookieSegundosPapa
	}
	return duracionCookieSegundosStaff
}

func (s *Server) registrarRutasAuth(r *gin.Engine) {
	r.POST("/login", s.loginLimiter.Middleware(), s.handleLogin)
	r.POST("/verificar-pin", middleware.Auth(s.JWTKey), s.pinLimiter.Middleware(), s.handleVerificarPin)
	r.GET("/me", middleware.Auth(s.JWTKey), s.handleMe)
	r.POST("/logout", middleware.Auth(s.JWTKey), s.handleLogout)
}

// generarCSRFToken produce un valor aleatorio para la cookie biosafe_csrf
// (patrón "double-submit cookie": el frontend la lee y la reenvía en el
// header X-CSRF-Token en cada petición que modifica datos; middleware.Auth
// exige que ambas coincidan).
func generarCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Server) handleLogin(c *gin.Context) {
	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
		// Tipo es el selector "Staff/Admin" vs "Soy Papá" de la pantalla de
		// login -- "aunque deje la opción de staff/admin, si pongo el
		// usuario y contraseña del papá me deja iniciar sesión como papá":
		// antes esto se mandaba pero nunca se validaba contra el rol real
		// de la cuenta, así que cualquier credencial válida entraba sin
		// importar qué pestaña estuviera seleccionada. Opcional (peticiones
		// viejas o de otros clientes sin este campo no se rompen: sin tipo
		// no se aplica el filtro).
		Tipo string `json:"tipo"`
	}

	if err := c.ShouldBindJSON(&creds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido"})
		return
	}

	var id, gID int
	var passHash, rol, pinAdmin, gNombre, gSlug string
	var activo bool
	var permisos []string
	// Nota: pin_admin se consulta solo para mantener la forma de la fila; nunca se
	// expone en la respuesta. La verificación del PIN se hace en /verificar-pin.
	query := `
		SELECT
            u.id, u.guarderia_id, u.password_hash, u.rol, u.pin_admin,
            g.nombre, g.slug, u.activo, u.permisos
        FROM usuarios u
        INNER JOIN guarderias g ON u.guarderia_id = g.id
        WHERE u.username = $1`

	err := s.DBAuth.QueryRow(query, creds.Username).Scan(&id, &gID, &passHash, &rol, &pinAdmin, &gNombre, &gSlug, &activo, pq.Array(&permisos))
	if err != nil {
		if err == sql.ErrNoRows {
			applog.Warn("Login: usuario no encontrado", "username", creds.Username)
			s.registrarAcceso("login_fallido", nil, nil, "usuario no encontrado: "+creds.Username, c.ClientIP())
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no existe"})
		} else {
			s.logError(c, "Error de BD al hacer login", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error de BD"})
		}
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(passHash), []byte(creds.Password))
	if err != nil {
		applog.Warn("Intento de login con contraseña incorrecta", "username", creds.Username)
		s.registrarAcceso("login_fallido", gID, id, "contraseña incorrecta", c.ClientIP())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Contraseña incorrecta"})
		return
	}

	// Mismo criterio que la validación de "activo" de abajo: se revisa
	// DESPUÉS de la contraseña, para no revelarle a quien intenta el login
	// si el usuario existe bajo el otro rol solo por el mensaje de error.
	// "papa" solo entra por la pestaña "Soy Papá"; "admin"/"staff" solo por
	// "Staff/Admin" -- sin esto, cualquier credencial válida entraba sin
	// importar qué pestaña estuviera seleccionada.
	if creds.Tipo == "papa" && rol != "papa" {
		applog.Warn("Login rechazado: tipo de acceso incorrecto", "username", creds.Username, "rol_real", rol, "tipo_pedido", "papa")
		s.registrarAcceso("login_fallido", gID, id, "tipo de acceso incorrecto: "+creds.Username, c.ClientIP())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Esta cuenta no es de una familia. Entra desde \"Staff / Admin\"."})
		return
	}
	if creds.Tipo == "staff" && rol == "papa" {
		applog.Warn("Login rechazado: tipo de acceso incorrecto", "username", creds.Username, "rol_real", rol, "tipo_pedido", "staff")
		s.registrarAcceso("login_fallido", gID, id, "tipo de acceso incorrecto: "+creds.Username, c.ClientIP())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Esta cuenta es de una familia. Entra desde \"Soy Papá\"."})
		return
	}

	// La contraseña ya se validó antes de revisar esto a propósito: así un
	// intento con contraseña incorrecta contra una cuenta desactivada no le
	// revela a quien lo intenta que la cuenta existe y está desactivada.
	//
	// Trade-off: esto bloquea logins NUEVOS de inmediato, pero una sesión ya
	// iniciada (cookie con JWT válido) sigue funcionando hasta que expira
	// (24h) — Auth() no vuelve a consultar la BD en cada request. Aceptable
	// para el tamaño de esta app (nadie más audita revocación en tiempo
	// real hoy); si se necesita corte inmediato, hay que revisar "activo" en
	// Auth() con el costo de una consulta extra por request.
	if !activo {
		applog.Warn("Login rechazado: cuenta desactivada", "username", creds.Username)
		s.registrarAcceso("login_fallido", gID, id, "cuenta desactivada: "+creds.Username, c.ClientIP())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Esta cuenta está desactivada"})
		return
	}

	duracion := duracionSesion(rol)
	expirationTime := time.Now().Add(time.Duration(duracion) * time.Second)
	claims := &middleware.Claims{
		UserID:      id,
		GuarderiaID: gID,
		Rol:         rol,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	// permisos == nil (columna NULL en BD, cuenta sin personalizar) se deja
	// fuera del JWT a propósito -- ver el comentario de Claims.Permisos.
	if permisos != nil {
		claims.Permisos = &permisos
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(s.JWTKey)
	if err != nil {
		s.logError(c, "Error al firmar el JWT", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al generar token"})
		return
	}

	csrfToken, err := generarCSRFToken()
	if err != nil {
		s.logError(c, "Error al generar el token CSRF", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al generar token"})
		return
	}

	// SameSite=None porque frontend y backend viven en orígenes distintos
	// (y seguirían siendo "sitios" distintos aunque ambos vivieran bajo
	// onrender.com, que está en la Public Suffix List) — el navegador no
	// mandaría la cookie con Lax en las peticiones del frontend. None exige
	// Secure, que ya cumplimos (Render termina TLS en producción).
	c.SetSameSite(http.SameSiteNoneMode)
	c.SetCookie(middleware.CookieToken, tokenStr, duracion, "/", "", true, true)
	c.SetCookie(middleware.CookieCSRF, csrfToken, duracion, "/", "", true, false)

	applog.Info("Login exitoso", "username", creds.Username)
	s.registrarAcceso("login_exitoso", gID, id, creds.Username, c.ClientIP())
	c.JSON(http.StatusOK, gin.H{
		"user_id":          id,
		"guarderia_id":     gID,
		"guarderia_nombre": gNombre,
		"guarderia_slug":   gSlug,
		"rol":              rol,
		"username":         creds.Username,
		"expires_at":       expirationTime.Unix(),
		// null = sin permisos personalizados (acceso completo tras el PIN,
		// como siempre); array (incluso vacío) = la lista exacta de
		// secciones permitidas. El frontend usa esto para decidir qué
		// pestañas mostrar sin tener que volver a pedir el PIN por cada una.
		"permisos": permisos,
		// El frontend lo guarda en memoria y lo reenvía en X-CSRF-Token en
		// cada petición que modifica datos -- no puede leerlo de la cookie
		// biosafe_csrf vía document.cookie cuando frontend y backend viven
		// en dominios de verdad distintos (ver el comentario de SameSite
		// arriba, y el de axiosConfig.js en el frontend).
		"csrf_token": csrfToken,
	})
}

// handleMe restaura la sesión al recargar la página: la cookie httpOnly es
// invisible a JavaScript, así que el frontend no puede leer quién es el
// usuario logueado directamente — se lo pregunta aquí, detrás de Auth()
// (que ya validó la cookie antes de llegar acá).
func (s *Server) handleMe(c *gin.Context) {
	uid, _ := c.Get("user_id")
	gID, _ := c.Get("guarderia_id")
	rol, _ := c.Get("rol")

	var username, gNombre, gSlug string
	err := s.DBAuth.QueryRow(
		`SELECT u.username, g.nombre, g.slug
         FROM usuarios u
         INNER JOIN guarderias g ON u.guarderia_id = g.id
         WHERE u.id = $1`,
		uid,
	).Scan(&username, &gNombre, &gSlug)
	if err != nil {
		s.logError(c, "No se pudo cargar la sesión", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo cargar la sesión"})
		return
	}

	var expiraEn int64
	if expRaw, ok := c.Get("token_exp"); ok {
		if expTime, ok := expRaw.(time.Time); ok {
			expiraEn = expTime.Unix()
		}
	}

	// Igual que en /login: si Auth() no puso "permisos" en el contexto es
	// porque el JWT no lo traía (cuenta sin personalizar) -- permisos se
	// queda en nil, que el JSON manda como null.
	var permisos []string
	if permisosRaw, ok := c.Get("permisos"); ok {
		permisos, _ = permisosRaw.([]string)
	}

	// El backend sí puede leer la cookie biosafe_csrf (vive en su propio
	// dominio) aunque el frontend no pueda -- se la regresa en el body para
	// que, al recargar la página, App.jsx vuelva a tener el valor en
	// memoria sin haber pasado por /login otra vez. Vacío si por lo que sea
	// no llegó (no debería pasar con una sesión válida, pero no hay razón
	// para tronar /me por eso).
	csrfToken, _ := c.Cookie(middleware.CookieCSRF)

	// Sesión deslizante SOLO para "papa": cada vez que abre la app (que es
	// justo cuando se llama /me) se le emite un JWT nuevo con la ventana
	// completa de otros 90 días -- mientras siga abriendo la app de vez en
	// cuando, nunca la ve expirar de verdad. admin/staff NO se tocan aquí a
	// propósito: se quedan con la sesión corta de siempre, sin renovación
	// automática (ver duracionSesion más arriba).
	if uidInt, okUID := uid.(int); okUID && rol == "papa" {
		if gIDInt, okGID := gID.(int); okGID {
			duracion := duracionSesion("papa")
			expirationTime := time.Now().Add(time.Duration(duracion) * time.Second)
			claims := &middleware.Claims{
				UserID:      uidInt,
				GuarderiaID: gIDInt,
				Rol:         "papa",
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(expirationTime),
				},
			}
			if permisos != nil {
				claims.Permisos = &permisos
			}
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			if tokenStr, err := token.SignedString(s.JWTKey); err != nil {
				s.logError(c, "No se pudo renovar la sesión del padre", err, "padre_id", uid)
			} else {
				c.SetSameSite(http.SameSiteNoneMode)
				c.SetCookie(middleware.CookieToken, tokenStr, duracion, "/", "", true, true)
				c.SetCookie(middleware.CookieCSRF, csrfToken, duracion, "/", "", true, false)
				expiraEn = expirationTime.Unix()
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":          uid,
		"guarderia_id":     gID,
		"guarderia_nombre": gNombre,
		"guarderia_slug":   gSlug,
		"rol":              rol,
		"username":         username,
		"expires_at":       expiraEn,
		"permisos":         permisos,
		"csrf_token":       csrfToken,
	})
}

// handleLogout borra las cookies de sesión. JavaScript no puede borrar una
// cookie httpOnly con document.cookie, por eso hace falta un endpoint que
// las sobreescriba con un maxAge negativo.
func (s *Server) handleLogout(c *gin.Context) {
	c.SetSameSite(http.SameSiteNoneMode)
	c.SetCookie(middleware.CookieToken, "", -1, "/", "", true, true)
	c.SetCookie(middleware.CookieCSRF, "", -1, "/", "", true, false)
	c.JSON(http.StatusOK, gin.H{"status": "Sesión cerrada"})
}

func (s *Server) handleVerificarPin(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req PinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de PIN inválido"})
		return
	}

	var pinDB string
	err := s.DBAuth.QueryRow("SELECT pin_admin FROM usuarios WHERE id = $1", userID).Scan(&pinDB)
	if err != nil {
		s.logError(c, "Error al verificar PIN", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al verificar PIN"})
		return
	}

	if strings.TrimSpace(pinDB) != strings.TrimSpace(req.Pin) {
		c.JSON(http.StatusUnauthorized, gin.H{"valid": false, "error": "PIN incorrecto"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":   true,
		"message": "PIN confirmado",
	})
}
