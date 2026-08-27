package server

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"biometrico/internal/middleware"
)

// ordenTiposDocumento fija el catálogo y el orden en el que siempre se
// presenta — el frontend recibe las 6 filas (subido o no) en este orden,
// para que la lista de documentos de un niño no salte de lugar según lo que
// ya se subió. Las etiquetas legibles ("Acta de Nacimiento", etc.) viven
// solo en el frontend, mismo criterio que Pago.Concepto.
var ordenTiposDocumento = []string{
	"acta_nacimiento", "curp", "comprobante_domicilio",
	"cartilla_vacunacion", "identificacion_tutor", "otro",
}

var tiposDocumentoValidos = map[string]bool{
	"acta_nacimiento":       true,
	"curp":                  true,
	"comprobante_domicilio": true,
	"cartilla_vacunacion":   true,
	"identificacion_tutor":  true,
	"otro":                  true,
}

// maxTamanoDocumento evita que alguien suba un archivo desproporcionado
// (fotos de documentos bien comprimidas o PDFs escaneados caben de sobra).
const maxTamanoDocumento = 10 << 20 // 10 MB

// DocumentoNino es una fila del checklist de documentos de inscripción de un
// niño. Los campos van en puntero porque, si el tipo todavía no se ha
// subido, solo viaja el "tipo" — el resto queda en null.
type DocumentoNino struct {
	Tipo          string  `json:"tipo"`
	NombreArchivo *string `json:"nombre_archivo"`
	SubidoEn      *string `json:"subido_en"`
	URL           *string `json:"url"`
}

func (s *Server) registrarRutasDocumentos(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)
	staff := middleware.RequireArea("perfiles")

	r.GET("/hijos/:id/documentos", auth, staff, s.handleListarDocumentos)
	r.POST("/hijos/:id/documentos", auth, staff, s.handleSubirDocumento)
	r.DELETE("/hijos/:id/documentos/:tipo", auth, staff, s.handleEliminarDocumento)
}

// hijoPerteneceAGuarderia evita que alguien con sesión de UNA guardería lea o
// modifique documentos de un niño de OTRA guardería con solo cambiar el id
// en la URL — el resto de rutas de hijos ya filtra así, esto lo replica
// antes de tocar S3.
func (s *Server) hijoPerteneceAGuarderia(hijoID string, gID any) bool {
	var existe bool
	err := s.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM hijos WHERE id = $1 AND guarderia_id = $2)`, hijoID, gID).Scan(&existe)
	return err == nil && existe
}

func (s *Server) handleListarDocumentos(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	hijoID := c.Param("id")

	if !s.hijoPerteneceAGuarderia(hijoID, gID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Niño no encontrado"})
		return
	}

	rows, err := s.DB.Query(
		`SELECT tipo, nombre_archivo, s3_key, subido_en FROM documentos_nino WHERE hijo_id = $1`,
		hijoID,
	)
	if err != nil {
		log.Printf("Error al consultar documentos: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar documentos"})
		return
	}
	defer rows.Close()

	type existente struct {
		nombre, key string
		subido      time.Time
	}
	subidos := map[string]existente{}
	for rows.Next() {
		var tipo, nombre, key string
		var subido time.Time
		if err := rows.Scan(&tipo, &nombre, &key, &subido); err != nil {
			continue
		}
		subidos[tipo] = existente{nombre, key, subido}
	}

	docs := make([]DocumentoNino, 0, len(ordenTiposDocumento))
	for _, tipo := range ordenTiposDocumento {
		d := DocumentoNino{Tipo: tipo}
		if e, ok := subidos[tipo]; ok {
			nombre := e.nombre
			subido := e.subido.Format(time.RFC3339)
			d.NombreArchivo = &nombre
			d.SubidoEn = &subido
			if url, err := s.firmarURLFoto(e.key); err == nil {
				d.URL = &url
			}
		}
		docs = append(docs, d)
	}
	c.JSON(http.StatusOK, docs)
}

func (s *Server) handleSubirDocumento(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	userID, _ := c.Get("user_id")
	hijoID := c.Param("id")

	if !s.hijoPerteneceAGuarderia(hijoID, gID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Niño no encontrado"})
		return
	}

	tipo := c.PostForm("tipo")
	if !tiposDocumentoValidos[tipo] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tipo de documento no válido"})
		return
	}

	fileHeader, err := c.FormFile("archivo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Selecciona un archivo"})
		return
	}
	if fileHeader.Size > maxTamanoDocumento {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El archivo no puede pesar más de 10 MB"})
		return
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Key con timestamp: si algo falla a medias, el documento anterior sigue
	// intacto en S3 hasta que el UPSERT de abajo confirma el reemplazo.
	key := fmt.Sprintf("documentos/guarderia_%v/hijo_%s/%s_%d_%s",
		gID, hijoID, tipo, time.Now().UnixNano(), fileHeader.Filename)

	var keyAnterior sql.NullString
	s.DB.QueryRow(`SELECT s3_key FROM documentos_nino WHERE hijo_id = $1 AND tipo = $2`, hijoID, tipo).Scan(&keyAnterior)

	if _, err := s.uploadToS3(fileHeader, key, contentType); err != nil {
		log.Printf("handleSubirDocumento: fallo al subir a S3 (hijo %s, tipo %s): %v", hijoID, tipo, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo subir el archivo"})
		return
	}

	_, err = s.DB.Exec(`
        INSERT INTO documentos_nino (hijo_id, guarderia_id, tipo, nombre_archivo, s3_key)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (hijo_id, tipo) DO UPDATE SET
            nombre_archivo = EXCLUDED.nombre_archivo,
            s3_key = EXCLUDED.s3_key,
            subido_en = CURRENT_TIMESTAMP`,
		hijoID, gID, tipo, fileHeader.Filename, key,
	)
	if err != nil {
		go s.borrarDeS3(key) // el registro no se guardó, no dejamos el archivo huérfano
		log.Printf("No se pudo guardar el documento: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo guardar el documento"})
		return
	}

	if keyAnterior.Valid && keyAnterior.String != key {
		go s.borrarDeS3(keyAnterior.String)
	}

	urlFirmada, _ := s.firmarURLFoto(key)
	s.registrarAcceso("documento_subido", gID, userID, fmt.Sprintf("hijo %s: %s", hijoID, tipo), c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "Documento guardado", "url": urlFirmada})
}

func (s *Server) handleEliminarDocumento(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	userID, _ := c.Get("user_id")
	hijoID := c.Param("id")
	tipo := c.Param("tipo")

	if !s.hijoPerteneceAGuarderia(hijoID, gID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Niño no encontrado"})
		return
	}

	var key string
	err := s.DB.QueryRow(
		`SELECT s3_key FROM documentos_nino WHERE hijo_id = $1 AND tipo = $2`,
		hijoID, tipo,
	).Scan(&key)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Documento no encontrado"})
		return
	} else if err != nil {
		log.Printf("Error al buscar el documento: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al buscar el documento"})
		return
	}

	if _, err := s.DB.Exec(`DELETE FROM documentos_nino WHERE hijo_id = $1 AND tipo = $2`, hijoID, tipo); err != nil {
		log.Printf("No se pudo eliminar el documento %s del hijo %s: %v", tipo, hijoID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo eliminar el documento"})
		return
	}

	go s.borrarDeS3(key)
	s.registrarAcceso("documento_eliminado", gID, userID, fmt.Sprintf("hijo %s: %s", hijoID, tipo), c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"message": "Documento eliminado"})
}
