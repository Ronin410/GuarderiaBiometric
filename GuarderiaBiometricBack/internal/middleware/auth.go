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
// request autenticado.
type Claims struct {
	UserID      int    `json:"user_id"`
	GuarderiaID int    `json:"guarderia_id"`
	Rol         string `json:"rol"`
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
