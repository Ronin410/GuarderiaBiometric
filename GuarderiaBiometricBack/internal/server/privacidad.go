package server

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"biometrico/internal/middleware"
)

func (s *Server) registrarRutasPrivacidad(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)
	// RequireAdmin, no RequireArea: Configuración es exclusiva del admin --
	// el staff no debe poder entrar aquí nunca, ni siquiera con permisos
	// personalizados (ver el comentario de AreasPermiso en personal.go).
	admin := middleware.RequireAdmin()
	r.GET("/aviso-privacidad", auth, s.handleObtenerAviso)
	r.PUT("/admin/aviso-privacidad", auth, admin, s.handleActualizarAviso)
	r.GET("/admin/aviso-privacidad/estadisticas", auth, admin, s.handleEstadisticasAviso)
}

func (s *Server) handleObtenerAviso(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")

	var texto, version string
	var pdfKey sql.NullString
	err := s.DB.QueryRow(
		"SELECT COALESCE(aviso_privacidad_texto, ''), aviso_privacidad_version, aviso_privacidad_pdf_s3_key FROM guarderias WHERE id = $1",
		gID,
	).Scan(&texto, &version, &pdfKey)
	if err != nil {
		log.Printf("No se pudo consultar el Aviso de Privacidad: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo consultar el Aviso de Privacidad"})
		return
	}

	respuesta := gin.H{
		"texto":   texto,
		"version": version,
		// configurado: hay aviso listo para mostrar si hay texto O un PDF --
		// son alternativas, no se necesitan las dos (ver el comentario largo
		// en handleActualizarAviso).
		"configurado": strings.TrimSpace(texto) != "" || pdfKey.Valid,
	}
	if pdfKey.Valid {
		if url, err := s.firmarURLFoto(pdfKey.String); err == nil {
			respuesta["pdf_url"] = url
		} else {
			log.Printf("No se pudo firmar el PDF del Aviso de Privacidad (guardería %v): %v", gID, err)
		}
	}

	c.JSON(http.StatusOK, respuesta)
}

// handleActualizarAviso recibe multipart/form-data (antes era JSON-solo-
// texto) para poder traer un PDF en vez de texto pegado a mano -- "en lugar
// de escribir el aviso, poner un archivo PDF". Son alternativas: subir un
// PDF nuevo borra el texto que hubiera (y viceversa), para que nunca quede
// ambiguo cuál de los dos es "el aviso vigente" al mostrárselo al tutor en
// el kiosco (ver AvisoPrivacidadModal.jsx, que muestra uno u otro según
// cuál venga en la respuesta).
func (s *Server) handleActualizarAviso(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	texto := strings.TrimSpace(c.PostForm("texto"))

	var pdfKeyNueva *string
	fileHeader, errArchivo := c.FormFile("pdf")
	if errArchivo == nil {
		if fileHeader.Size > maxTamanoDocumento {
			c.JSON(http.StatusBadRequest, gin.H{"error": "El PDF no puede pesar más de 10 MB"})
			return
		}
		contentType := fileHeader.Header.Get("Content-Type")
		if contentType != "application/pdf" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "El archivo debe ser un PDF"})
			return
		}
		key := fmt.Sprintf("aviso-privacidad/guarderia_%v/%d_%s", gID, time.Now().UnixNano(), fileHeader.Filename)
		if _, err := s.uploadToS3(fileHeader, key, contentType); err != nil {
			log.Printf("No se pudo subir el PDF del Aviso de Privacidad a S3: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo subir el PDF"})
			return
		}
		pdfKeyNueva = &key
		texto = "" // el PDF reemplaza al texto, no conviven los dos a la vez
	} else if texto == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Escribe el texto del Aviso de Privacidad o sube un PDF"})
		return
	}

	var versionActual string
	var pdfKeyAnterior sql.NullString
	if err := s.DB.QueryRow("SELECT aviso_privacidad_version, aviso_privacidad_pdf_s3_key FROM guarderias WHERE id = $1", gID).Scan(&versionActual, &pdfKeyAnterior); err != nil {
		log.Printf("No se pudo consultar la versión actual del Aviso de Privacidad (guardería %v): %v", gID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo consultar la versión actual"})
		return
	}

	nuevaVersion := siguienteVersion(versionActual)

	_, err := s.DB.Exec(
		"UPDATE guarderias SET aviso_privacidad_texto = $1, aviso_privacidad_version = $2, aviso_privacidad_pdf_s3_key = $3 WHERE id = $4",
		texto, nuevaVersion, pdfKeyNueva, gID,
	)
	if err != nil {
		if pdfKeyNueva != nil {
			go s.borrarDeS3(*pdfKeyNueva) // no se guardó, no dejamos el PDF huérfano
		}
		log.Printf("No se pudo guardar el Aviso de Privacidad: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo guardar el Aviso de Privacidad"})
		return
	}

	// El PDF (o texto) que quedó reemplazado ya no lo referencia nadie --
	// se limpia de S3 aparte, después de confirmar que el UPDATE sí pasó.
	if pdfKeyAnterior.Valid && (pdfKeyNueva == nil || *pdfKeyNueva != pdfKeyAnterior.String) {
		go s.borrarDeS3(pdfKeyAnterior.String)
	}

	c.JSON(http.StatusOK, gin.H{"status": "Aviso de Privacidad actualizado", "version": nuevaVersion})
}

func (s *Server) handleEstadisticasAviso(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")

	var total int
	err := s.DB.QueryRow(
		"SELECT COUNT(*) FROM consentimientos WHERE guarderia_id = $1",
		gID,
	).Scan(&total)
	if err != nil {
		log.Printf("No se pudo consultar los consentimientos: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo consultar los consentimientos"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"total_consentimientos": total})
}

// siguienteVersion incrementa una versión con forma "vN" (ej. "v1" -> "v2").
// Si el formato no es el esperado (dato viejo/manual), reinicia en "v1" en
// vez de fallar — el punto es que cada guardado quede en una versión nueva
// y distinguible, no reconstruir un historial perfecto de versiones previas.
func siguienteVersion(actual string) string {
	numero, err := strconv.Atoi(strings.TrimPrefix(actual, "v"))
	if err != nil {
		return "v1"
	}
	return "v" + strconv.Itoa(numero+1)
}
