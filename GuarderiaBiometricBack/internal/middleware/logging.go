package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"biometrico/internal/applog"
)

// StructuredLogger reemplaza el logger de texto plano por defecto de Gin
// (gin.Logger(), el que trae gin.Default()) por uno en el mismo formato
// JSON que el resto del logging de la app (ver internal/applog). Es la
// garantía de fondo de "quiero saber si hay un error más fácil": toda
// petición que responde 4xx/5xx queda registrada con su nivel correcto
// SIEMPRE, la haya logueado o no explícitamente el handler que la atendió
// -- no depende de que cada uno de los ~200 handlers se acuerde de llamar a
// logError.
func StructuredLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		inicio := time.Now()
		c.Next()

		status := c.Writer.Status()
		campos := []any{
			"method", c.Request.Method,
			"path", c.FullPath(),
			"status", status,
			"duracion_ms", time.Since(inicio).Milliseconds(),
			"ip", c.ClientIP(),
		}
		if gID, ok := c.Get("guarderia_id"); ok {
			campos = append(campos, "guarderia_id", gID)
		}

		switch {
		case status >= 500:
			applog.Error("petición con error de servidor", ultimoErrorGin(c), campos...)
		case status >= 400:
			applog.Warn("petición rechazada", campos...)
		default:
			applog.Info("petición", campos...)
		}
	}
}

// ultimoErrorGin regresa el último error que algún handler haya agregado
// vía c.Error(...) -- la mayoría de los handlers de esta app responden el
// error directo con c.JSON() en vez de c.Error(), así que casi siempre esto
// da nil; cuando sí hay uno, es información extra gratis para el log.
func ultimoErrorGin(c *gin.Context) error {
	if len(c.Errors) > 0 {
		return c.Errors.Last()
	}
	return nil
}
