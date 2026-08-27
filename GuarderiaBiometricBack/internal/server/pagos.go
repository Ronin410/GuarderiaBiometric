package server

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"biometrico/internal/middleware"
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
// TotalPagado es SOLO lo pagado por concepto "Colegiatura" — antes sumaba
// todos los conceptos juntos (Material, Inscripción, Otro incluidos), lo que
// hacía que pagar el material, por ejemplo, contara como abono a la
// mensualidad sin haberla tocado. Estado se calcula sobre este mismo total,
// así que el bug también inflaba el estado "pagado"/"parcial" cuando en
// realidad la colegiatura seguía sin pagarse.
type EstadoPagoNino struct {
	HijoID             int     `json:"hijo_id"`
	Nombre             string  `json:"nombre"`
	ColegiaturaMensual float64 `json:"colegiatura_mensual"`
	TotalPagado        float64 `json:"total_pagado"`
	Estado             string  `json:"estado"` // pagado | parcial | pendiente | vencido
	// Desglose de los demás conceptos en el mismo periodo — no tienen un
	// monto esperado configurado (a diferencia de colegiatura_mensual), así
	// que aquí solo se informa cuánto se pagó de cada uno, sin estado.
	TotalInscripcion float64 `json:"total_inscripcion"`
	TotalMaterial    float64 `json:"total_material"`
	TotalOtro        float64 `json:"total_otro"`
}

func (s *Server) registrarRutasPagos(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)
	staff := middleware.RequireArea("pagos")

	// --- REGISTRAR UN PAGO ---
	r.POST("/pagos", auth, staff, func(c *gin.Context) {
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
			input.FechaPago = hoyEnZonaLocal(time.Now())
		}

		var nuevoID int
		query := `
        INSERT INTO pagos (hijo_id, guarderia_id, monto, concepto, periodo, fecha_pago, metodo_pago, observaciones, registrado_por)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING id`

		err := s.DB.QueryRow(query,
			input.HijoID, gID, input.Monto, input.Concepto, input.Periodo,
			input.FechaPago, input.MetodoPago, input.Observaciones, userID,
		).Scan(&nuevoID)

		if err != nil {
			log.Printf("No se pudo registrar el pago: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo registrar el pago"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"id": nuevoID, "status": "Pago registrado correctamente"})
	})

	// --- HISTORIAL DE PAGOS (por niño y/o periodo) ---
	r.GET("/pagos", auth, staff, func(c *gin.Context) {
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

		rows, err := s.DB.Query(query, gID, hijoID, periodo)
		if err != nil {
			log.Printf("Error al consultar pagos: %v", err)
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
			if len(p.FechaPago) >= 10 {
				p.FechaPago = p.FechaPago[:10]
			}
			pagos = append(pagos, p)
		}

		c.JSON(http.StatusOK, pagos)
	})

	// --- ESTADO DE PAGO DE TODOS LOS NIÑOS ACTIVOS EN UN PERIODO ---
	r.GET("/pagos/estado", auth, staff, func(c *gin.Context) {
		gID, _ := c.Get("guarderia_id")
		periodo := c.Query("periodo")
		if len(periodo) != 7 {
			periodo = time.Now().In(zonaMazatlan()).Format("2006-01")
		}

		query := `
        SELECT h.id, h.nombre_niño, h.colegiatura_mensual,
               COALESCE(SUM(p.monto) FILTER (WHERE p.concepto = 'Colegiatura'), 0) as total_colegiatura,
               COALESCE(SUM(p.monto) FILTER (WHERE p.concepto = 'Inscripción'), 0) as total_inscripcion,
               COALESCE(SUM(p.monto) FILTER (WHERE p.concepto = 'Material'), 0) as total_material,
               COALESCE(SUM(p.monto) FILTER (WHERE p.concepto = 'Otro'), 0) as total_otro
        FROM hijos h
        LEFT JOIN pagos p ON p.hijo_id = h.id AND p.periodo = $2
        WHERE h.guarderia_id = $1 AND h.activo = true
        GROUP BY h.id, h.nombre_niño, h.colegiatura_mensual
        ORDER BY h.nombre_niño ASC`

		rows, err := s.DB.Query(query, gID, periodo)
		if err != nil {
			log.Printf("Error al consultar el estado de pagos: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar el estado de pagos"})
			return
		}
		defer rows.Close()

		periodoActual := time.Now().In(zonaMazatlan()).Format("2006-01")

		estados := []EstadoPagoNino{}
		for rows.Next() {
			var e EstadoPagoNino
			if err := rows.Scan(&e.HijoID, &e.Nombre, &e.ColegiaturaMensual, &e.TotalPagado, &e.TotalInscripcion, &e.TotalMaterial, &e.TotalOtro); err != nil {
				continue
			}
			e.Estado = calcularEstadoPago(e.ColegiaturaMensual, e.TotalPagado, periodo, periodoActual)
			estados = append(estados, e)
		}

		c.JSON(http.StatusOK, estados)
	})

	// --- ELIMINAR UN PAGO (corrección de captura) ---
	r.DELETE("/pagos/:id", auth, staff, func(c *gin.Context) {
		gID, _ := c.Get("guarderia_id")
		pagoID := c.Param("id")

		result, err := s.DB.Exec("DELETE FROM pagos WHERE id = $1 AND guarderia_id = $2", pagoID, gID)
		if err != nil {
			log.Printf("No se pudo eliminar el pago: %v", err)
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

	// --- MIS PAGOS (portal del papá) ---
	// Expone el estado de pago SOLO de los hijos del padre autenticado (a diferencia
	// de /pagos* que es para el staff y ve toda la guardería). padre_id se resuelve
	// igual que en /padre/:id/hijos: el user_id del token.
	r.GET("/padre/mis-pagos", auth, func(c *gin.Context) {
		gID, _ := c.Get("guarderia_id")
		tokenUsuarioID, _ := c.Get("user_id")

		loc := zonaMazatlan()
		periodo := c.Query("periodo")
		if len(periodo) != 7 {
			periodo = time.Now().In(loc).Format("2006-01")
		}

		query := `
        SELECT h.id, h.nombre_niño, h.colegiatura_mensual,
               COALESCE(SUM(p.monto) FILTER (WHERE p.concepto = 'Colegiatura'), 0) as total_colegiatura,
               COALESCE(SUM(p.monto) FILTER (WHERE p.concepto = 'Inscripción'), 0) as total_inscripcion,
               COALESCE(SUM(p.monto) FILTER (WHERE p.concepto = 'Material'), 0) as total_material,
               COALESCE(SUM(p.monto) FILTER (WHERE p.concepto = 'Otro'), 0) as total_otro
        FROM hijos h
        INNER JOIN tutor_hijos th ON th.hijo_id = h.id
        LEFT JOIN pagos p ON p.hijo_id = h.id AND p.periodo = $3
        WHERE th.padre_id = $1 AND h.guarderia_id = $2 AND h.activo = true
        GROUP BY h.id, h.nombre_niño, h.colegiatura_mensual
        ORDER BY h.nombre_niño ASC`

		rows, err := s.DB.Query(query, tokenUsuarioID, gID, periodo)
		if err != nil {
			log.Printf("Error al consultar tus pagos: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar tus pagos"})
			return
		}
		defer rows.Close()

		periodoActual := time.Now().In(loc).Format("2006-01")

		estados := []EstadoPagoNino{}
		for rows.Next() {
			var e EstadoPagoNino
			if err := rows.Scan(&e.HijoID, &e.Nombre, &e.ColegiaturaMensual, &e.TotalPagado, &e.TotalInscripcion, &e.TotalMaterial, &e.TotalOtro); err != nil {
				continue
			}
			e.Estado = calcularEstadoPago(e.ColegiaturaMensual, e.TotalPagado, periodo, periodoActual)
			estados = append(estados, e)
		}

		c.JSON(http.StatusOK, estados)
	})

	r.GET("/padre/mis-pagos/historial", auth, func(c *gin.Context) {
		gID, _ := c.Get("guarderia_id")
		tokenUsuarioID, _ := c.Get("user_id")
		hijoID := c.Query("hijo_id")

		if hijoID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "hijo_id requerido"})
			return
		}

		var esPropio bool
		err := s.DB.QueryRow(
			"SELECT EXISTS(SELECT 1 FROM tutor_hijos WHERE hijo_id = $1 AND padre_id = $2)",
			hijoID, tokenUsuarioID,
		).Scan(&esPropio)
		if err != nil || !esPropio {
			c.JSON(http.StatusForbidden, gin.H{"error": "No tienes permiso para consultar este historial"})
			return
		}

		query := `
        SELECT id, hijo_id, monto, concepto, periodo, fecha_pago, metodo_pago, COALESCE(observaciones, '')
        FROM pagos
        WHERE guarderia_id = $1 AND hijo_id::text = $2
        ORDER BY fecha_pago DESC, id DESC`

		rows, err := s.DB.Query(query, gID, hijoID)
		if err != nil {
			log.Printf("Error al consultar el historial: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar el historial"})
			return
		}
		defer rows.Close()

		pagos := []Pago{}
		for rows.Next() {
			var p Pago
			if err := rows.Scan(&p.ID, &p.HijoID, &p.Monto, &p.Concepto, &p.Periodo, &p.FechaPago, &p.MetodoPago, &p.Observaciones); err != nil {
				continue
			}
			if len(p.FechaPago) >= 10 {
				p.FechaPago = p.FechaPago[:10]
			}
			pagos = append(pagos, p)
		}

		c.JSON(http.StatusOK, pagos)
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
