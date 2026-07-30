package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Claims es la forma del JWT que emite /login y que AuthMiddleware valida en
// cada request autenticado.
type Claims struct {
	UserID      int    `json:"user_id"`
	GuarderiaID int    `json:"guarderia_id"`
	Rol         string `json:"rol"`
	jwt.RegisteredClaims
}

// Auth valida el JWT del header Authorization y publica guarderia_id/user_id/rol
// en el contexto de gin para que los handlers los lean con c.Get(...).
func Auth(jwtKey []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token requerido"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido o expirado"})
			c.Abort()
			return
		}

		c.Set("guarderia_id", claims.GuarderiaID)
		c.Set("user_id", claims.UserID)
		c.Set("rol", claims.Rol)
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
