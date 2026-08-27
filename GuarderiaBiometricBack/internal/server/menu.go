package server

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"biometrico/internal/middleware"
)

// DiaMenu es un día del menú semanal. Los tres campos van en puntero: un día
// sin capturar todavía (o con un campo vacío a propósito) se distingue de
// uno con cadena vacía real.
type DiaMenu struct {
	Fecha    string  `json:"fecha"`
	Desayuno *string `json:"desayuno"`
	Comida   *string `json:"comida"`
	Merienda *string `json:"merienda"`
}

func (s *Server) registrarRutasMenu(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)
	staff := middleware.RequireArea("menu")

	r.GET("/menu-semanal", auth, staff, s.handleObtenerMenuSemanal)
	r.PUT("/menu-semanal/:fecha", auth, staff, s.handleGuardarDiaMenu)
	// Mismo handler para el portal del padre: el menú no es información
	// sensible (a diferencia de domicilios/pagos, que sí exigen RequireStaff),
	// así que basta con Auth() -- guarderia_id ya viene del token igual para
	// admin, staff o papá.
	r.GET("/padre/menu-semanal", auth, s.handleObtenerMenuSemanal)
}

func (s *Server) handleObtenerMenuSemanal(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	inicio := c.Query("inicio")
	fin := c.Query("fin")

	if _, err := time.Parse("2006-01-02", inicio); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parámetro 'inicio' inválido (usa YYYY-MM-DD)"})
		return
	}
	if _, err := time.Parse("2006-01-02", fin); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parámetro 'fin' inválido (usa YYYY-MM-DD)"})
		return
	}

	rows, err := s.DB.Query(
		`SELECT fecha, desayuno, comida, merienda FROM menu_semanal
         WHERE guarderia_id = $1 AND fecha BETWEEN $2 AND $3
         ORDER BY fecha ASC`,
		gID, inicio, fin,
	)
	if err != nil {
		log.Printf("Error al consultar el menú: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar el menú"})
		return
	}
	defer rows.Close()

	dias := []DiaMenu{}
	for rows.Next() {
		var d DiaMenu
		var fechaRaw *string
		if err := rows.Scan(&fechaRaw, &d.Desayuno, &d.Comida, &d.Merienda); err != nil {
			continue
		}
		if f := soloFecha(fechaRaw); f != nil {
			d.Fecha = *f
		}
		dias = append(dias, d)
	}
	c.JSON(http.StatusOK, dias)
}

func (s *Server) handleGuardarDiaMenu(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	fecha := c.Param("fecha")

	if _, err := time.Parse("2006-01-02", fecha); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Fecha inválida (usa YYYY-MM-DD)"})
		return
	}

	var input struct {
		Desayuno *string `json:"desayuno"`
		Comida   *string `json:"comida"`
		Merienda *string `json:"merienda"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	_, err := s.DB.Exec(`
        INSERT INTO menu_semanal (guarderia_id, fecha, desayuno, comida, merienda)
        VALUES ($1, $2::date, $3, $4, $5)
        ON CONFLICT (guarderia_id, fecha) DO UPDATE SET
            desayuno = EXCLUDED.desayuno,
            comida = EXCLUDED.comida,
            merienda = EXCLUDED.merienda,
            actualizado_en = CURRENT_TIMESTAMP`,
		gID, fecha, input.Desayuno, input.Comida, input.Merienda,
	)
	if err != nil {
		log.Printf("No se pudo guardar el menú: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo guardar el menú"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Menú guardado"})
}
