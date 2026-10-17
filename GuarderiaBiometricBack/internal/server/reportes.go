package server

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"biometrico/internal/middleware"
)

// horaLimiteEntrada define a partir de qué hora una ENTRADA se considera "llegada tarde".
// Simplificación deliberada para la primera versión: es la misma para todas las guarderías.
// Si en el futuro se necesita por sede, se puede mover a una columna en "guarderias".
const horaLimiteEntrada = "09:00:00"

type ResumenAsistenciaNino struct {
	HijoID               int     `json:"hijo_id"`
	Nombre               string  `json:"nombre"`
	DiasHabiles          int     `json:"dias_habiles"`
	DiasAsistio          int     `json:"dias_asistio"`
	DiasAusente          int     `json:"dias_ausente"`
	DiasTarde            int     `json:"dias_tarde"`
	PorcentajeAsistencia float64 `json:"porcentaje_asistencia"`
}

func (s *Server) registrarRutasReportes(r *gin.Engine) {
	// Nota: esta ruta alimenta la pestaña "Estadísticas" del panel (área de
	// permisos "estadisticas"), no la pestaña "Reportes" (esa es
	// /reportes-asistencia, en bitacora.go) -- nombres parecidos, pestañas
	// distintas.
	r.GET("/reportes/asistencia-resumen", middleware.Auth(s.JWTKey), middleware.RequireArea("estadisticas"), func(c *gin.Context) {
		gID, _ := c.Get("guarderia_id")

		loc := zonaMazatlan()
		hoy := time.Now().In(loc)

		desdeStr := c.Query("desde")
		hastaStr := c.Query("hasta")

		desde, errD := time.ParseInLocation("2006-01-02", desdeStr, loc)
		hasta, errH := time.ParseInLocation("2006-01-02", hastaStr, loc)
		if errD != nil || errH != nil {
			// Sin rango válido: default al mes actual completo hasta hoy.
			desde = time.Date(hoy.Year(), hoy.Month(), 1, 0, 0, 0, 0, loc)
			hasta = hoy
		}
		// No contamos días futuros como "ausencia" todavía.
		if hasta.After(hoy) {
			hasta = hoy
		}
		if hasta.Before(desde) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "El rango de fechas es inválido"})
			return
		}

		hijoIDFiltro := 0
		if v := c.Query("hijo_id"); v != "" {
			// Ignoramos silenciosamente si no es numérico; el filtro simplemente no aplica.
			if parsed, errParse := strconv.Atoi(v); errParse == nil {
				hijoIDFiltro = parsed
			}
		}

		query := `
        SELECT h.id, h.nombre_niño,
            COUNT(DISTINCT CASE WHEN a.tipo_movimiento = 'ENTRADA'
                THEN (a.fecha_hora AT TIME ZONE 'America/Mazatlan')::date END) as dias_asistio,
            COUNT(DISTINCT CASE WHEN a.tipo_movimiento = 'ENTRADA'
                AND (a.fecha_hora AT TIME ZONE 'America/Mazatlan')::time > $4::time
                THEN (a.fecha_hora AT TIME ZONE 'America/Mazatlan')::date END) as dias_tarde
        FROM hijos h
        LEFT JOIN asistencia a ON a.hijo_id = h.id
            AND (a.fecha_hora AT TIME ZONE 'America/Mazatlan')::date BETWEEN $2::date AND $3::date
        WHERE h.guarderia_id = $1 AND h.activo = true
            AND ($5 = 0 OR h.id = $5)
        GROUP BY h.id, h.nombre_niño
        ORDER BY h.nombre_niño ASC`

		rows, err := s.DB.Query(query,
			gID, desde.Format("2006-01-02"), hasta.Format("2006-01-02"), horaLimiteEntrada, hijoIDFiltro,
		)
		if err != nil {
			s.logError(c, "Error al calcular el resumen de asistencia", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al calcular el resumen de asistencia"})
			return
		}
		defer rows.Close()

		diasHabiles := contarDiasHabiles(desde, hasta)

		resumen := []ResumenAsistenciaNino{}
		for rows.Next() {
			var res ResumenAsistenciaNino
			if err := rows.Scan(&res.HijoID, &res.Nombre, &res.DiasAsistio, &res.DiasTarde); err != nil {
				continue
			}
			res.DiasHabiles = diasHabiles
			res.DiasAusente = diasHabiles - res.DiasAsistio
			if res.DiasAusente < 0 {
				res.DiasAusente = 0
			}
			if diasHabiles > 0 {
				res.PorcentajeAsistencia = (float64(res.DiasAsistio) / float64(diasHabiles)) * 100
			}
			resumen = append(resumen, res)
		}

		// Peor asistencia primero, para que salte a la vista quién necesita seguimiento.
		sort.Slice(resumen, func(i, j int) bool {
			return resumen[i].PorcentajeAsistencia < resumen[j].PorcentajeAsistencia
		})

		c.JSON(http.StatusOK, resumen)
	})
}

// contarDiasHabiles cuenta los días de lunes a viernes en el rango [desde, hasta], ambos inclusive.
func contarDiasHabiles(desde, hasta time.Time) int {
	dias := 0
	for d := desde; !d.After(hasta); d = d.AddDate(0, 0, 1) {
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
			dias++
		}
	}
	return dias
}
