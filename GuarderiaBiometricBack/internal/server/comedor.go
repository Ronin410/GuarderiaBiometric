package server

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"biometrico/internal/middleware"
)

// PedidoComedor es una EXCEPCIÓN al comedor por defecto de un día -- ver
// comentario de la migración 000027. Por defecto todo niño activo come las
// tres comidas; esta fila solo existe cuando algo cambia.
type PedidoComedor struct {
	ID       int    `json:"id"`
	HijoID   int    `json:"hijo_id"`
	Fecha    string `json:"fecha"`
	Desayuno bool   `json:"desayuno"`
	Comida   bool   `json:"comida"`
	Merienda bool   `json:"merienda"`
	Notas    string `json:"notas"`
}

// PedidoComedorConNino es lo que ve el staff: la excepción más a qué niño
// pertenece.
type PedidoComedorConNino struct {
	PedidoComedor
	HijoNombre string `json:"hijo_nombre"`
}

// ResumenConteo es cuántos niños activos comen cada tiempo de comida ese
// día (total de niños activos menos las excepciones que marcaron "no").
type ResumenConteo struct {
	Desayuno int `json:"desayuno"`
	Comida   int `json:"comida"`
	Merienda int `json:"merienda"`
}

// ResumenComedorDia es lo que consulta cocina/catering antes de preparar el
// día: cuántas porciones de cada comida y quién tiene alguna excepción o
// nota especial (alergias, instrucciones).
type ResumenComedorDia struct {
	Fecha       string                 `json:"fecha"`
	TotalNinos  int                    `json:"total_ninos"`
	Resumen     ResumenConteo          `json:"resumen"`
	Excepciones []PedidoComedorConNino `json:"excepciones"`
}

func (s *Server) registrarRutasComedor(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)
	staff := middleware.RequireStaff()

	r.GET("/padre/hijos/:hijoId/pedidos-comedor", auth, s.handleListarPedidosComedorHijo)
	r.PUT("/padre/hijos/:hijoId/pedidos-comedor/:fecha", auth, s.handleGuardarPedidoComedor)

	r.GET("/pedidos-comedor", auth, staff, s.handleResumenComedor)
}

func (s *Server) handleListarPedidosComedorHijo(c *gin.Context) {
	userID, _ := c.Get("user_id")
	hijoID := c.Param("hijoId")

	if !s.hijoPerteneceAPadre(hijoID, userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "No tienes permiso para ver los pedidos de comedor de este niño"})
		return
	}

	desde := strings.TrimSpace(c.Query("desde"))
	if desde == "" {
		desde = hoyEnZonaLocal(time.Now())
	}
	hasta := strings.TrimSpace(c.Query("hasta"))
	if hasta == "" {
		inicio, err := time.Parse("2006-01-02", desde)
		if err != nil {
			inicio = time.Now()
		}
		hasta = inicio.AddDate(0, 0, 14).Format("2006-01-02")
	}

	rows, err := s.DB.Query(
		`SELECT id, hijo_id, fecha, desayuno, comida, merienda, COALESCE(notas, '')
         FROM pedidos_comedor
         WHERE hijo_id = $1 AND fecha BETWEEN $2 AND $3
         ORDER BY fecha ASC`,
		hijoID, desde, hasta,
	)
	if err != nil {
		log.Printf("Error al consultar los pedidos de comedor: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar los pedidos de comedor"})
		return
	}
	defer rows.Close()

	pedidos := []PedidoComedor{}
	for rows.Next() {
		var p PedidoComedor
		var fecha *string
		if err := rows.Scan(&p.ID, &p.HijoID, &fecha, &p.Desayuno, &p.Comida, &p.Merienda, &p.Notas); err != nil {
			continue
		}
		if f := soloFecha(fecha); f != nil {
			p.Fecha = *f
		}
		pedidos = append(pedidos, p)
	}
	c.JSON(http.StatusOK, pedidos)
}

func (s *Server) handleGuardarPedidoComedor(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	userID, _ := c.Get("user_id")
	hijoID := c.Param("hijoId")
	fecha := c.Param("fecha")

	if !s.hijoPerteneceAPadre(hijoID, userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "No tienes permiso para modificar los pedidos de comedor de este niño"})
		return
	}
	if _, err := time.Parse("2006-01-02", fecha); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Fecha inválida (usa YYYY-MM-DD)"})
		return
	}

	var input struct {
		Desayuno bool   `json:"desayuno"`
		Comida   bool   `json:"comida"`
		Merienda bool   `json:"merienda"`
		Notas    string `json:"notas"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}
	notas := strings.TrimSpace(input.Notas)

	// Sin ninguna excepción que registrar (come las tres comidas y sin
	// notas) no tiene caso guardar una fila -- se borra cualquier pedido
	// previo de ese día para volver al comedor por defecto.
	if input.Desayuno && input.Comida && input.Merienda && notas == "" {
		if _, err := s.DB.Exec(`DELETE FROM pedidos_comedor WHERE hijo_id = $1 AND fecha = $2`, hijoID, fecha); err != nil {
			log.Printf("No se pudo restablecer el pedido de comedor del hijo %v el %s: %v", hijoID, fecha, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo actualizar el pedido"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Restablecido al comedor por defecto"})
		return
	}

	if _, err := s.DB.Exec(
		`INSERT INTO pedidos_comedor (hijo_id, guarderia_id, fecha, desayuno, comida, merienda, notas, creado_por)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
         ON CONFLICT (hijo_id, fecha) DO UPDATE SET
             desayuno = EXCLUDED.desayuno,
             comida = EXCLUDED.comida,
             merienda = EXCLUDED.merienda,
             notas = EXCLUDED.notas,
             actualizado_en = CURRENT_TIMESTAMP`,
		hijoID, gID, fecha, input.Desayuno, input.Comida, input.Merienda, notas, userID,
	); err != nil {
		log.Printf("No se pudo guardar el pedido de comedor del hijo %v el %s: %v", hijoID, fecha, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo guardar el pedido"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pedido guardado"})
}

// handleResumenComedor es lo que consulta staff antes de pedirle a cocina o
// al catering las porciones del día: cuántos niños activos hay, cuántos
// comen cada tiempo de comida (descontando las excepciones) y el detalle de
// quién tiene alguna excepción o nota.
func (s *Server) handleResumenComedor(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")

	fecha := strings.TrimSpace(c.Query("fecha"))
	if fecha == "" {
		fecha = hoyEnZonaLocal(time.Now())
	}
	if _, err := time.Parse("2006-01-02", fecha); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Fecha inválida (usa YYYY-MM-DD)"})
		return
	}

	var totalNinos int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM hijos WHERE guarderia_id = $1 AND activo = true`, gID).Scan(&totalNinos); err != nil {
		log.Printf("Error al consultar el resumen del comedor (guardería %v, fecha %s): %v", gID, fecha, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar el resumen del comedor"})
		return
	}

	rows, err := s.DB.Query(
		`SELECT p.id, p.hijo_id, p.fecha, p.desayuno, p.comida, p.merienda, COALESCE(p.notas, ''), h.nombre_niño
         FROM pedidos_comedor p
         JOIN hijos h ON h.id = p.hijo_id
         WHERE p.guarderia_id = $1 AND p.fecha = $2 AND h.activo = true
         ORDER BY h.nombre_niño ASC`,
		gID, fecha,
	)
	if err != nil {
		log.Printf("Error al consultar el resumen del comedor: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar el resumen del comedor"})
		return
	}
	defer rows.Close()

	resumen := ResumenConteo{Desayuno: totalNinos, Comida: totalNinos, Merienda: totalNinos}
	excepciones := []PedidoComedorConNino{}
	for rows.Next() {
		var p PedidoComedorConNino
		var fechaRaw *string
		if err := rows.Scan(&p.ID, &p.HijoID, &fechaRaw, &p.Desayuno, &p.Comida, &p.Merienda, &p.Notas, &p.HijoNombre); err != nil {
			continue
		}
		if f := soloFecha(fechaRaw); f != nil {
			p.Fecha = *f
		}
		if !p.Desayuno {
			resumen.Desayuno--
		}
		if !p.Comida {
			resumen.Comida--
		}
		if !p.Merienda {
			resumen.Merienda--
		}
		excepciones = append(excepciones, p)
	}

	c.JSON(http.StatusOK, ResumenComedorDia{
		Fecha:       fecha,
		TotalNinos:  totalNinos,
		Resumen:     resumen,
		Excepciones: excepciones,
	})
}
