package server

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"biometrico/internal/middleware"
)

// AusenciaPlanificada es un día en que un padre avisó que su hijo no
// asistirá -- "Planificación de ausencias por parte de los padres" del PDF
// de referencia.
type AusenciaPlanificada struct {
	ID       int    `json:"id"`
	HijoID   int    `json:"hijo_id"`
	Fecha    string `json:"fecha"`
	Motivo   string `json:"motivo"`
	CreadoEn string `json:"creado_en"`
}

// AusenciaConNino es lo que ve el staff: la ausencia más a qué niño
// pertenece, sin tener que consultar hijo por hijo.
type AusenciaConNino struct {
	AusenciaPlanificada
	HijoNombre string `json:"hijo_nombre"`
}

func (s *Server) registrarRutasAusencias(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)
	staff := middleware.RequireStaff()

	r.GET("/padre/hijos/:hijoId/ausencias", auth, s.handleListarAusenciasHijo)
	r.POST("/padre/hijos/:hijoId/ausencias", auth, s.handleCrearAusencia)
	r.DELETE("/padre/ausencias/:id", auth, s.handleCancelarAusencia)

	r.GET("/ausencias", auth, staff, s.handleListarAusenciasStaff)
}

// hijoPerteneceAPadre confirma que un niño está vinculado a la cuenta que
// hace la petición -- evita que un papá reporte o cancele ausencias de un
// niño que no es suyo con solo cambiar el id en la URL.
func (s *Server) hijoPerteneceAPadre(hijoID string, padreID any) bool {
	var esPropio bool
	err := s.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM tutor_hijos WHERE hijo_id = $1 AND padre_id = $2)`, hijoID, padreID).Scan(&esPropio)
	return err == nil && esPropio
}

func (s *Server) handleListarAusenciasHijo(c *gin.Context) {
	userID, _ := c.Get("user_id")
	hijoID := c.Param("hijoId")

	if !s.hijoPerteneceAPadre(hijoID, userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "No tienes permiso para ver las ausencias de este niño"})
		return
	}

	rows, err := s.DB.Query(
		`SELECT id, hijo_id, fecha, COALESCE(motivo, ''), creado_en
         FROM ausencias_planificadas
         WHERE hijo_id = $1 AND fecha >= CURRENT_DATE
         ORDER BY fecha ASC`,
		hijoID,
	)
	if err != nil {
		log.Printf("Error al consultar las ausencias: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar las ausencias"})
		return
	}
	defer rows.Close()

	ausencias := []AusenciaPlanificada{}
	for rows.Next() {
		var a AusenciaPlanificada
		var fecha, creadoEn *string
		if err := rows.Scan(&a.ID, &a.HijoID, &fecha, &a.Motivo, &creadoEn); err != nil {
			continue
		}
		if f := soloFecha(fecha); f != nil {
			a.Fecha = *f
		}
		if creadoEn != nil {
			a.CreadoEn = *creadoEn
		}
		ausencias = append(ausencias, a)
	}
	c.JSON(http.StatusOK, ausencias)
}

func (s *Server) handleCrearAusencia(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	userID, _ := c.Get("user_id")
	hijoID := c.Param("hijoId")

	if !s.hijoPerteneceAPadre(hijoID, userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "No tienes permiso para reportar ausencias de este niño"})
		return
	}

	var input struct {
		FechaInicio string `json:"fecha_inicio"`
		FechaFin    string `json:"fecha_fin"`
		Motivo      string `json:"motivo"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	inicio, err := time.Parse("2006-01-02", input.FechaInicio)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Fecha de inicio inválida"})
		return
	}
	fin := inicio
	finStr := strings.TrimSpace(input.FechaFin)
	if finStr != "" {
		fin, err = time.Parse("2006-01-02", finStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Fecha final inválida"})
			return
		}
	}
	if fin.Before(inicio) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "La fecha final no puede ser anterior a la fecha de inicio"})
		return
	}
	if input.FechaInicio < hoyEnZonaLocal(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No se pueden reportar ausencias en fechas pasadas"})
		return
	}
	// Tope de 31 días por reporte -- evita que un rango mal tecleado (ej. año
	// equivocado) inserte cientos de filas sin querer.
	dias := int(fin.Sub(inicio).Hours()/24) + 1
	if dias > 31 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El rango no puede ser mayor a 31 días"})
		return
	}

	motivo := strings.TrimSpace(input.Motivo)
	for d := inicio; !d.After(fin); d = d.AddDate(0, 0, 1) {
		// ON CONFLICT: si el padre ya había reportado ese día, actualiza el
		// motivo en vez de fallar -- reportar de nuevo el mismo día es más
		// fácil de entender como "corregir" que como error.
		if _, err := s.DB.Exec(
			`INSERT INTO ausencias_planificadas (hijo_id, guarderia_id, fecha, motivo, creado_por)
             VALUES ($1, $2, $3, $4, $5)
             ON CONFLICT (hijo_id, fecha) DO UPDATE SET motivo = EXCLUDED.motivo`,
			hijoID, gID, d.Format("2006-01-02"), motivo, userID,
		); err != nil {
			log.Printf("No se pudo guardar la ausencia del hijo %v el día %s: %v", hijoID, d.Format("2006-01-02"), err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudieron guardar las ausencias"})
			return
		}
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Ausencia registrada", "dias": dias})
}

func (s *Server) handleCancelarAusencia(c *gin.Context) {
	userID, _ := c.Get("user_id")
	id := c.Param("id")

	// creado_por = userID ya limita esto a los propios reportes del padre, sin
	// necesitar otra vuelta a tutor_hijos. fecha >= CURRENT_DATE evita
	// "des-reportar" retroactivamente una ausencia ya pasada.
	res, err := s.DB.Exec(
		`DELETE FROM ausencias_planificadas WHERE id = $1 AND creado_por = $2 AND fecha >= CURRENT_DATE`,
		id, userID,
	)
	if err != nil {
		log.Printf("No se pudo cancelar la ausencia: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo cancelar la ausencia"})
		return
	}
	if filas, _ := res.RowsAffected(); filas == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ausencia no encontrada"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Ausencia cancelada"})
}

// handleListarAusenciasStaff regresa las ausencias planificadas de toda la
// guardería en un rango de fechas (por defecto, solo hoy) -- para que staff
// sepa de antemano quién no viene.
func (s *Server) handleListarAusenciasStaff(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")

	desde := strings.TrimSpace(c.Query("desde"))
	if desde == "" {
		desde = hoyEnZonaLocal(time.Now())
	}
	hasta := strings.TrimSpace(c.Query("hasta"))
	if hasta == "" {
		hasta = desde
	}

	rows, err := s.DB.Query(
		`SELECT a.id, a.hijo_id, a.fecha, COALESCE(a.motivo, ''), a.creado_en, h.nombre_niño
         FROM ausencias_planificadas a
         JOIN hijos h ON h.id = a.hijo_id
         WHERE a.guarderia_id = $1 AND a.fecha BETWEEN $2 AND $3
         ORDER BY a.fecha ASC, h.nombre_niño ASC`,
		gID, desde, hasta,
	)
	if err != nil {
		log.Printf("Error al consultar las ausencias: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar las ausencias"})
		return
	}
	defer rows.Close()

	ausencias := []AusenciaConNino{}
	for rows.Next() {
		var a AusenciaConNino
		var fecha, creadoEn *string
		if err := rows.Scan(&a.ID, &a.HijoID, &fecha, &a.Motivo, &creadoEn, &a.HijoNombre); err != nil {
			continue
		}
		if f := soloFecha(fecha); f != nil {
			a.Fecha = *f
		}
		if creadoEn != nil {
			a.CreadoEn = *creadoEn
		}
		ausencias = append(ausencias, a)
	}
	c.JSON(http.StatusOK, ausencias)
}
