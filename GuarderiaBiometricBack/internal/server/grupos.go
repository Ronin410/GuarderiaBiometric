package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"biometrico/internal/middleware"
)

// Grupo es un salón/agrupación de niños dentro de una guardería (ej. "Sala
// Maternal", "Preescolar A"). NinosActivos ayuda al panel a advertir antes
// de borrar un grupo que todavía tiene niños asignados.
type Grupo struct {
	ID           int    `json:"id"`
	Nombre       string `json:"nombre"`
	NinosActivos int    `json:"ninos_activos"`
}

func (s *Server) registrarRutasGrupos(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)
	staff := middleware.RequireStaff()

	r.GET("/admin/grupos", auth, staff, s.handleListarGrupos)
	r.POST("/admin/grupos", auth, staff, s.handleCrearGrupo)
	r.PUT("/admin/grupos/:id", auth, staff, s.handleRenombrarGrupo)
	r.DELETE("/admin/grupos/:id", auth, staff, s.handleEliminarGrupo)
}

func (s *Server) handleListarGrupos(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")

	rows, err := s.DB.Query(
		`SELECT g.id, g.nombre, COUNT(h.id) FILTER (WHERE h.activo)
         FROM grupos g
         LEFT JOIN hijos h ON h.grupo_id = g.id
         WHERE g.guarderia_id = $1
         GROUP BY g.id
         ORDER BY g.nombre ASC`,
		gID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar grupos"})
		return
	}
	defer rows.Close()

	grupos := []Grupo{}
	for rows.Next() {
		var g Grupo
		if err := rows.Scan(&g.ID, &g.Nombre, &g.NinosActivos); err != nil {
			continue
		}
		grupos = append(grupos, g)
	}
	c.JSON(http.StatusOK, grupos)
}

func (s *Server) handleCrearGrupo(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")

	var input struct {
		Nombre string `json:"nombre"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Nombre) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El nombre del grupo es obligatorio"})
		return
	}

	var nuevoID int
	err := s.DB.QueryRow(
		`INSERT INTO grupos (guarderia_id, nombre) VALUES ($1, $2) RETURNING id`,
		gID, strings.TrimSpace(input.Nombre),
	).Scan(&nuevoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo crear el grupo"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": nuevoID, "message": "Grupo creado"})
}

func (s *Server) handleRenombrarGrupo(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	grupoID := c.Param("id")

	var input struct {
		Nombre string `json:"nombre"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Nombre) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El nombre del grupo es obligatorio"})
		return
	}

	res, err := s.DB.Exec(
		`UPDATE grupos SET nombre = $1 WHERE id = $2 AND guarderia_id = $3`,
		strings.TrimSpace(input.Nombre), grupoID, gID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo renombrar el grupo"})
		return
	}
	if filas, _ := res.RowsAffected(); filas == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Grupo no encontrado"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Grupo renombrado"})
}

// handleEliminarGrupo rechaza el borrado si el grupo todavía tiene niños
// activos asignados — obliga a reasignarlos primero en vez de dejarlos
// "sin grupo" en silencio (mismo criterio de "nada desaparece sin un paso
// explícito" que ya usa el resto del panel con las bajas lógicas).
func (s *Server) handleEliminarGrupo(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	grupoID := c.Param("id")

	var conNinos int
	err := s.DB.QueryRow(
		`SELECT COUNT(*) FROM hijos WHERE grupo_id = $1 AND activo`,
		grupoID,
	).Scan(&conNinos)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al verificar el grupo"})
		return
	}
	if conNinos > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"error":         "Este grupo todavía tiene niños asignados. Reasígnalos a otro grupo antes de eliminarlo.",
			"ninos_activos": conNinos,
		})
		return
	}

	res, err := s.DB.Exec(`DELETE FROM grupos WHERE id = $1 AND guarderia_id = $2`, grupoID, gID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo eliminar el grupo"})
		return
	}
	if filas, _ := res.RowsAffected(); filas == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Grupo no encontrado"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Grupo eliminado"})
}
