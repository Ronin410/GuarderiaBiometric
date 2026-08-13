package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"biometrico/internal/middleware"
)

// tiposEventoValidos son los tipos que acepta eventos_calendario.tipo (mismo
// juego que el CHECK de la migración) -- se validan aquí también para poder
// devolver un 400 claro en vez de que el INSERT truene con un error de
// Postgres genérico.
var tiposEventoValidos = map[string]bool{
	"evento":     true,
	"suspension": true,
	"vacaciones": true,
	"junta":      true,
}

// EventoCalendario es una fecha importante del centro (suspensión de
// clases, actividad, junta de padres, vacaciones) -- "Calendario escolar"
// del PDF de referencia. FechaFin es opcional: nil significa evento de un
// solo día.
type EventoCalendario struct {
	ID          int     `json:"id"`
	Titulo      string  `json:"titulo"`
	Descripcion string  `json:"descripcion"`
	FechaInicio string  `json:"fecha_inicio"`
	FechaFin    *string `json:"fecha_fin"`
	Tipo        string  `json:"tipo"`
	CreadoEn    string  `json:"creado_en"`
}

func (s *Server) registrarRutasCalendario(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)
	staff := middleware.RequireStaff()

	r.GET("/calendario", auth, staff, s.handleListarCalendario)
	r.POST("/calendario", auth, staff, s.handleCrearEventoCalendario)
	r.DELETE("/calendario/:id", auth, staff, s.handleEliminarEventoCalendario)

	// Mismo handler que arriba, solo detrás de Auth(): un papá también debe
	// poder consultar el calendario (mismo criterio que /padre/circulares),
	// sin poder publicar ni borrar.
	r.GET("/padre/calendario", auth, s.handleListarCalendario)
}

// handleListarCalendario regresa los eventos que se traslapan con el rango
// [desde, hasta] -- no solo los que EMPIEZAN en el rango, para que un
// evento de varios días (ej. vacaciones) siga apareciendo aunque ya haya
// comenzado. Por defecto: hoy hasta 90 días adelante (un calendario escolar
// se consulta con más anticipación que, por ejemplo, las ausencias).
func (s *Server) handleListarCalendario(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")

	desde := strings.TrimSpace(c.Query("desde"))
	if desde == "" {
		desde = hoyEnZonaLocal(time.Now())
	}
	hasta := strings.TrimSpace(c.Query("hasta"))
	if hasta == "" {
		inicio, err := time.Parse("2006-01-02", desde)
		if err != nil {
			inicio = time.Now()
		}
		hasta = inicio.AddDate(0, 0, 90).Format("2006-01-02")
	}

	rows, err := s.DB.Query(
		`SELECT id, titulo, COALESCE(descripcion, ''), fecha_inicio, fecha_fin, tipo, creado_en
         FROM eventos_calendario
         WHERE guarderia_id = $1 AND fecha_inicio <= $3 AND COALESCE(fecha_fin, fecha_inicio) >= $2
         ORDER BY fecha_inicio ASC`,
		gID, desde, hasta,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar el calendario"})
		return
	}
	defer rows.Close()

	eventos := []EventoCalendario{}
	for rows.Next() {
		var ev EventoCalendario
		var fechaInicio, fechaFin, creadoEn *string
		if err := rows.Scan(&ev.ID, &ev.Titulo, &ev.Descripcion, &fechaInicio, &fechaFin, &ev.Tipo, &creadoEn); err != nil {
			continue
		}
		if f := soloFecha(fechaInicio); f != nil {
			ev.FechaInicio = *f
		}
		ev.FechaFin = soloFecha(fechaFin)
		if creadoEn != nil {
			ev.CreadoEn = *creadoEn
		}
		eventos = append(eventos, ev)
	}
	c.JSON(http.StatusOK, eventos)
}

func (s *Server) handleCrearEventoCalendario(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	userID, _ := c.Get("user_id")

	var input struct {
		Titulo      string `json:"titulo"`
		Descripcion string `json:"descripcion"`
		FechaInicio string `json:"fecha_inicio"`
		FechaFin    string `json:"fecha_fin"`
		Tipo        string `json:"tipo"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	titulo := strings.TrimSpace(input.Titulo)
	if titulo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El título es obligatorio"})
		return
	}

	inicio, err := time.Parse("2006-01-02", input.FechaInicio)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Fecha de inicio inválida"})
		return
	}

	var finPtr *string
	finStr := strings.TrimSpace(input.FechaFin)
	if finStr != "" {
		fin, err := time.Parse("2006-01-02", finStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Fecha final inválida"})
			return
		}
		if fin.Before(inicio) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "La fecha final no puede ser anterior a la fecha de inicio"})
			return
		}
		finPtr = &finStr
	}

	tipo := strings.TrimSpace(input.Tipo)
	if tipo == "" {
		tipo = "evento"
	}
	if !tiposEventoValidos[tipo] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tipo de evento inválido"})
		return
	}

	var nuevoID int
	err = s.DB.QueryRow(
		`INSERT INTO eventos_calendario (guarderia_id, titulo, descripcion, fecha_inicio, fecha_fin, tipo, creado_por)
         VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		gID, titulo, strings.TrimSpace(input.Descripcion), input.FechaInicio, finPtr, tipo, userID,
	).Scan(&nuevoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo crear el evento"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": nuevoID, "message": "Evento creado"})
}

func (s *Server) handleEliminarEventoCalendario(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	eventoID := c.Param("id")

	res, err := s.DB.Exec(`DELETE FROM eventos_calendario WHERE id = $1 AND guarderia_id = $2`, eventoID, gID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo eliminar el evento"})
		return
	}
	if filas, _ := res.RowsAffected(); filas == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Evento no encontrado"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Evento eliminado"})
}
