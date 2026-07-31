package server

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/gin-gonic/gin"

	"biometrico/internal/middleware"
)

// registrarRutasArco expone el soporte técnico para los derechos ARCO
// (Acceso, Rectificación, Cancelación, Oposición) de la LFPDPPP: que un
// padre pueda pedir copia de sus datos, o que se borren.
func (s *Server) registrarRutasArco(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)
	staff := middleware.RequireStaff()
	r.GET("/admin/familias/:padre_id/exportar", auth, staff, s.handleExportarFamilia)
	r.DELETE("/admin/familias/:padre_id", auth, staff, s.handleEliminarFamilia)
}

type consentimientoExport struct {
	VersionAviso string `json:"version_aviso"`
	AceptadoEn   string `json:"aceptado_en"`
	IP           string `json:"ip"`
}

type bitacoraExport struct {
	Fecha         string `json:"fecha"`
	Desayuno      string `json:"desayuno"`
	Comida        string `json:"comida"`
	Merienda      string `json:"merienda"`
	Esfinter      string `json:"esfinter"`
	Observaciones string `json:"observaciones"`
	Durmio        bool   `json:"durmio"`
}

type hijoExport struct {
	NinoPerfil
	Pagos     []Pago           `json:"pagos"`
	Bitacoras []bitacoraExport `json:"bitacoras"`
}

// handleExportarFamilia arma el expediente completo de un padre (derecho de
// acceso): su perfil, el historial de consentimientos que dio, y por cada
// hijo vinculado su perfil, pagos y bitácora — el mismo dato que ya puede
// ver un padre autenticado desde el portal, empaquetado para descarga.
func (s *Server) handleExportarFamilia(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	padreID := c.Param("padre_id")

	var padre struct {
		ID             int    `json:"id"`
		Nombre         string `json:"nombre"`
		Celular        string `json:"celular"`
		RecibeWhatsapp bool   `json:"recibe_whatsapp"`
	}
	err := s.DB.QueryRow(
		"SELECT id, nombre, COALESCE(celular, ''), COALESCE(recibe_whatsapp, false) FROM padres WHERE id = $1 AND guarderia_id = $2",
		padreID, gID,
	).Scan(&padre.ID, &padre.Nombre, &padre.Celular, &padre.RecibeWhatsapp)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tutor no encontrado en esta guardería"})
		return
	}

	consentimientos := []consentimientoExport{}
	rowsCons, err := s.DB.Query(
		"SELECT version_aviso, aceptado_en, COALESCE(ip, '') FROM consentimientos WHERE padre_id = $1 ORDER BY aceptado_en DESC",
		padreID,
	)
	if err == nil {
		defer rowsCons.Close()
		for rowsCons.Next() {
			var ce consentimientoExport
			if err := rowsCons.Scan(&ce.VersionAviso, &ce.AceptadoEn, &ce.IP); err == nil {
				consentimientos = append(consentimientos, ce)
			}
		}
	}

	hijos := []hijoExport{}
	rowsHijos, err := s.DB.Query(`
        SELECT h.id, h.nombre_niño, h.activo, h.fecha_nacimiento, h.direccion,
               h.contacto_emergencia_nombre, h.contacto_emergencia_telefono, h.colegiatura_mensual
        FROM hijos h
        JOIN tutor_hijos th ON h.id = th.hijo_id
        WHERE th.padre_id = $1 AND th.guarderia_id = $2`, padreID, gID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar hijos vinculados"})
		return
	}
	defer rowsHijos.Close()

	for rowsHijos.Next() {
		var he hijoExport
		if err := rowsHijos.Scan(
			&he.ID, &he.Nombre, &he.Activo, &he.FechaNacimiento, &he.Direccion,
			&he.ContactoEmergenciaNombre, &he.ContactoEmergenciaTelefono, &he.ColegiaturaMensual,
		); err != nil {
			continue
		}
		he.FechaNacimiento = soloFecha(he.FechaNacimiento)
		he.Pagos = obtenerPagosDeHijo(s.DB, he.ID, gID)
		he.Bitacoras = obtenerBitacorasDeHijo(s.DB, he.ID, gID)
		hijos = append(hijos, he)
	}

	exportacion := gin.H{
		"padre":           padre,
		"consentimientos": consentimientos,
		"hijos":           hijos,
		"generado_en":     time.Now().Format(time.RFC3339),
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=expediente_padre_%s.json", padreID))
	c.JSON(http.StatusOK, exportacion)
}

func obtenerPagosDeHijo(db *sql.DB, hijoID int, guarderiaID any) []Pago {
	pagos := []Pago{}
	rows, err := db.Query(`
        SELECT id, hijo_id, monto, concepto, periodo, fecha_pago, metodo_pago, COALESCE(observaciones, '')
        FROM pagos WHERE hijo_id = $1 AND guarderia_id = $2 ORDER BY fecha_pago DESC`, hijoID, guarderiaID)
	if err != nil {
		return pagos
	}
	defer rows.Close()
	for rows.Next() {
		var p Pago
		if err := rows.Scan(&p.ID, &p.HijoID, &p.Monto, &p.Concepto, &p.Periodo, &p.FechaPago, &p.MetodoPago, &p.Observaciones); err == nil {
			if len(p.FechaPago) >= 10 {
				p.FechaPago = p.FechaPago[:10]
			}
			pagos = append(pagos, p)
		}
	}
	return pagos
}

func obtenerBitacorasDeHijo(db *sql.DB, hijoID int, guarderiaID any) []bitacoraExport {
	bitacoras := []bitacoraExport{}
	rows, err := db.Query(`
        SELECT fecha, desayuno, comida, merienda, esfinter, COALESCE(observaciones, ''), durmio
        FROM seguimiento_diario WHERE hijo_id = $1 AND guarderia_id = $2 ORDER BY fecha DESC`, hijoID, guarderiaID)
	if err != nil {
		return bitacoras
	}
	defer rows.Close()
	for rows.Next() {
		var b bitacoraExport
		if err := rows.Scan(&b.Fecha, &b.Desayuno, &b.Comida, &b.Merienda, &b.Esfinter, &b.Observaciones, &b.Durmio); err == nil {
			if len(b.Fecha) >= 10 {
				b.Fecha = b.Fecha[:10]
			}
			bitacoras = append(bitacoras, b)
		}
	}
	return bitacoras
}

// handleEliminarFamilia atiende el derecho de cancelación: borra el perfil
// del padre, su rostro de Rekognition, su cuenta de notificaciones push y su
// vínculo con sus hijos. NO borra la bitácora/asistencia/pagos de los
// niños — son el expediente del niño, no del padre, y pueden ser necesarios
// para otro tutor o por obligaciones de la guardería.
func (s *Server) handleEliminarFamilia(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	padreID := c.Param("padre_id")

	var faceID string
	if err := s.DB.QueryRow(
		"SELECT face_id FROM padres WHERE id = $1 AND guarderia_id = $2", padreID, gID,
	).Scan(&faceID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tutor no encontrado en esta guardería"})
		return
	}

	rekognitionEliminado := false
	colID := getCollectionID(gID)
	_, err := s.Rek.DeleteFaces(context.TODO(), &rekognition.DeleteFacesInput{
		CollectionId: aws.String(colID),
		FaceIds:      []string{faceID},
	})
	if err != nil {
		log.Printf("handleEliminarFamilia: no se pudo borrar el rostro de Rekognition (padre %s): %v", padreID, err)
	} else {
		rekognitionEliminado = true
	}

	// Se desvincula (no se borra) el historial de asistencia ANTES de borrar
	// al padre: asistencia.padre_id tiene ON DELETE CASCADE pensado para
	// integridad referencial normal, no para este borrado de alcance
	// acotado — sin este paso se perdería también la asistencia del niño.
	if _, err := s.DB.Exec("UPDATE asistencia SET padre_id = NULL WHERE padre_id = $1", padreID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo desvincular el historial de asistencia"})
		return
	}

	if _, err := s.DB.Exec("DELETE FROM push_subscripciones WHERE padre_id = $1", padreID); err != nil {
		log.Printf("handleEliminarFamilia: no se pudieron borrar las suscripciones push del padre %s: %v", padreID, err)
	}

	result, err := s.DB.Exec("DELETE FROM padres WHERE id = $1 AND guarderia_id = $2", padreID, gID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo eliminar al tutor"})
		return
	}
	filas, _ := result.RowsAffected()
	if filas == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tutor no encontrado en esta guardería"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":                "Datos del tutor eliminados",
		"rekognition_eliminado": rekognitionEliminado,
	})
}
