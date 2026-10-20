package server

import (
	"github.com/gin-gonic/gin"

	"biometrico/internal/applog"
)

// logError es el reemplazo estandarizado de los antiguos "log.Printf(...)"
// sueltos en cada handler -- agrega automáticamente el contexto de la
// petición (ruta, guardería, usuario) cuando existe, para no tener que
// repetirlo a mano en cada mensaje como antes. c puede ser nil: las tareas
// de fondo sin *gin.Context (ver push.go, soporte.go) siguen pudiendo
// loguear, nada más sin ese contexto de petición.
func (s *Server) logError(c *gin.Context, msg string, err error, args ...any) {
	campos := append([]any{}, args...)
	if c != nil {
		campos = append(campos, "path", c.FullPath(), "method", c.Request.Method)
		if gID, ok := c.Get("guarderia_id"); ok {
			campos = append(campos, "guarderia_id", gID)
		}
		if uID, ok := c.Get("user_id"); ok {
			campos = append(campos, "user_id", uID)
		}
	}
	applog.Error(msg, err, campos...)
}
