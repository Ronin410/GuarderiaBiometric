package server

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"biometrico/internal/middleware"
)

// PinRequest es el cuerpo esperado por /verificar-pin.
type PinRequest struct {
	Pin string `json:"pin"`
}

// duracionCookieSegundos es la vida de las cookies de sesión (biosafe_token
// y biosafe_csrf) — coincide con la expiración del JWT (24h) para que
// ambas cosas caduquen juntas.
const duracionCookieSegundos = 24 * 60 * 60

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
	}

	if err := c.ShouldBindJSON(&creds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido"})
		return
	}

	var id, gID int
	var passHash, rol, pinAdmin, gNombre, gSlug string
	var activo bool
	// Nota: pin_admin se consulta solo para mantener la forma de la fila; nunca se
	// expone en la respuesta. La verificación del PIN se hace en /verificar-pin.
	query := `
		SELECT
            u.id, u.guarderia_id, u.password_hash, u.rol, u.pin_admin,
            g.nombre, g.slug, u.activo
        FROM usuarios u
        INNER JOIN guarderias g ON u.guarderia_id = g.id
        WHERE u.username = $1`

	err := s.DBAuth.QueryRow(query, creds.Username).Scan(&id, &gID, &passHash, &rol, &pinAdmin, &gNombre, &gSlug, &activo)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Printf("Usuario no encontrado: %s\n", creds.Username)
			s.registrarAcceso("login_fallido", nil, nil, "usuario no encontrado: "+creds.Username, c.ClientIP())
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no existe"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error de BD"})
		}
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(passHash), []byte(creds.Password))
	if err != nil {
		log.Printf("Intento de login inválido para usuario %s", creds.Username)
		s.registrarAcceso("login_fallido", gID, id, "contraseña incorrecta", c.ClientIP())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Contraseña incorrecta"})
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
		log.Printf("Login rechazado (cuenta desactivada): %s", creds.Username)
		s.registrarAcceso("login_fallido", gID, id, "cuenta desactivada: "+creds.Username, c.ClientIP())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Esta cuenta está desactivada"})
		return
	}

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &middleware.Claims{
		UserID:      id,
		GuarderiaID: gID,
		Rol:         rol,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(s.JWTKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al generar token"})
		return
	}

	csrfToken, err := generarCSRFToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al generar token"})
		return
	}

	// SameSite=None porque frontend y backend viven en orígenes distintos
	// (y seguirían siendo "sitios" distintos aunque ambos vivieran bajo
	// onrender.com, que está en la Public Suffix List) — el navegador no
	// mandaría la cookie con Lax en las peticiones del frontend. None exige
	// Secure, que ya cumplimos (Render termina TLS en producción).
	c.SetSameSite(http.SameSiteNoneMode)
	c.SetCookie(middleware.CookieToken, tokenStr, duracionCookieSegundos, "/", "", true, true)
	c.SetCookie(middleware.CookieCSRF, csrfToken, duracionCookieSegundos, "/", "", true, false)

	log.Printf("Login exitoso: %s", creds.Username)
	s.registrarAcceso("login_exitoso", gID, id, creds.Username, c.ClientIP())
	c.JSON(http.StatusOK, gin.H{
		"user_id":          id,
		"guarderia_id":     gID,
		"guarderia_nombre": gNombre,
		"guarderia_slug":   gSlug,
		"rol":              rol,
		"username":         creds.Username,
		"expires_at":       expirationTime.Unix(),
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo cargar la sesión"})
		return
	}

	var expiraEn int64
	if expRaw, ok := c.Get("token_exp"); ok {
		if expTime, ok := expRaw.(time.Time); ok {
			expiraEn = expTime.Unix()
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
