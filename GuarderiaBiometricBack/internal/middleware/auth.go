package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// CookieToken guarda el JWT en una cookie httpOnly (invisible a JavaScript,
// a diferencia del localStorage que se usaba antes) — protege contra robo
// de sesión vía XSS, relevante porque esta app maneja datos biométricos y
// de menores. CookieCSRF es su contraparte NO httpOnly: el frontend sí
// puede leerla y la reenvía en el header X-CSRF-Token en cada petición que
// modifica datos, para el patrón "double-submit cookie" — sin esto, una
// cookie de sesión que el navegador adjunta solo (SameSite=None, necesario
// porque frontend y backend están en orígenes distintos) sería vulnerable a
// CSRF, algo de lo que el token en un header manejado por JS estaba libre
// sin querer.
const (
	CookieToken = "biosafe_token"
	CookieCSRF  = "biosafe_csrf"
)

// Claims es la forma del JWT que emite /login y que Auth() valida en cada
// request autenticado. Permisos va con puntero + omitempty a propósito: nil
// (el JSON del token ni siquiera trae el campo) distingue "esta cuenta de
// staff no tiene permisos personalizados, acceso completo de siempre" de un
// slice vacío "*[]string{}", que significa "sin acceso a ninguna sección
// protegida" — ver RequireArea.
type Claims struct {
	UserID      int       `json:"user_id"`
	GuarderiaID int       `json:"guarderia_id"`
	Rol         string    `json:"rol"`
	Permisos    *[]string `json:"permisos,omitempty"`
	jwt.RegisteredClaims
}

// Auth valida el JWT de la cookie CookieToken y publica guarderia_id/user_id/
// rol/token_exp en el contexto de gin para que los handlers los lean con
// c.Get(...). En peticiones que modifican datos (todo menos GET/HEAD/OPTIONS)
// también exige que el header X-CSRF-Token coincida con la cookie CookieCSRF.
func Auth(jwtKey []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := c.Cookie(CookieToken)
		if err != nil || tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesión requerida"})
			c.Abort()
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido o expirado"})
			c.Abort()
			return
		}

		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead && c.Request.Method != http.MethodOptions {
			csrfCookie, errCookie := c.Cookie(CookieCSRF)
			csrfHeader := c.GetHeader("X-CSRF-Token")
			if errCookie != nil || csrfCookie == "" || csrfHeader == "" || csrfCookie != csrfHeader {
				c.JSON(http.StatusForbidden, gin.H{"error": "Token CSRF inválido"})
				c.Abort()
				return
			}
		}

		c.Set("guarderia_id", claims.GuarderiaID)
		c.Set("user_id", claims.UserID)
		c.Set("rol", claims.Rol)
		if claims.ExpiresAt != nil {
			c.Set("token_exp", claims.ExpiresAt.Time)
		}
		// Solo se publica en el contexto si la cuenta tiene permisos
		// personalizados (claims.Permisos != nil) — RequireArea distingue
		// "la clave 'permisos' no existe en el contexto" (sin personalizar,
		// acceso completo) de "existe pero está vacía" (sin acceso a nada)
		// con c.Get(), así que no hay que poner un valor centinela.
		if claims.Permisos != nil {
			c.Set("permisos", *claims.Permisos)
		}
		c.Next()
	}
}

// RequireStaff bloquea el acceso a cuentas con rol "papa". Se usa después de
// Auth() en endpoints que exponen datos de TODAS las familias de la
// guardería (domicilios, contactos de emergencia, pagos), para que una cuenta
// de padre no pueda leerlos llamando la ruta directo, sin pasar por la UI.
func RequireStaff() gin.HandlerFunc {
	return func(c *gin.Context) {
		rol, _ := c.Get("rol")
		if rol != "admin" && rol != "staff" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Acceso restringido al personal de la guardería"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireArea exige acceso a una sección concreta (ej. "pagos", "reportes")
// en vez de a todo el panel por igual como RequireStaff — es "permisos
// personalizados por docente": antes, cualquier staff que conociera SU
// PROPIO PIN (por diseño, /verificar-pin ya lo compara contra el pin_admin
// de quien está logueado, no uno compartido) desbloqueaba TODAS las
// secciones protegidas del frontend por igual, y el backend no volvía a
// revisar nada — cualquier cuenta de staff podía llamar esas rutas
// directamente sin pasar por el PIN. RequireArea sí lo exige en el backend:
//   - admin: pasa siempre (control total, igual que con RequireStaff).
//   - staff sin permisos personalizados (la cuenta no tiene nada
//     configurado todavía): pasa siempre — mismo comportamiento de hoy,
//     para no romper ninguna cuenta de golpe al desplegar esto.
//   - staff con permisos personalizados: pasa solo si `area` está en su
//     lista.
//   - papa: nunca pasa, igual que RequireStaff.
func RequireArea(area string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rol, _ := c.Get("rol")
		if rol == "admin" {
			c.Next()
			return
		}
		if rol != "staff" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Acceso restringido al personal de la guardería"})
			c.Abort()
			return
		}

		permisosRaw, personalizado := c.Get("permisos")
		if !personalizado {
			c.Next()
			return
		}
		permisos, _ := permisosRaw.([]string)
		for _, p := range permisos {
			if p == area {
				c.Next()
				return
			}
		}
		c.JSON(http.StatusForbidden, gin.H{"error": "Tu cuenta no tiene permiso para acceder a esta sección"})
		c.Abort()
	}
}

// RequireAdmin bloquea el acceso a cualquier cuenta que no sea "admin" — a
// diferencia de RequireStaff (que solo excluye "papa"), esto también excluye
// "staff". Se usa en la gestión de personal: dejar que una cuenta de staff
// cree otras cuentas, se reasigne el rol admin a sí misma, o cambie el PIN
// de alguien más sería un escalamiento de privilegios.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		rol, _ := c.Get("rol")
		if rol != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Acceso restringido al administrador de la guardería"})
			c.Abort()
			return
		}
		c.Next()
	}
}
