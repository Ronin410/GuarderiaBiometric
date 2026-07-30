package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
}

func registrarRutasPerfiles(r *gin.Engine) {
	// --- LISTA COMPLETA DE NIÑOS CON DATOS EXTENDIDOS (Panel de Administración) ---
	r.GET("/admin/ninos", AuthMiddleware(), func(c *gin.Context) {
		gID, _ := c.Get("guarderia_id")

		query := `
        SELECT
            h.id, h.nombre_niño, h.activo, h.fecha_nacimiento, h.direccion,
            h.contacto_emergencia_nombre, h.contacto_emergencia_telefono, h.colegiatura_mensual,
            COALESCE(string_agg(DISTINCT p.nombre, ', '), '') as tutores
        FROM hijos h
        LEFT JOIN tutor_hijos th ON th.hijo_id = h.id
        LEFT JOIN padres p ON p.id = th.padre_id
        WHERE h.guarderia_id = $1
        GROUP BY h.id
        ORDER BY h.nombre_niño ASC`

		rows, err := db.Query(query, gID)
		if err != nil {
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
				&n.Tutores,
			); err != nil {
				continue
			}
			ninos = append(ninos, n)
		}

		c.JSON(http.StatusOK, ninos)
	})

	// --- ACTUALIZAR PERFIL EXTENDIDO DE UN NIÑO ---
	r.PUT("/hijos/:id/perfil", AuthMiddleware(), func(c *gin.Context) {
		gID, _ := c.Get("guarderia_id")
		hijoID := c.Param("id")

		var input struct {
			FechaNacimiento            *string `json:"fecha_nacimiento"`
			Direccion                  *string `json:"direccion"`
			ContactoEmergenciaNombre   *string `json:"contacto_emergencia_nombre"`
			ContactoEmergenciaTelefono *string `json:"contacto_emergencia_telefono"`
			ColegiaturaMensual         float64 `json:"colegiatura_mensual"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
			return
		}

		// nullif('', ...) permite guardar NULL cuando el frontend manda cadena vacía
		query := `
        UPDATE hijos SET
            fecha_nacimiento = NULLIF($1, '')::date,
            direccion = $2,
            contacto_emergencia_nombre = $3,
            contacto_emergencia_telefono = $4,
            colegiatura_mensual = $5
        WHERE id = $6 AND guarderia_id = $7`

		result, err := db.Exec(query,
			derefOrEmpty(input.FechaNacimiento),
			input.Direccion,
			input.ContactoEmergenciaNombre,
			input.ContactoEmergenciaTelefono,
			input.ColegiaturaMensual,
			hijoID, gID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo actualizar el perfil"})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Niño no encontrado o no pertenece a esta guardería"})
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
