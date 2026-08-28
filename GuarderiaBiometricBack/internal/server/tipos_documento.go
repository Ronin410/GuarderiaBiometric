package server

import (
	"log"
	"net/http"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"

	"biometrico/internal/middleware"
)

// TipoDocumento es una fila del catálogo configurable de documentos que
// cada guardería le pide a sus familias (acta de nacimiento, CURP, etc.) --
// "Quiero que los documentos que se subirán por niño sea modificable por
// guardería, ellos decidirán qué documentos son los que les pedirán al
// papá". EnUso ayuda al panel a advertir antes de borrar un tipo que ya
// tiene documentos subidos con él (mismo criterio que NinosActivos en
// Grupo).
type TipoDocumento struct {
	ID     int    `json:"id"`
	Clave  string `json:"clave"`
	Nombre string `json:"nombre"`
	Orden  int    `json:"orden"`
	EnUso  int    `json:"en_uso"`
}

func (s *Server) registrarRutasTiposDocumento(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)
	// Misma área que /hijos/:id/documentos (documentos.go) -- es el mismo
	// checklist, solo que aquí se configura CUÁLES tipos existen en vez de
	// subir un archivo a uno de ellos.
	staff := middleware.RequireArea("perfiles")

	r.GET("/admin/tipos-documento", auth, staff, s.handleListarTiposDocumento)
	r.POST("/admin/tipos-documento", auth, staff, s.handleCrearTipoDocumento)
	r.PUT("/admin/tipos-documento/:id", auth, staff, s.handleRenombrarTipoDocumento)
	r.DELETE("/admin/tipos-documento/:id", auth, staff, s.handleEliminarTipoDocumento)
}

func (s *Server) handleListarTiposDocumento(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")

	rows, err := s.DB.Query(
		`SELECT t.id, t.clave, t.nombre, t.orden, COUNT(d.id)
         FROM tipos_documento t
         LEFT JOIN documentos_nino d ON d.guarderia_id = t.guarderia_id AND d.tipo = t.clave
         WHERE t.guarderia_id = $1
         GROUP BY t.id
         ORDER BY t.orden ASC, t.nombre ASC`,
		gID,
	)
	if err != nil {
		log.Printf("Error al consultar tipos de documento: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar tipos de documento"})
		return
	}
	defer rows.Close()

	tipos := []TipoDocumento{}
	for rows.Next() {
		var t TipoDocumento
		if err := rows.Scan(&t.ID, &t.Clave, &t.Nombre, &t.Orden, &t.EnUso); err != nil {
			continue
		}
		tipos = append(tipos, t)
	}
	c.JSON(http.StatusOK, tipos)
}

func (s *Server) handleCrearTipoDocumento(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")

	var input struct {
		Nombre string `json:"nombre"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Nombre) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El nombre del tipo de documento es obligatorio"})
		return
	}
	nombre := strings.TrimSpace(input.Nombre)
	if len(nombre) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El nombre es demasiado largo (máximo 100 caracteres)"})
		return
	}

	clave := slugificar(nombre)
	if clave == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ese nombre no se puede usar -- prueba con otro"})
		return
	}

	var nuevoID int
	err := s.DB.QueryRow(
		`INSERT INTO tipos_documento (guarderia_id, clave, nombre, orden)
         VALUES ($1, $2, $3, COALESCE((SELECT MAX(orden) + 1 FROM tipos_documento WHERE guarderia_id = $1), 0))
         RETURNING id`,
		gID, clave, nombre,
	).Scan(&nuevoID)
	if err != nil {
		if esClaveTipoDocumentoDuplicada(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "Ya existe un tipo de documento con ese nombre"})
			return
		}
		log.Printf("No se pudo crear el tipo de documento: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo crear el tipo de documento"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": nuevoID, "clave": clave, "message": "Tipo de documento creado"})
}

// handleRenombrarTipoDocumento cambia el nombre visible pero NO la clave --
// la clave es lo que documentos_nino.tipo guarda por cada archivo ya
// subido, así que cambiarla desligaría esos documentos de este tipo. Mismo
// criterio que Pago.Concepto: la etiqueta se puede pulir sin tocar lo que
// ya hay guardado.
func (s *Server) handleRenombrarTipoDocumento(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	tipoID := c.Param("id")

	var input struct {
		Nombre string `json:"nombre"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Nombre) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El nombre del tipo de documento es obligatorio"})
		return
	}

	res, err := s.DB.Exec(
		`UPDATE tipos_documento SET nombre = $1 WHERE id = $2 AND guarderia_id = $3`,
		strings.TrimSpace(input.Nombre), tipoID, gID,
	)
	if err != nil {
		log.Printf("No se pudo renombrar el tipo de documento: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo renombrar el tipo de documento"})
		return
	}
	if filas, _ := res.RowsAffected(); filas == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tipo de documento no encontrado"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Tipo de documento renombrado"})
}

// handleEliminarTipoDocumento rechaza el borrado si ya hay documentos
// subidos con ese tipo -- mismo criterio de "nada desaparece sin un paso
// explícito" que handleEliminarGrupo: obliga a eliminar esos documentos
// primero (o dejarlos, y no borrar el tipo) en vez de que sus archivos se
// queden huérfanos de catálogo en silencio.
func (s *Server) handleEliminarTipoDocumento(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	tipoID := c.Param("id")

	var clave string
	if err := s.DB.QueryRow(`SELECT clave FROM tipos_documento WHERE id = $1 AND guarderia_id = $2`, tipoID, gID).Scan(&clave); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tipo de documento no encontrado"})
		return
	}

	var enUso int
	if err := s.DB.QueryRow(
		`SELECT COUNT(*) FROM documentos_nino WHERE guarderia_id = $1 AND tipo = $2`,
		gID, clave,
	).Scan(&enUso); err != nil {
		log.Printf("Error al verificar el tipo de documento: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al verificar el tipo de documento"})
		return
	}
	if enUso > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"error":  "Ya hay documentos subidos con este tipo. Elimínalos primero desde el expediente de cada niño.",
			"en_uso": enUso,
		})
		return
	}

	res, err := s.DB.Exec(`DELETE FROM tipos_documento WHERE id = $1 AND guarderia_id = $2`, tipoID, gID)
	if err != nil {
		log.Printf("No se pudo eliminar el tipo de documento: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo eliminar el tipo de documento"})
		return
	}
	if filas, _ := res.RowsAffected(); filas == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tipo de documento no encontrado"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Tipo de documento eliminado"})
}

// slugificar convierte un nombre libre ("Constancia de Estudios") en una
// clave estable para guardar en documentos_nino.tipo ("constancia_de_estudios")
// -- minúsculas, sin acentos, espacios/símbolos colapsados a "_". Trunca a
// 50 (el ancho de tipos_documento.clave) para no reventar el INSERT con un
// nombre largo.
func slugificar(nombre string) string {
	reemplazosAcentos := map[rune]rune{
		'á': 'a', 'é': 'e', 'í': 'i', 'ó': 'o', 'ú': 'u', 'ñ': 'n', 'ü': 'u',
		'Á': 'a', 'É': 'e', 'Í': 'i', 'Ó': 'o', 'Ú': 'u', 'Ñ': 'n', 'Ü': 'u',
	}

	var b strings.Builder
	ultimoFueGuion := true // evita un "_" inicial si el nombre empieza con símbolos
	for _, r := range nombre {
		if reemplazo, ok := reemplazosAcentos[r]; ok {
			r = reemplazo
		}
		r = unicode.ToLower(r)
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			ultimoFueGuion = false
		default:
			if !ultimoFueGuion {
				b.WriteRune('_')
				ultimoFueGuion = true
			}
		}
	}
	clave := strings.Trim(b.String(), "_")
	if len(clave) > 50 {
		clave = strings.Trim(clave[:50], "_")
	}
	return clave
}

func esClaveTipoDocumentoDuplicada(err error) bool {
	pqErr, ok := err.(*pq.Error)
	return ok && pqErr.Code == "23505" && pqErr.Constraint == "tipos_documento_guarderia_id_clave_key"
}
