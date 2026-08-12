package server

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"biometrico/internal/middleware"
)

// TurnoDia es el horario esperado de un día de la semana (0=lunes..6=domingo)
// para una cuenta de personal. HoraEntrada/HoraSalida van en puntero: un día
// sin turno asignado se distingue de uno con horario capturado.
type TurnoDia struct {
	DiaSemana   int     `json:"dia_semana"`
	HoraEntrada *string `json:"hora_entrada"`
	HoraSalida  *string `json:"hora_salida"`
}

// RegistroHoras es una fila del control de horas trabajadas REAL (a
// diferencia de TurnoDia, que es el plan) -- para nómina, puede diferir del
// turno por faltas, horas extra, etc.
type RegistroHoras struct {
	ID              int     `json:"id"`
	Fecha           string  `json:"fecha"`
	HorasTrabajadas float64 `json:"horas_trabajadas"`
	Observaciones   string  `json:"observaciones"`
}

func (s *Server) registrarRutasHorarios(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)
	// RequireAdmin, no RequireStaff: esto es información ligada a nómina
	// (horas trabajadas, turnos) del mismo personal -- mismo nivel de
	// permiso que la Gestión de Personal (Fase 1.2), no el de operación
	// diaria que sí puede tocar cualquier staff.
	admin := middleware.RequireAdmin()

	r.GET("/admin/horarios/:usuarioId", auth, admin, s.handleObtenerTurnos)
	r.PUT("/admin/horarios/:usuarioId/:diaSemana", auth, admin, s.handleGuardarTurno)

	r.GET("/admin/horas/:usuarioId", auth, admin, s.handleListarHoras)
	r.PUT("/admin/horas/:usuarioId/:fecha", auth, admin, s.handleGuardarHoras)
	r.DELETE("/admin/horas/:usuarioId/:fecha", auth, admin, s.handleEliminarHoras)
}

// personalPerteneceAGuarderia evita que un admin de UNA guardería lea o
// modifique el horario/horas de una cuenta de personal de OTRA guardería con
// solo cambiar el id en la URL. Solo cuenta personal (admin/staff), no papás.
func (s *Server) personalPerteneceAGuarderia(usuarioID string, gID any) bool {
	var existe bool
	err := s.DBAuth.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM usuarios WHERE id = $1 AND guarderia_id = $2 AND rol IN ('admin', 'staff'))`,
		usuarioID, gID,
	).Scan(&existe)
	return err == nil && existe
}

func (s *Server) handleObtenerTurnos(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	usuarioID := c.Param("usuarioId")

	if !s.personalPerteneceAGuarderia(usuarioID, gID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cuenta de personal no encontrada"})
		return
	}

	rows, err := s.DBAuth.Query(
		`SELECT dia_semana, hora_entrada, hora_salida FROM horarios_personal WHERE usuario_id = $1`,
		usuarioID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar el horario"})
		return
	}
	defer rows.Close()

	existentes := map[int]TurnoDia{}
	for rows.Next() {
		var t TurnoDia
		if err := rows.Scan(&t.DiaSemana, &t.HoraEntrada, &t.HoraSalida); err != nil {
			continue
		}
		t.HoraEntrada = soloHora(t.HoraEntrada)
		t.HoraSalida = soloHora(t.HoraSalida)
		existentes[t.DiaSemana] = t
	}

	turnos := make([]TurnoDia, 7)
	for dia := 0; dia < 7; dia++ {
		if t, ok := existentes[dia]; ok {
			turnos[dia] = t
		} else {
			turnos[dia] = TurnoDia{DiaSemana: dia}
		}
	}
	c.JSON(http.StatusOK, turnos)
}

func (s *Server) handleGuardarTurno(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	usuarioID := c.Param("usuarioId")

	dia, err := strconv.Atoi(c.Param("diaSemana"))
	if err != nil || dia < 0 || dia > 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Día de la semana inválido (0-6)"})
		return
	}
	if !s.personalPerteneceAGuarderia(usuarioID, gID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cuenta de personal no encontrada"})
		return
	}

	var input struct {
		HoraEntrada *string `json:"hora_entrada"`
		HoraSalida  *string `json:"hora_salida"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	_, err = s.DBAuth.Exec(`
        INSERT INTO horarios_personal (usuario_id, guarderia_id, dia_semana, hora_entrada, hora_salida)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (usuario_id, dia_semana) DO UPDATE SET
            hora_entrada = EXCLUDED.hora_entrada,
            hora_salida = EXCLUDED.hora_salida`,
		usuarioID, gID, dia, input.HoraEntrada, input.HoraSalida,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo guardar el turno"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Turno guardado"})
}

func (s *Server) handleListarHoras(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	usuarioID := c.Param("usuarioId")
	inicio := c.Query("inicio")
	fin := c.Query("fin")

	if !s.personalPerteneceAGuarderia(usuarioID, gID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cuenta de personal no encontrada"})
		return
	}
	if _, err := time.Parse("2006-01-02", inicio); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parámetro 'inicio' inválido (usa YYYY-MM-DD)"})
		return
	}
	if _, err := time.Parse("2006-01-02", fin); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parámetro 'fin' inválido (usa YYYY-MM-DD)"})
		return
	}

	rows, err := s.DBAuth.Query(
		`SELECT id, fecha, horas_trabajadas, COALESCE(observaciones, '') FROM registro_horas
         WHERE usuario_id = $1 AND fecha BETWEEN $2 AND $3
         ORDER BY fecha ASC`,
		usuarioID, inicio, fin,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar el registro de horas"})
		return
	}
	defer rows.Close()

	registros := []RegistroHoras{}
	for rows.Next() {
		var reg RegistroHoras
		var fechaRaw *string
		if err := rows.Scan(&reg.ID, &fechaRaw, &reg.HorasTrabajadas, &reg.Observaciones); err != nil {
			continue
		}
		if f := soloFecha(fechaRaw); f != nil {
			reg.Fecha = *f
		}
		registros = append(registros, reg)
	}
	c.JSON(http.StatusOK, registros)
}

func (s *Server) handleGuardarHoras(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	usuarioID := c.Param("usuarioId")
	fecha := c.Param("fecha")

	if _, err := time.Parse("2006-01-02", fecha); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Fecha inválida (usa YYYY-MM-DD)"})
		return
	}
	if !s.personalPerteneceAGuarderia(usuarioID, gID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cuenta de personal no encontrada"})
		return
	}

	var input struct {
		HorasTrabajadas float64 `json:"horas_trabajadas"`
		Observaciones   string  `json:"observaciones"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}
	if input.HorasTrabajadas <= 0 || input.HorasTrabajadas > 24 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Las horas trabajadas deben ser mayores a 0 y hasta 24"})
		return
	}

	_, err := s.DBAuth.Exec(`
        INSERT INTO registro_horas (usuario_id, guarderia_id, fecha, horas_trabajadas, observaciones)
        VALUES ($1, $2, $3::date, $4, $5)
        ON CONFLICT (usuario_id, fecha) DO UPDATE SET
            horas_trabajadas = EXCLUDED.horas_trabajadas,
            observaciones = EXCLUDED.observaciones`,
		usuarioID, gID, fecha, input.HorasTrabajadas, input.Observaciones,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo guardar el registro"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Registro guardado"})
}

func (s *Server) handleEliminarHoras(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	usuarioID := c.Param("usuarioId")
	fecha := c.Param("fecha")

	if !s.personalPerteneceAGuarderia(usuarioID, gID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cuenta de personal no encontrada"})
		return
	}

	res, err := s.DBAuth.Exec(`DELETE FROM registro_horas WHERE usuario_id = $1 AND fecha = $2::date`, usuarioID, fecha)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo eliminar el registro"})
		return
	}
	if filas, _ := res.RowsAffected(); filas == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Registro no encontrado"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Registro eliminado"})
}

// soloHora recorta un valor de columna TIME a "HH:MM:SS". lib/pq entrega las
// columnas TIME escaneadas en *string con una fecha ficticia por delante
// ("0000-01-01T08:00:00Z"), mismo comportamiento que soloFecha ya documenta
// para DATE -- sin este recorte, <input type="time"> del frontend descarta
// el valor en silencio por no ser una hora válida.
func soloHora(s *string) *string {
	if s == nil {
		return nil
	}
	idx := strings.Index(*s, "T")
	if idx == -1 || len(*s) < idx+9 {
		return s
	}
	recortada := (*s)[idx+1 : idx+9]
	return &recortada
}
