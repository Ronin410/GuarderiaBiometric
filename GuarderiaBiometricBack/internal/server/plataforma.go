package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"biometrico/internal/applog"
	"biometrico/internal/middleware"
)

// plataforma.go -- panorama de las guarderías YA aprobadas (a diferencia de
// solicitudes.go, que solo maneja las solicitudes pendientes de
// aprobación). "Quiero ver qué guarderías están registradas, cuántos papás
// y cuántos niños hay registrados, y más información relevante" -- para
// que el dueño de la plataforma tenga un panorama de adopción sin tener
// que entrar a cada guardería una por una.
type GuarderiaPlataforma struct {
	ID           int     `json:"id"`
	Nombre       string  `json:"nombre"`
	Slug         string  `json:"slug"`
	Direccion    *string `json:"direccion"`
	CreadoEn     string  `json:"creado_en"`
	TotalNinos   int     `json:"total_ninos"`
	TotalPapas   int     `json:"total_papas"`
	TotalStaff   int     `json:"total_staff"` // incluye admin
	UltimoAcceso *string `json:"ultimo_acceso"`
	// "Quiero una forma de bloquear el acceso a una guardería... eso lo
	// haré manualmente" -- ver handleBloquearGuarderia/
	// handleDesbloquearGuarderia más abajo.
	Bloqueada   bool    `json:"bloqueada"`
	BloqueadaEn *string `json:"bloqueada_en"`
}

func (s *Server) registrarRutasPlataformaGuarderias(r *gin.Engine) {
	platform := middleware.RequirePlatformKey(s.PlatformAdminKey)
	r.GET("/plataforma/guarderias", platform, s.handleListarGuarderiasPlataforma)
	r.POST("/plataforma/guarderias/:id/bloquear", platform, s.handleBloquearGuarderia)
	r.POST("/plataforma/guarderias/:id/desbloquear", platform, s.handleDesbloquearGuarderia)
}

// handleBloquearGuarderia corta el acceso de TODA la guardería (admin,
// staff y papás) -- "si no pagan manualmente quiero quitarles el acceso
// dándoles tiempo para que paguen, pero eso lo haré manualmente": nada
// automático la vuelve a bloquear ni la desbloquea sola, es un interruptor
// que el dueño de la plataforma prende/apaga a mano. Ver el efecto real en
// handleLogin/handleMe (auth.go).
func (s *Server) handleBloquearGuarderia(c *gin.Context) {
	id := c.Param("id")
	res, err := s.DBAuth.Exec(`UPDATE guarderias SET bloqueada = true, bloqueada_en = now() WHERE id = $1`, id)
	if err != nil {
		s.logError(c, "Error al bloquear la guardería", err, "guarderia_id", id)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo bloquear la guardería"})
		return
	}
	if filas, _ := res.RowsAffected(); filas == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Guardería no encontrada"})
		return
	}
	applog.Info("Guardería bloqueada desde /plataforma", "guarderia_id", id)
	c.JSON(http.StatusOK, gin.H{"message": "Guardería bloqueada"})
}

func (s *Server) handleDesbloquearGuarderia(c *gin.Context) {
	id := c.Param("id")
	res, err := s.DBAuth.Exec(`UPDATE guarderias SET bloqueada = false, bloqueada_en = NULL WHERE id = $1`, id)
	if err != nil {
		s.logError(c, "Error al desbloquear la guardería", err, "guarderia_id", id)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo desbloquear la guardería"})
		return
	}
	if filas, _ := res.RowsAffected(); filas == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Guardería no encontrada"})
		return
	}
	applog.Info("Guardería desbloqueada desde /plataforma", "guarderia_id", id)
	c.JSON(http.StatusOK, gin.H{"message": "Guardería desbloqueada"})
}

// handleListarGuarderiasPlataforma junta datos de DBAuth (guarderias,
// usuarios) y de DB (hijos, logs_acceso) en tres consultas por separado --
// database/sql no puede unir en una sola query dos *sql.DB distintos,
// aunque en este despliegue ambos apunten a la misma base física (mismo
// motivo que el comentario sobre usuario_id en la migración 000015 de
// logs_acceso).
func (s *Server) handleListarGuarderiasPlataforma(c *gin.Context) {
	rows, err := s.DBAuth.Query(`
        SELECT g.id, g.nombre, g.slug, g.direccion, g.created_at, g.bloqueada, g.bloqueada_en,
               COUNT(*) FILTER (WHERE u.rol = 'papa') AS total_papas,
               COUNT(*) FILTER (WHERE u.rol IN ('admin', 'staff') AND u.activo) AS total_staff
        FROM guarderias g
        LEFT JOIN usuarios u ON u.guarderia_id = g.id
        GROUP BY g.id, g.nombre, g.slug, g.direccion, g.created_at, g.bloqueada, g.bloqueada_en
        ORDER BY g.created_at DESC`)
	if err != nil {
		s.logError(c, "Error al consultar guarderías", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar guarderías"})
		return
	}
	guarderias := []GuarderiaPlataforma{}
	for rows.Next() {
		var g GuarderiaPlataforma
		if err := rows.Scan(&g.ID, &g.Nombre, &g.Slug, &g.Direccion, &g.CreadoEn, &g.Bloqueada, &g.BloqueadaEn, &g.TotalPapas, &g.TotalStaff); err != nil {
			continue
		}
		guarderias = append(guarderias, g)
	}
	rows.Close()

	ninosPorGuarderia := map[int]int{}
	rowsNinos, err := s.DB.Query(`SELECT guarderia_id, COUNT(*) FROM hijos WHERE activo = true GROUP BY guarderia_id`)
	if err != nil {
		s.logError(c, "Error al contar niños por guardería", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al contar niños por guardería"})
		return
	}
	for rowsNinos.Next() {
		var gID, total int
		if err := rowsNinos.Scan(&gID, &total); err != nil {
			continue
		}
		ninosPorGuarderia[gID] = total
	}
	rowsNinos.Close()

	// Última vez que alguien de esa guardería (admin, staff o papá) entró
	// de verdad -- un indicador rápido de si la guardería sigue usando el
	// sistema activamente o quedó abandonada tras el alta.
	ultimoAccesoPorGuarderia := map[int]string{}
	rowsAcceso, err := s.DB.Query(`
        SELECT guarderia_id, MAX(creado_en) FROM logs_acceso
        WHERE evento = 'login_exitoso' AND guarderia_id IS NOT NULL
        GROUP BY guarderia_id`)
	if err != nil {
		s.logError(c, "Error al consultar el último acceso por guardería", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar el último acceso por guardería"})
		return
	}
	for rowsAcceso.Next() {
		var gID int
		var ultimo string
		if err := rowsAcceso.Scan(&gID, &ultimo); err != nil {
			continue
		}
		ultimoAccesoPorGuarderia[gID] = ultimo
	}
	rowsAcceso.Close()

	for i := range guarderias {
		guarderias[i].TotalNinos = ninosPorGuarderia[guarderias[i].ID]
		if ultimo, ok := ultimoAccesoPorGuarderia[guarderias[i].ID]; ok {
			guarderias[i].UltimoAcceso = &ultimo
		}
	}

	c.JSON(http.StatusOK, guarderias)
}
