package server

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

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
}

func (s *Server) registrarRutasPlataformaGuarderias(r *gin.Engine) {
	platform := middleware.RequirePlatformKey(s.PlatformAdminKey)
	r.GET("/plataforma/guarderias", platform, s.handleListarGuarderiasPlataforma)
}

// handleListarGuarderiasPlataforma junta datos de DBAuth (guarderias,
// usuarios) y de DB (hijos, logs_acceso) en tres consultas por separado --
// database/sql no puede unir en una sola query dos *sql.DB distintos,
// aunque en este despliegue ambos apunten a la misma base física (mismo
// motivo que el comentario sobre usuario_id en la migración 000015 de
// logs_acceso).
func (s *Server) handleListarGuarderiasPlataforma(c *gin.Context) {
	rows, err := s.DBAuth.Query(`
        SELECT g.id, g.nombre, g.slug, g.direccion, g.created_at,
               COUNT(*) FILTER (WHERE u.rol = 'papa') AS total_papas,
               COUNT(*) FILTER (WHERE u.rol IN ('admin', 'staff') AND u.activo) AS total_staff
        FROM guarderias g
        LEFT JOIN usuarios u ON u.guarderia_id = g.id
        GROUP BY g.id, g.nombre, g.slug, g.direccion, g.created_at
        ORDER BY g.created_at DESC`)
	if err != nil {
		log.Printf("Error al consultar guarderías: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar guarderías"})
		return
	}
	guarderias := []GuarderiaPlataforma{}
	for rows.Next() {
		var g GuarderiaPlataforma
		if err := rows.Scan(&g.ID, &g.Nombre, &g.Slug, &g.Direccion, &g.CreadoEn, &g.TotalPapas, &g.TotalStaff); err != nil {
			continue
		}
		guarderias = append(guarderias, g)
	}
	rows.Close()

	ninosPorGuarderia := map[int]int{}
	rowsNinos, err := s.DB.Query(`SELECT guarderia_id, COUNT(*) FROM hijos WHERE activo = true GROUP BY guarderia_id`)
	if err != nil {
		log.Printf("Error al contar niños por guardería: %v", err)
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
		log.Printf("Error al consultar el último acceso por guardería: %v", err)
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
