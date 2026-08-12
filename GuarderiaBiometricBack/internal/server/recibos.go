package server

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"biometrico/internal/middleware"
)

// ReciboPago es la vista imprimible de un pago ya registrado -- trae los
// datos que ya viven en otras tablas (nombre del niño, nombre/dirección de
// la guardería) para que el frontend no tenga que pedirlos aparte.
type ReciboPago struct {
	ID                 int     `json:"id"`
	Folio              string  `json:"folio"`
	NinoNombre         string  `json:"nino_nombre"`
	GuarderiaNombre    string  `json:"guarderia_nombre"`
	GuarderiaDireccion *string `json:"guarderia_direccion"`
	Monto              float64 `json:"monto"`
	Concepto           string  `json:"concepto"`
	Periodo            string  `json:"periodo"`
	FechaPago          string  `json:"fecha_pago"`
	MetodoPago         string  `json:"metodo_pago"`
	Observaciones      string  `json:"observaciones"`
}

func (s *Server) registrarRutasRecibos(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)
	staff := middleware.RequireStaff()

	r.GET("/pagos/:id/recibo", auth, staff, s.handleObtenerRecibo)
	// Mismo handler para el portal del padre -- la diferencia es que aquí
	// además se verifica que el pago sea de un hijo suyo (ver abajo), en vez
	// de solo confiar en el guarderia_id del token como alcanza para staff.
	r.GET("/padre/pagos/:id/recibo", auth, s.handleObtenerRecibo)
	r.POST("/pagos/recordatorio", auth, staff, s.handleEnviarRecordatorios)
}

func (s *Server) handleObtenerRecibo(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	rol, _ := c.Get("rol")
	userID, _ := c.Get("user_id")
	pagoID := c.Param("id")

	var rec ReciboPago
	var direccion sql.NullString
	var fechaPago *string
	var hijoID int

	err := s.DB.QueryRow(`
        SELECT p.id, p.monto, p.concepto, p.periodo, p.fecha_pago, p.metodo_pago, COALESCE(p.observaciones, ''),
               h.id, h.nombre_niño, g.nombre, g.direccion
        FROM pagos p
        JOIN hijos h ON h.id = p.hijo_id
        JOIN guarderias g ON g.id = p.guarderia_id
        WHERE p.id = $1 AND p.guarderia_id = $2`,
		pagoID, gID,
	).Scan(&rec.ID, &rec.Monto, &rec.Concepto, &rec.Periodo, &fechaPago, &rec.MetodoPago, &rec.Observaciones,
		&hijoID, &rec.NinoNombre, &rec.GuarderiaNombre, &direccion)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Recibo no encontrado"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar el recibo"})
		return
	}

	// El filtro por guarderia_id de arriba evita que un papá vea recibos de
	// OTRA guardería, pero no evita que vea el de OTRA familia dentro de la
	// misma guardería con solo adivinar el id del pago en la URL -- por eso
	// esta verificación aparte, solo para rol "papa".
	if rol == "papa" {
		var esPropio bool
		if err := s.DB.QueryRow(
			"SELECT EXISTS(SELECT 1 FROM tutor_hijos WHERE hijo_id = $1 AND padre_id = $2)",
			hijoID, userID,
		).Scan(&esPropio); err != nil || !esPropio {
			c.JSON(http.StatusForbidden, gin.H{"error": "No tienes permiso para ver este recibo"})
			return
		}
	}

	if f := soloFecha(fechaPago); f != nil {
		rec.FechaPago = *f
	}
	rec.Folio = fmt.Sprintf("REC-%06d", rec.ID)
	if direccion.Valid {
		rec.GuarderiaDireccion = &direccion.String
	}

	c.JSON(http.StatusOK, rec)
}

// handleEnviarRecordatorios manda una notificación push a los tutores de
// cada niño con estado "pendiente" o "vencido" en el periodo indicado. Es
// una acción manual (el staff decide cuándo, no un cron automático) para
// no mandar recordatorios de dinero a horas raras o antes de que alguien
// capture un pago en efectivo que ya se hizo pero no se ha registrado.
func (s *Server) handleEnviarRecordatorios(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	periodo := c.Query("periodo")
	if len(periodo) != 7 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "periodo (YYYY-MM) es obligatorio"})
		return
	}
	if !s.PushConfigurado() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Notificaciones push no configuradas en el servidor"})
		return
	}

	rows, err := s.DB.Query(`
        SELECT h.id, h.colegiatura_mensual,
               COALESCE(SUM(p.monto) FILTER (WHERE p.concepto = 'Colegiatura'), 0)
        FROM hijos h
        LEFT JOIN pagos p ON p.hijo_id = h.id AND p.periodo = $2
        WHERE h.guarderia_id = $1 AND h.activo = true
        GROUP BY h.id, h.colegiatura_mensual`,
		gID, periodo,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar el estado de pagos"})
		return
	}

	periodoActual := time.Now().In(zonaMazatlan()).Format("2006-01")
	var hijosIDsPendientes []int
	for rows.Next() {
		var hijoID int
		var colegiatura, pagado float64
		if err := rows.Scan(&hijoID, &colegiatura, &pagado); err != nil {
			continue
		}
		estado := calcularEstadoPago(colegiatura, pagado, periodo, periodoActual)
		if estado == "pendiente" || estado == "vencido" {
			hijosIDsPendientes = append(hijosIDsPendientes, hijoID)
		}
	}
	rows.Close()

	for _, hijoID := range hijosIDsPendientes {
		go s.notificarEvento(hijoID, "RECORDATORIO_PAGO", periodo)
	}

	c.JSON(http.StatusOK, gin.H{"enviados": len(hijosIDsPendientes)})
}
