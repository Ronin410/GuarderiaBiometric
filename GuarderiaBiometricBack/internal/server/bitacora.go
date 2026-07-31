package server

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"biometrico/internal/middleware"
)

func (s *Server) registrarRutasBitacora(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)

	r.GET("/bitacora", auth, s.handleBitacora)
	r.GET("/reportes-asistencia", auth, s.handleReportesAsistencia)
	r.POST("/seguimiento", auth, s.handleGuardarSeguimiento)
	r.GET("/seguimiento/:hijo_id", auth, s.handleObtenerSeguimiento)
	// Nota: sin AuthMiddleware a propósito, para que sea accesible vía link de WhatsApp.
	r.GET("/publico/seguimiento/:token", s.handleSeguimientoPublico)
}

func (s *Server) handleBitacora(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	fechaQuery := c.Query("fecha")

	if fechaQuery == "" {
		fechaQuery = hoyEnZonaLocal(time.Now())
	}

	// Definimos el rango del día en formato ISO para Postgres. Esto cubre todo el
	// día sin importar el desfase de horas UTC.
	inicioDia := fechaQuery + " 00:00:00-07" // -07 es el offset de Mazatlán
	finDia := fechaQuery + " 23:59:59-07"

	query := `
		SELECT
			h.id,
			h.nombre_niño,
			COALESCE(ult_mov.tipo_movimiento, 'AUSENTE') as estatus,
			COALESCE(TO_CHAR(ult_mov.fecha_hora AT TIME ZONE 'America/Mazatlan', 'HH12:MI AM'), '--:--') as hora_formateada,
			COALESCE(ult_mov.aseado, true) as aseado,
			COALESCE(ult_mov.reporte_golpe, false) as reporte_golpe,
			COALESCE(ult_mov.observaciones, '') as observaciones
		FROM hijos h
		LEFT JOIN LATERAL (
			SELECT tipo_movimiento, fecha_hora, aseado, reporte_golpe, observaciones
			FROM asistencia
			WHERE hijo_id = h.id
			AND (fecha_hora >= $2::timestamptz AND fecha_hora <= $3::timestamptz)
			ORDER BY fecha_hora DESC
			LIMIT 1
		) ult_mov ON true
		WHERE h.guarderia_id = $1 AND h.activo = true
		ORDER BY h.nombre_niño ASC`

	rows, err := s.DB.Query(query, gID, inicioDia, finDia)
	if err != nil {
		log.Printf("Error SQL Bitacora: %v", err)
		c.JSON(500, gin.H{"error": "Error de base de datos"})
		return
	}
	defer rows.Close()

	var registros []map[string]interface{}
	for rows.Next() {
		var id int
		var niño, estatus, hora, obs string
		var aseado, golpe bool
		if err := rows.Scan(&id, &niño, &estatus, &hora, &aseado, &golpe, &obs); err != nil {
			continue
		}
		registros = append(registros, map[string]interface{}{
			"id": id, "hijo": niño, "estatus": estatus, "fecha_hora": hora,
			"aseado": aseado, "golpe": golpe, "observaciones": obs,
		})
	}

	if registros == nil {
		registros = []map[string]interface{}{}
	}
	c.JSON(200, registros)
}

func (s *Server) handleReportesAsistencia(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	inicio := c.Query("inicio")
	fin := c.Query("fin")

	if inicio == "" || fin == "" {
		hoy := hoyEnZonaLocal(time.Now())
		inicio = hoy
		fin = hoy
	}

	// El LEFT JOIN con seguimiento se hace por la fecha del registro.
	query := `
        SELECT
            TO_CHAR(a.fecha_hora AT TIME ZONE 'America/Mazatlan', 'YYYY-MM-DD HH24:MI:SS') as fecha_formateada,
            h.nombre_niño,
            p.nombre as tutor_nombre,
            a.tipo_movimiento,
            a.aseado,
            a.reporte_golpe,
            COALESCE(a.observaciones, '') as obs_asistencia,
            -- Campos pedagógicos de la bitácora
            COALESCE(s.desayuno, '') as desayuno,
            COALESCE(s.comida, '') as comida,
            COALESCE(s.merienda, '') as merienda,
            COALESCE(s.esfinter, '') as esfinter,
            COALESCE(s.durmio, false) as durmio,
            COALESCE(s.observaciones, '') as obs_pedagogicas
        FROM asistencia a
        INNER JOIN hijos h ON a.hijo_id = h.id
        INNER JOIN padres p ON a.padre_id = p.id
        -- Unimos con seguimiento usando el ID del hijo y la FECHA (sin hora) del movimiento
        LEFT JOIN seguimiento_diario s ON s.hijo_id = a.hijo_id
            AND s.fecha = (a.fecha_hora AT TIME ZONE 'America/Mazatlan')::date
        WHERE a.guarderia_id = $3
          AND (a.fecha_hora AT TIME ZONE 'America/Mazatlan')::date >= $1::date
          AND (a.fecha_hora AT TIME ZONE 'America/Mazatlan')::date <= $2::date
        ORDER BY a.fecha_hora DESC`

	rows, err := s.DB.Query(query, inicio, fin, gID)
	if err != nil {
		log.Printf("Error en reporte detallado: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar reportes"})
		return
	}
	defer rows.Close()

	var reportes []map[string]interface{}
	for rows.Next() {
		var fecha, niño, tutor, tipo, obsAsis, desayuno, comida, merienda, esfinter, obsPed string
		var aseado, golpe, durmio bool

		err := rows.Scan(
			&fecha, &niño, &tutor, &tipo, &aseado, &golpe, &obsAsis,
			&desayuno, &comida, &merienda, &esfinter, &durmio, &obsPed,
		)
		if err != nil {
			log.Printf("Error escaneando fila: %v", err)
			continue
		}

		reportes = append(reportes, map[string]interface{}{
			"fecha":          fecha,
			"hijo_nombre":    niño,
			"tutor_nombre":   tutor,
			"tipo":           tipo,
			"aseado":         aseado,
			"golpe":          golpe,
			"obs_asistencia": obsAsis,
			"bitacora": map[string]interface{}{
				"desayuno":      desayuno,
				"comida":        comida,
				"merienda":      merienda,
				"esfinter":      esfinter,
				"durmio":        durmio,
				"observaciones": obsPed,
			},
		})
	}

	c.JSON(http.StatusOK, reportes)
}

func (s *Server) handleGuardarSeguimiento(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")

	hijoID := c.PostForm("hijo_id")
	desayuno := c.PostForm("desayuno")
	comida := c.PostForm("comida")
	merienda := c.PostForm("merienda")
	esfinter := c.PostForm("esfinter")
	observaciones := c.PostForm("observaciones")
	durmio := c.PostForm("durmio") == "true"

	ahora := time.Now().In(zonaMazatlan())
	fechaHoy := ahora.Format("2006-01-02")

	var seguimientoID int
	query := `
    INSERT INTO seguimiento_diario
    (hijo_id, guarderia_id, fecha, desayuno, comida, merienda, esfinter, observaciones, durmio)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    ON CONFLICT (hijo_id, fecha)
    DO UPDATE SET
        desayuno = EXCLUDED.desayuno,
        comida = EXCLUDED.comida,
        merienda = EXCLUDED.merienda,
        esfinter = EXCLUDED.esfinter,
        observaciones = EXCLUDED.observaciones,
        durmio = EXCLUDED.durmio
    RETURNING id;`

	err := s.DB.QueryRow(query, hijoID, gID, fechaHoy, desayuno, comida, merienda, esfinter, observaciones, durmio).Scan(&seguimientoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo actualizar la bitácora"})
		return
	}

	if hijoIDNum, errConv := strconv.Atoi(hijoID); errConv == nil {
		detalle := fmt.Sprintf("Desayuno %s, Comida %s, Merienda %s", desayuno, comida, merienda)
		go s.notificarEvento(hijoIDNum, "BITACORA", detalle)
	}

	// Tutor configurado para recibir el link por WhatsApp.
	var padreID int
	var telefonoPadre string

	queryPadre := `
        SELECT p.id, p.celular
        FROM padres p
        JOIN tutor_hijos th ON th.padre_id = p.id
        WHERE th.hijo_id = $1 AND p.recibe_whatsapp = TRUE
        LIMIT 1`

	err = s.DB.QueryRow(queryPadre, hijoID).Scan(&padreID, &telefonoPadre)
	if err != nil {
		fmt.Println("Aviso: No se encontró tutor con WhatsApp activo:", err)
	}

	// Manejo de MÚLTIPLES FOTOS
	form, _ := c.MultipartForm()
	files := form.File["fotos"]
	var urlsSubidas []string

	for _, file := range files {
		nombreArchivo := fmt.Sprintf("guarderia_%v/hijo_%s/%s_%s_%s",
			gID, hijoID, fechaHoy, ahora.Format("150405"), file.Filename)

		key, errS3 := s.uploadToS3(file, nombreArchivo)
		if errS3 != nil {
			continue
		}

		_, errDBFoto := s.DB.Exec("INSERT INTO fotos_seguimiento (seguimiento_id, url) VALUES ($1, $2)", seguimientoID, key)
		if errDBFoto != nil {
			continue
		}
		if urlFirmada, errFirma := s.firmarURLFoto(key); errFirma == nil {
			urlsSubidas = append(urlsSubidas, urlFirmada)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"mensaje":        "Información guardada correctamente",
		"seguimiento_id": seguimientoID,
		"fotos_subidas":  len(urlsSubidas),
		"urls":           urlsSubidas,
		"padre_id":       padreID,
		"telefono_padre": telefonoPadre,
	})
}

type seguimientoCompleto struct {
	ID            int      `json:"id"`
	HijoID        int      `json:"hijo_id"`
	HijoNombre    string   `json:"hijo_nombre,omitempty"`
	Fecha         string   `json:"fecha"`
	Desayuno      string   `json:"desayuno"`
	Comida        string   `json:"comida"`
	Merienda      string   `json:"merienda"`
	Esfinter      string   `json:"esfinter"`
	Observaciones string   `json:"observaciones"`
	Durmio        bool     `json:"durmio"`
	Fotos         []string `json:"fotos"`
}

func (s *Server) handleObtenerSeguimiento(c *gin.Context) {
	hijoID := c.Param("hijo_id")

	// Si no viene en la URL, usamos la fecha de hoy.
	fechaConsulta := c.Query("fecha")
	if fechaConsulta == "" {
		fechaConsulta = hoyEnZonaLocal(time.Now())
	}

	var sc seguimientoCompleto

	querySeguimiento := `
        SELECT id, hijo_id, fecha, desayuno, comida, merienda, esfinter, observaciones, durmio
        FROM seguimiento_diario
        WHERE hijo_id = $1 AND fecha = $2`

	err := s.DB.QueryRow(querySeguimiento, hijoID, fechaConsulta).Scan(
		&sc.ID, &sc.HijoID, &sc.Fecha, &sc.Desayuno, &sc.Comida,
		&sc.Merienda, &sc.Esfinter, &sc.Observaciones, &sc.Durmio,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "No hay reporte disponible para la fecha: " + fechaConsulta,
		})
		return
	}
	if len(sc.Fecha) >= 10 {
		sc.Fecha = sc.Fecha[:10]
	}

	rows, err := s.DB.Query("SELECT url FROM fotos_seguimiento WHERE seguimiento_id = $1", sc.ID)
	if err == nil {
		defer rows.Close()
		sc.Fotos = []string{}
		for rows.Next() {
			var valorGuardado string
			if err := rows.Scan(&valorGuardado); err != nil {
				continue
			}
			if urlFirmada, errFirma := s.firmarURLFoto(valorGuardado); errFirma == nil {
				sc.Fotos = append(sc.Fotos, urlFirmada)
			}
		}
	}

	c.JSON(http.StatusOK, sc)
}

func (s *Server) handleSeguimientoPublico(c *gin.Context) {
	token := c.Param("token") // El UUID del niño
	fechaConsulta := c.Query("fecha")

	if fechaConsulta == "" {
		fechaConsulta = hoyEnZonaLocal(time.Now())
	}

	var sc seguimientoCompleto

	// Unimos la tabla hijos con seguimiento_diario usando el token.
	querySeguimiento := `
        SELECT
            s.id, s.hijo_id, h.nombre_niño, s.fecha, s.desayuno,
            s.comida, s.merienda, s.esfinter, s.observaciones, s.durmio
        FROM hijos h
        JOIN seguimiento_diario s ON h.id = s.hijo_id
        WHERE h.url_token = $1 AND s.fecha = $2`

	err := s.DB.QueryRow(querySeguimiento, token, fechaConsulta).Scan(
		&sc.ID, &sc.HijoID, &sc.HijoNombre, &sc.Fecha, &sc.Desayuno, &sc.Comida,
		&sc.Merienda, &sc.Esfinter, &sc.Observaciones, &sc.Durmio,
	)
	if err != nil {
		// Si no hay bitácora, al menos intentamos traer el nombre del niño para que
		// la pantalla no se vea vacía.
		var nombre string
		s.DB.QueryRow("SELECT nombre_niño FROM hijos WHERE url_token = $1", token).Scan(&nombre)

		c.JSON(http.StatusNotFound, gin.H{
			"hijo_nombre": nombre,
			"error":       "Aún no hay reporte para la fecha seleccionada.",
		})
		return
	}
	if len(sc.Fecha) >= 10 {
		sc.Fecha = sc.Fecha[:10]
	}

	rows, err := s.DB.Query("SELECT url FROM fotos_seguimiento WHERE seguimiento_id = $1", sc.ID)
	if err == nil {
		defer rows.Close()
		sc.Fotos = []string{}
		for rows.Next() {
			var valorGuardado string
			if err := rows.Scan(&valorGuardado); err != nil {
				continue
			}
			if urlFirmada, errFirma := s.firmarURLFoto(valorGuardado); errFirma == nil {
				sc.Fotos = append(sc.Fotos, urlFirmada)
			}
		}
	}

	c.JSON(http.StatusOK, sc)
}
