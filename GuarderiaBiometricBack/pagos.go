package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Pago struct {
	ID            int     `json:"id"`
	HijoID        int     `json:"hijo_id"`
	Monto         float64 `json:"monto"`
	Concepto      string  `json:"concepto"`
	Periodo       string  `json:"periodo"`
	FechaPago     string  `json:"fecha_pago"`
	MetodoPago    string  `json:"metodo_pago"`
	Observaciones string  `json:"observaciones"`
}

// EstadoPagoNino resume, para un periodo dado, cuánto debe y cuánto ha pagado un niño.
type EstadoPagoNino struct {
	HijoID             int     `json:"hijo_id"`
	Nombre             string  `json:"nombre"`
	ColegiaturaMensual float64 `json:"colegiatura_mensual"`
	TotalPagado        float64 `json:"total_pagado"`
	Estado             string  `json:"estado"` // pagado | parcial | pendiente | vencido
}

func registrarRutasPagos(r *gin.Engine) {
	// --- REGISTRAR UN PAGO ---
	r.POST("/pagos", AuthMiddleware(), func(c *gin.Context) {
		gID, _ := c.Get("guarderia_id")
		userID, _ := c.Get("user_id")

		var input struct {
			HijoID        int     `json:"hijo_id"`
			Monto         float64 `json:"monto"`
			Concepto      string  `json:"concepto"`
			Periodo       string  `json:"periodo"`
			FechaPago     string  `json:"fecha_pago"`
			MetodoPago    string  `json:"metodo_pago"`
			Observaciones string  `json:"observaciones"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
			return
		}

		if input.HijoID == 0 || input.Monto <= 0 || len(input.Periodo) != 7 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "hijo_id, monto y periodo (YYYY-MM) son obligatorios"})
			return
		}
		if input.Concepto == "" {
			input.Concepto = "Colegiatura"
		}
		if input.MetodoPago == "" {
			input.MetodoPago = "efectivo"
		}
		if input.FechaPago == "" {
			loc, err := time.LoadLocation("America/Mazatlan")
			if err != nil {
				loc = time.UTC
			}
			input.FechaPago = time.Now().In(loc).Format("2006-01-02")
		}

		var nuevoID int
		query := `
        INSERT INTO pagos (hijo_id, guarderia_id, monto, concepto, periodo, fecha_pago, metodo_pago, observaciones, registrado_por)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING id`

		err := db.QueryRow(query,
			input.HijoID, gID, input.Monto, input.Concepto, input.Periodo,
			input.FechaPago, input.MetodoPago, input.Observaciones, userID,
		).Scan(&nuevoID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo registrar el pago"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"id": nuevoID, "status": "Pago registrado correctamente"})
	})

	// --- HISTORIAL DE PAGOS (por niño y/o periodo) ---
	r.GET("/pagos", AuthMiddleware(), func(c *gin.Context) {
		gID, _ := c.Get("guarderia_id")
		hijoID := c.Query("hijo_id")
		periodo := c.Query("periodo")

		query := `
        SELECT id, hijo_id, monto, concepto, periodo, fecha_pago, metodo_pago, COALESCE(observaciones, '')
        FROM pagos
        WHERE guarderia_id = $1
          AND ($2 = '' OR hijo_id::text = $2)
          AND ($3 = '' OR periodo = $3)
        ORDER BY fecha_pago DESC, id DESC`

		rows, err := db.Query(query, gID, hijoID, periodo)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar pagos"})
			return
		}
		defer rows.Close()

		pagos := []Pago{}
		for rows.Next() {
			var p Pago
			if err := rows.Scan(&p.ID, &p.HijoID, &p.Monto, &p.Concepto, &p.Periodo, &p.FechaPago, &p.MetodoPago, &p.Observaciones); err != nil {
				continue
			}
			pagos = append(pagos, p)
		}

		c.JSON(http.StatusOK, pagos)
	})

	// --- ESTADO DE PAGO DE TODOS LOS NIÑOS ACTIVOS EN UN PERIODO ---
	r.GET("/pagos/estado", AuthMiddleware(), func(c *gin.Context) {
		gID, _ := c.Get("guarderia_id")
		periodo := c.Query("periodo")
		if len(periodo) != 7 {
			loc, err := time.LoadLocation("America/Mazatlan")
			if err != nil {
				loc = time.UTC
			}
			periodo = time.Now().In(loc).Format("2006-01")
		}

		query := `
        SELECT h.id, h.nombre_niño, h.colegiatura_mensual,
               COALESCE(SUM(p.monto), 0) as total_pagado
        FROM hijos h
        LEFT JOIN pagos p ON p.hijo_id = h.id AND p.periodo = $2
        WHERE h.guarderia_id = $1 AND h.activo = true
        GROUP BY h.id, h.nombre_niño, h.colegiatura_mensual
        ORDER BY h.nombre_niño ASC`

		rows, err := db.Query(query, gID, periodo)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar el estado de pagos"})
			return
		}
		defer rows.Close()

		loc, err := time.LoadLocation("America/Mazatlan")
		if err != nil {
			loc = time.UTC
		}
		periodoActual := time.Now().In(loc).Format("2006-01")

		estados := []EstadoPagoNino{}
		for rows.Next() {
			var e EstadoPagoNino
			if err := rows.Scan(&e.HijoID, &e.Nombre, &e.ColegiaturaMensual, &e.TotalPagado); err != nil {
				continue
			}
			e.Estado = calcularEstadoPago(e.ColegiaturaMensual, e.TotalPagado, periodo, periodoActual)
			estados = append(estados, e)
		}

		c.JSON(http.StatusOK, estados)
	})

	// --- ELIMINAR UN PAGO (corrección de captura) ---
	r.DELETE("/pagos/:id", AuthMiddleware(), func(c *gin.Context) {
		gID, _ := c.Get("guarderia_id")
		pagoID := c.Param("id")

		result, err := db.Exec("DELETE FROM pagos WHERE id = $1 AND guarderia_id = $2", pagoID, gID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo eliminar el pago"})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pago no encontrado"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "Pago eliminado"})
	})
}

// calcularEstadoPago decide si un niño está "pagado", "parcial", "vencido" o "pendiente"
// para un periodo dado, comparando lo pagado contra la colegiatura mensual configurada.
func calcularEstadoPago(colegiatura, totalPagado float64, periodo, periodoActual string) string {
	if colegiatura > 0 && totalPagado >= colegiatura {
		return "pagado"
	}
	if totalPagado > 0 {
		return "parcial"
	}
	if periodo < periodoActual {
		return "vencido"
	}
	return "pendiente"
}
