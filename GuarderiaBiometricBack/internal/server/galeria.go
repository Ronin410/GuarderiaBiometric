package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"biometrico/internal/middleware"
)

// FotoGaleria es una foto de la bitácora con su fecha, ya con URL firmada
// lista para mostrarse -- "Galería de fotos" del PDF de referencia: junta
// en una sola vista todas las fotos que ya se suben día a día desde la
// bitácora (fotos_seguimiento), en vez de tener que revisar fecha por
// fecha. No incluye video: la app hoy solo sube imágenes (uploadToS3 fija
// el content-type a image/jpeg en handleGuardarSeguimiento), así que el
// alcance de esta vista es fotos.
type FotoGaleria struct {
	ID    int    `json:"id"`
	URL   string `json:"url"`
	Fecha string `json:"fecha"`
}

func (s *Server) registrarRutasGaleria(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)
	staff := middleware.RequireStaff()

	r.GET("/hijos/:id/galeria", auth, staff, s.handleGaleriaStaff)
	r.GET("/padre/hijos/:hijoId/galeria", auth, s.handleGaleriaPadre)
}

func (s *Server) handleGaleriaStaff(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	hijoID := c.Param("id")

	if !s.hijoPerteneceAGuarderia(hijoID, gID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Niño no encontrado"})
		return
	}

	fotos, err := s.obtenerGaleria(hijoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar la galería"})
		return
	}
	c.JSON(http.StatusOK, fotos)
}

func (s *Server) handleGaleriaPadre(c *gin.Context) {
	userID, _ := c.Get("user_id")
	hijoID := c.Param("hijoId")

	if !s.hijoPerteneceAPadre(hijoID, userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "No tienes permiso para ver la galería de este niño"})
		return
	}

	fotos, err := s.obtenerGaleria(hijoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar la galería"})
		return
	}
	c.JSON(http.StatusOK, fotos)
}

// obtenerGaleria trae TODAS las fotos de un niño, de todas las bitácoras
// diarias, más recientes primero -- compartido entre la vista de staff y
// la del padre, solo cambia cómo se verificó el permiso para llegar aquí.
func (s *Server) obtenerGaleria(hijoID string) ([]FotoGaleria, error) {
	rows, err := s.DB.Query(
		`SELECT f.id, f.url, s.fecha
         FROM fotos_seguimiento f
         JOIN seguimiento_diario s ON s.id = f.seguimiento_id
         WHERE s.hijo_id = $1
         ORDER BY s.fecha DESC, f.id DESC`,
		hijoID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fotos := []FotoGaleria{}
	for rows.Next() {
		var id int
		var valorGuardado string
		var fecha *string
		if err := rows.Scan(&id, &valorGuardado, &fecha); err != nil {
			continue
		}
		urlFirmada, errFirma := s.firmarURLFoto(valorGuardado)
		if errFirma != nil {
			continue
		}
		foto := FotoGaleria{ID: id, URL: urlFirmada}
		if f := soloFecha(fecha); f != nil {
			foto.Fecha = *f
		}
		fotos = append(fotos, foto)
	}
	return fotos, nil
}
