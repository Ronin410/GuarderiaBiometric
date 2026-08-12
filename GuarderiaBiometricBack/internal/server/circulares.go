package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"biometrico/internal/middleware"
)

// Circular es un aviso que el admin o staff manda a todos los padres de la
// guardería (inscripciones, eventos, cierres, etc.), sin ligarse a ningún
// niño en particular.
type Circular struct {
	ID        int    `json:"id"`
	Titulo    string `json:"titulo"`
	Contenido string `json:"contenido"`
	CreadoEn  string `json:"creado_en"`
}

func (s *Server) registrarRutasCirculares(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)
	staff := middleware.RequireStaff()

	r.GET("/circulares", auth, staff, s.handleListarCirculares)
	r.POST("/circulares", auth, staff, s.handleCrearCircular)
	r.DELETE("/circulares/:id", auth, staff, s.handleEliminarCircular)
	// Mismo handler que arriba, pero solo detrás de Auth(): un papá también
	// debe poder leer los avisos que le mandan (mismo criterio que
	// /padre/menu-semanal), sin poder publicar ni borrar.
	r.GET("/padre/circulares", auth, s.handleListarCirculares)
}

func (s *Server) handleListarCirculares(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")

	rows, err := s.DB.Query(
		`SELECT id, titulo, contenido, creado_en FROM circulares
         WHERE guarderia_id = $1
         ORDER BY creado_en DESC
         LIMIT 50`,
		gID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar las circulares"})
		return
	}
	defer rows.Close()

	circulares := []Circular{}
	for rows.Next() {
		var cir Circular
		if err := rows.Scan(&cir.ID, &cir.Titulo, &cir.Contenido, &cir.CreadoEn); err != nil {
			continue
		}
		circulares = append(circulares, cir)
	}
	c.JSON(http.StatusOK, circulares)
}

func (s *Server) handleCrearCircular(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	userID, _ := c.Get("user_id")

	var input struct {
		Titulo    string `json:"titulo"`
		Contenido string `json:"contenido"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}
	titulo := strings.TrimSpace(input.Titulo)
	contenido := strings.TrimSpace(input.Contenido)
	if titulo == "" || contenido == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El título y el contenido son obligatorios"})
		return
	}

	var nuevoID int
	err := s.DB.QueryRow(
		`INSERT INTO circulares (guarderia_id, titulo, contenido, creado_por)
         VALUES ($1, $2, $3, $4) RETURNING id`,
		gID, titulo, contenido, userID,
	).Scan(&nuevoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo publicar la circular"})
		return
	}

	go s.notificarCircular(gID, titulo, contenido)

	c.JSON(http.StatusCreated, gin.H{"id": nuevoID, "message": "Circular publicada"})
}

func (s *Server) handleEliminarCircular(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	circularID := c.Param("id")

	res, err := s.DB.Exec(`DELETE FROM circulares WHERE id = $1 AND guarderia_id = $2`, circularID, gID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo eliminar la circular"})
		return
	}
	if filas, _ := res.RowsAffected(); filas == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Circular no encontrada"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Circular eliminada"})
}
