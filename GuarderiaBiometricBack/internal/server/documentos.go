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

// maxTamanoDocumento evita que alguien suba un archivo desproporcionado
// (fotos de documentos bien comprimidas o PDFs escaneados caben de sobra).
const maxTamanoDocumento = 10 << 20 // 10 MB

// DocumentoNino es una fila del checklist de documentos de inscripción de un
// niño. Los campos van en puntero porque, si el tipo todavía no se ha
// subido, solo viaja "tipo"/"nombre" (el catálogo configurado de la
// guardería, ver tipos_documento.go) — el resto queda en null.
type DocumentoNino struct {
	Tipo          string  `json:"tipo"`
	Nombre        string  `json:"nombre"`
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
	// Espejo de /padre/hijos/:hijoId/galeria en galeria.go: mismo checklist
	// que ve el staff, pero solo lectura y solo para hijos propios -- "Quiero
	// que en la parte de expediente los papás puedan ver cuáles son los
	// documentos que han entregado a la guardería y cuáles son los que les
	// falta".
	r.GET("/padre/hijos/:hijoId/documentos", auth, s.handleListarDocumentosPadre)
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

	docs, err := s.obtenerChecklistDocumentos(gID, hijoID)
	if err != nil {
		log.Printf("Error al consultar documentos: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar documentos"})
		return
	}
	c.JSON(http.StatusOK, docs)
}

// handleListarDocumentosPadre es el mismo checklist que ve el staff, en
// solo lectura -- "Quiero que en la parte de expediente los papás puedan
// ver cuáles son los documentos que han entregado a la guardería y cuáles
// son los que les falta". Mismo criterio de permiso que handleGaleriaPadre
// en galeria.go: el hijo tiene que ser suyo.
func (s *Server) handleListarDocumentosPadre(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	userID, _ := c.Get("user_id")
	hijoID := c.Param("hijoId")

	if !s.hijoPerteneceAPadre(hijoID, userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "No tienes permiso para ver los documentos de este niño"})
		return
	}

	docs, err := s.obtenerChecklistDocumentos(gID, hijoID)
	if err != nil {
		log.Printf("Error al consultar documentos: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar documentos"})
		return
	}
	c.JSON(http.StatusOK, docs)
}

// obtenerChecklistDocumentos junta el catálogo configurado de la guardería
// (tipos_documento, ver tipos_documento.go) con lo que ESTE niño ya tiene
// subido -- un LEFT JOIN en vez del mapa+loop de antes, porque ahora el
// catálogo es dinámico por guardería, no una lista fija en memoria. Se
// comparte entre el checklist de staff y el de solo lectura del papá, que
// solo cambia cómo se verificó el permiso para llegar aquí (mismo criterio
// que obtenerGaleria en galeria.go).
func (s *Server) obtenerChecklistDocumentos(gID, hijoID any) ([]DocumentoNino, error) {
	rows, err := s.DB.Query(`
        SELECT t.clave, t.nombre, d.nombre_archivo, d.s3_key, d.subido_en
        FROM tipos_documento t
        LEFT JOIN documentos_nino d ON d.guarderia_id = t.guarderia_id AND d.tipo = t.clave AND d.hijo_id = $2
        WHERE t.guarderia_id = $1
        ORDER BY t.orden ASC, t.nombre ASC`,
		gID, hijoID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	docs := []DocumentoNino{}
	for rows.Next() {
		var clave, nombreTipo string
		var nombreArchivo, key sql.NullString
		var subido sql.NullTime
		if err := rows.Scan(&clave, &nombreTipo, &nombreArchivo, &key, &subido); err != nil {
			continue
		}
		d := DocumentoNino{Tipo: clave, Nombre: nombreTipo}
		if nombreArchivo.Valid {
			d.NombreArchivo = &nombreArchivo.String
		}
		if subido.Valid {
			f := subido.Time.Format(time.RFC3339)
			d.SubidoEn = &f
		}
		if key.Valid {
			if url, err := s.firmarURLFoto(key.String); err == nil {
				d.URL = &url
			}
		}
		docs = append(docs, d)
	}
	return docs, nil
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
	var tipoValido bool
	if err := s.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM tipos_documento WHERE guarderia_id = $1 AND clave = $2)`, gID, tipo).Scan(&tipoValido); err != nil {
		log.Printf("Error al validar el tipo de documento: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al validar el tipo de documento"})
		return
	}
	if !tipoValido {
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
