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

// CircularConLecturas es lo que ve el staff en el listado: la circular más
// cuántas familias ya la leyeron, sobre el total de familias de la
// guardería (ej. "5 de 8 familias").
type CircularConLecturas struct {
	Circular
	LeidoPor      int `json:"leido_por"`
	TotalFamilias int `json:"total_familias"`
}

// LecturaCircular es una fila del detalle "quién la ha leído" (staff).
type LecturaCircular struct {
	Nombre  string `json:"nombre"`
	LeidoEn string `json:"leido_en"`
}

func (s *Server) registrarRutasCirculares(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)
	staff := middleware.RequireArea("circulares")

	r.GET("/circulares", auth, staff, s.handleListarCircularesStaff)
	r.POST("/circulares", auth, staff, s.handleCrearCircular)
	r.DELETE("/circulares/:id", auth, staff, s.handleEliminarCircular)
	r.GET("/circulares/:id/lecturas", auth, staff, s.handleDetalleLecturasCircular)

	// Un papá también debe poder leer los avisos que le mandan (mismo
	// criterio que /padre/menu-semanal) y marcarlos como leídos, sin poder
	// publicar ni borrar.
	r.GET("/padre/circulares", auth, s.handleListarCircularesPadre)
	r.POST("/padre/circulares/:id/leido", auth, s.handleMarcarCircularLeida)
}

// handleListarCircularesStaff incluye, a diferencia de la vista del padre,
// cuántas familias ya leyeron cada circular -- el "verifica quiénes han
// leído tus mensajes" del PDF de referencia.
func (s *Server) handleListarCircularesStaff(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")

	rows, err := s.DB.Query(
		`SELECT c.id, c.titulo, c.contenido, c.creado_en,
                COUNT(cl.padre_id),
                (SELECT COUNT(*) FROM padres p WHERE p.guarderia_id = $1)
         FROM circulares c
         LEFT JOIN circulares_lecturas cl ON cl.circular_id = c.id
         WHERE c.guarderia_id = $1
         GROUP BY c.id
         ORDER BY c.creado_en DESC
         LIMIT 50`,
		gID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar las circulares"})
		return
	}
	defer rows.Close()

	circulares := []CircularConLecturas{}
	for rows.Next() {
		var cir CircularConLecturas
		if err := rows.Scan(&cir.ID, &cir.Titulo, &cir.Contenido, &cir.CreadoEn, &cir.LeidoPor, &cir.TotalFamilias); err != nil {
			continue
		}
		circulares = append(circulares, cir)
	}
	c.JSON(http.StatusOK, circulares)
}

// handleListarCircularesPadre es la vista simple que ya existía -- el padre
// NO se marca lector solo por listar (eso lo dispara el frontend, uno por
// uno, para los avisos que de verdad se le muestran en pantalla; ver
// handleMarcarCircularLeida).
func (s *Server) handleListarCircularesPadre(c *gin.Context) {
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

// handleMarcarCircularLeida la llama el frontend del padre por cada aviso
// que realmente se muestra en su pantalla (no en el simple listado) -- así
// el conteo que ve staff refleja avisos vistos de verdad, no solo
// consultados por la API.
func (s *Server) handleMarcarCircularLeida(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	userID, _ := c.Get("user_id")
	circularID := c.Param("id")

	var existe bool
	if err := s.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM circulares WHERE id = $1 AND guarderia_id = $2)`, circularID, gID).Scan(&existe); err != nil || !existe {
		c.JSON(http.StatusNotFound, gin.H{"error": "Circular no encontrada"})
		return
	}

	if _, err := s.DB.Exec(
		`INSERT INTO circulares_lecturas (circular_id, padre_id) VALUES ($1, $2) ON CONFLICT (circular_id, padre_id) DO NOTHING`,
		circularID, userID,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo registrar la lectura"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Marcada como leída"})
}

// handleDetalleLecturasCircular regresa la lista de familias que ya leyeron
// una circular (nombre + fecha), para el "quiénes" del staff.
func (s *Server) handleDetalleLecturasCircular(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	circularID := c.Param("id")

	var existe bool
	if err := s.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM circulares WHERE id = $1 AND guarderia_id = $2)`, circularID, gID).Scan(&existe); err != nil || !existe {
		c.JSON(http.StatusNotFound, gin.H{"error": "Circular no encontrada"})
		return
	}

	rows, err := s.DB.Query(
		`SELECT COALESCE(pa.nombre, 'Familia'), cl.leido_en
         FROM circulares_lecturas cl
         LEFT JOIN padres pa ON pa.id = cl.padre_id
         WHERE cl.circular_id = $1
         ORDER BY cl.leido_en DESC`,
		circularID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar las lecturas"})
		return
	}
	defer rows.Close()

	lecturas := []LecturaCircular{}
	for rows.Next() {
		var l LecturaCircular
		if err := rows.Scan(&l.Nombre, &l.LeidoEn); err != nil {
			continue
		}
		lecturas = append(lecturas, l)
	}
	c.JSON(http.StatusOK, lecturas)
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
