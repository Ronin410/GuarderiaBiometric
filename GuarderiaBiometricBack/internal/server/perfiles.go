package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"biometrico/internal/middleware"
)

// NinoPerfil representa la ficha administrativa completa de un niño,
// incluyendo los tutores vinculados (para la tabla del panel de Administración).
type NinoPerfil struct {
	ID                         int     `json:"id"`
	Nombre                     string  `json:"nombre"`
	Activo                     bool    `json:"activo"`
	FechaNacimiento            *string `json:"fecha_nacimiento"`
	Direccion                  *string `json:"direccion"`
	ContactoEmergenciaNombre   *string `json:"contacto_emergencia_nombre"`
	ContactoEmergenciaTelefono *string `json:"contacto_emergencia_telefono"`
	ColegiaturaMensual         float64 `json:"colegiatura_mensual"`
	Tutores                    string  `json:"tutores"`
	GrupoID                    *int    `json:"grupo_id"`
	GrupoNombre                *string `json:"grupo_nombre"`
}

func (s *Server) registrarRutasPerfiles(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)
	staff := middleware.RequireArea("perfiles")

	// --- LISTA COMPLETA DE NIÑOS CON DATOS EXTENDIDOS (Panel de Administración) ---
	r.GET("/admin/ninos", auth, staff, func(c *gin.Context) {
		gID, _ := c.Get("guarderia_id")

		query := `
        SELECT
            h.id, h.nombre_niño, h.activo, h.fecha_nacimiento, h.direccion,
            h.contacto_emergencia_nombre, h.contacto_emergencia_telefono, h.colegiatura_mensual,
            COALESCE(string_agg(DISTINCT p.nombre, ', '), '') as tutores,
            h.grupo_id, g.nombre
        FROM hijos h
        LEFT JOIN tutor_hijos th ON th.hijo_id = h.id
        LEFT JOIN padres p ON p.id = th.padre_id
        LEFT JOIN grupos g ON g.id = h.grupo_id
        WHERE h.guarderia_id = $1
        GROUP BY h.id, g.nombre
        ORDER BY h.nombre_niño ASC`

		rows, err := s.DB.Query(query, gID)
		if err != nil {
			s.logError(c, "Error al consultar niños", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar niños"})
			return
		}
		defer rows.Close()

		ninos := []NinoPerfil{}
		for rows.Next() {
			var n NinoPerfil
			if err := rows.Scan(
				&n.ID, &n.Nombre, &n.Activo, &n.FechaNacimiento, &n.Direccion,
				&n.ContactoEmergenciaNombre, &n.ContactoEmergenciaTelefono, &n.ColegiaturaMensual,
				&n.Tutores, &n.GrupoID, &n.GrupoNombre,
			); err != nil {
				continue
			}
			n.FechaNacimiento = soloFecha(n.FechaNacimiento)
			ninos = append(ninos, n)
		}

		c.JSON(http.StatusOK, ninos)
	})

	// --- ACTUALIZAR PERFIL EXTENDIDO DE UN NIÑO ---
	r.PUT("/hijos/:id/perfil", auth, staff, func(c *gin.Context) {
		gID, _ := c.Get("guarderia_id")
		hijoID := c.Param("id")

		var input struct {
			FechaNacimiento            *string `json:"fecha_nacimiento"`
			Direccion                  *string `json:"direccion"`
			ContactoEmergenciaNombre   *string `json:"contacto_emergencia_nombre"`
			ContactoEmergenciaTelefono *string `json:"contacto_emergencia_telefono"`
			ColegiaturaMensual         float64 `json:"colegiatura_mensual"`
			GrupoID                    *int    `json:"grupo_id"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
			return
		}

		// nullif('', ...) permite guardar NULL cuando el frontend manda cadena vacía.
		// El EXISTS sobre grupo_id evita asignar un grupo de otra guardería aunque
		// alguien arme la petición a mano (el frontend solo ofrece los propios).
		query := `
        UPDATE hijos SET
            fecha_nacimiento = NULLIF($1, '')::date,
            direccion = $2,
            contacto_emergencia_nombre = $3,
            contacto_emergencia_telefono = $4,
            colegiatura_mensual = $5,
            grupo_id = $6
        WHERE id = $7 AND guarderia_id = $8
          AND ($6::int IS NULL OR EXISTS (SELECT 1 FROM grupos gr WHERE gr.id = $6 AND gr.guarderia_id = $8))`

		result, err := s.DB.Exec(query,
			derefOrEmpty(input.FechaNacimiento),
			input.Direccion,
			input.ContactoEmergenciaNombre,
			input.ContactoEmergenciaTelefono,
			input.ColegiaturaMensual,
			input.GrupoID,
			hijoID, gID,
		)
		if err != nil {
			s.logError(c, "No se pudo actualizar el perfil", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo actualizar el perfil"})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Niño no encontrado, no pertenece a esta guardería, o el grupo no es válido"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "Perfil actualizado correctamente"})
	})
}

func derefOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// soloFecha recorta un valor de columna DATE a "YYYY-MM-DD". lib/pq entrega las
// columnas DATE escaneadas en *string como RFC3339 (ej. "2020-05-14T00:00:00Z"),
// que rompe el <input type="date"> del frontend si se envía tal cual.
func soloFecha(s *string) *string {
	if s == nil || len(*s) < 10 {
		return s
	}
	recortada := (*s)[:10]
	return &recortada
}
