package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequireStaff bloquea el acceso a cuentas con rol "papa". Se usa después de
// AuthMiddleware() en endpoints que exponen datos de TODAS las familias de la
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
