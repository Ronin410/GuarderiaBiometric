package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"biometrico/internal/middleware"
)

func (s *Server) registrarRutasPrivacidad(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)
	staff := middleware.RequireArea("configuracion")
	r.GET("/aviso-privacidad", auth, s.handleObtenerAviso)
	r.PUT("/admin/aviso-privacidad", auth, staff, s.handleActualizarAviso)
	r.GET("/admin/aviso-privacidad/estadisticas", auth, staff, s.handleEstadisticasAviso)
}

func (s *Server) handleObtenerAviso(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")

	var texto, version string
	err := s.DB.QueryRow(
		"SELECT COALESCE(aviso_privacidad_texto, ''), aviso_privacidad_version FROM guarderias WHERE id = $1",
		gID,
	).Scan(&texto, &version)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo consultar el Aviso de Privacidad"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"texto":       texto,
		"version":     version,
		"configurado": strings.TrimSpace(texto) != "",
	})
}

func (s *Server) handleActualizarAviso(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")

	var input struct {
		Texto string `json:"texto"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Texto) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El texto del Aviso de Privacidad es requerido"})
		return
	}

	var versionActual string
	if err := s.DB.QueryRow("SELECT aviso_privacidad_version FROM guarderias WHERE id = $1", gID).Scan(&versionActual); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo consultar la versión actual"})
		return
	}

	nuevaVersion := siguienteVersion(versionActual)

	_, err := s.DB.Exec(
		"UPDATE guarderias SET aviso_privacidad_texto = $1, aviso_privacidad_version = $2 WHERE id = $3",
		input.Texto, nuevaVersion, gID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo guardar el Aviso de Privacidad"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "Aviso de Privacidad actualizado", "version": nuevaVersion})
}

func (s *Server) handleEstadisticasAviso(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")

	var total int
	err := s.DB.QueryRow(
		"SELECT COUNT(*) FROM consentimientos WHERE guarderia_id = $1",
		gID,
	).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo consultar los consentimientos"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total_consentimientos": total})
}

// siguienteVersion incrementa una versión con forma "vN" (ej. "v1" -> "v2").
// Si el formato no es el esperado (dato viejo/manual), reinicia en "v1" en
// vez de fallar — el punto es que cada guardado quede en una versión nueva
// y distinguible, no reconstruir un historial perfecto de versiones previas.
func siguienteVersion(actual string) string {
	numero, err := strconv.Atoi(strings.TrimPrefix(actual, "v"))
	if err != nil {
		return "v1"
	}
	return "v" + strconv.Itoa(numero+1)
}
