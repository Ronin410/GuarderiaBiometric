package server

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"biometrico/internal/middleware"
)

// Circular es un aviso que el admin o staff manda a todos los padres de la
// guardería (inscripciones, eventos, cierres, etc.), sin ligarse a ningún
// niño en particular. ImagenURL va en puntero porque la mayoría de las
// circulares son solo texto -- si es nil, el frontend no muestra nada de
// imagen.
type Circular struct {
	ID        int     `json:"id"`
	Titulo    string  `json:"titulo"`
	Contenido string  `json:"contenido"`
	CreadoEn  string  `json:"creado_en"`
	ImagenURL *string `json:"imagen_url,omitempty"`
	// ParaTodos + Grupos: a quién va dirigida (ver destinatarios.go). Con
	// ParaTodos en true, Grupos viene vacío y la circular es para toda la
	// guardería, que es como se comportaban todas antes de esto.
	ParaTodos bool           `json:"para_todos"`
	Grupos    []GrupoDestino `json:"grupos"`
}

// CircularConLecturas es lo que ve el staff en el listado: la circular más
// cuántas familias ya la leyeron, sobre el total de familias de la
// guardería (ej. "5 de 8 familias").
type CircularConLecturas struct {
	Circular
	LeidoPor      int `json:"leido_por"`
	TotalFamilias int `json:"total_familias"`
}

// LecturaCircular es una fila del detalle "quién la ha leído" (staff).
type LecturaCircular struct {
	Nombre  string `json:"nombre"`
	LeidoEn string `json:"leido_en"`
}

func (s *Server) registrarRutasCirculares(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)
	staff := middleware.RequireArea("circulares")

	r.GET("/circulares", auth, staff, s.handleListarCircularesStaff)
	r.POST("/circulares", auth, staff, s.handleCrearCircular)
	r.DELETE("/circulares/:id", auth, staff, s.handleEliminarCircular)
	r.GET("/circulares/:id/lecturas", auth, staff, s.handleDetalleLecturasCircular)

	// Un papá también debe poder leer los avisos que le mandan (mismo
	// criterio que /padre/menu-semanal) y marcarlos como leídos, sin poder
	// publicar ni borrar.
	r.GET("/padre/circulares", auth, s.handleListarCircularesPadre)
	r.POST("/padre/circulares/:id/leido", auth, s.handleMarcarCircularLeida)
}

// handleListarCircularesStaff incluye, a diferencia de la vista del padre,
// cuántas familias ya leyeron cada circular -- el "verifica quiénes han
// leído tus mensajes" del PDF de referencia.
func (s *Server) handleListarCircularesStaff(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")

	// COUNT(DISTINCT cl.padre_id) y no COUNT(cl.padre_id): el total de
	// familias destinatarias ahora se calcula con subconsultas que unen
	// grupos e hijos, y cualquier join que se agregue a este SELECT
	// multiplicaría las filas de lecturas. DISTINCT lo deja a salvo.
	rows, err := s.DB.Query(fmt.Sprintf(
		`SELECT c.id, c.titulo, c.contenido, c.creado_en, c.imagen_s3_key, c.para_todos,
                COUNT(DISTINCT cl.padre_id),
                %s
         FROM circulares c
         LEFT JOIN circulares_lecturas cl ON cl.circular_id = c.id
         WHERE c.guarderia_id = $1
         GROUP BY c.id
         ORDER BY c.creado_en DESC
         LIMIT 50`,
		contarFamiliasDestino("c", "circulares_grupos", "circular_id", 1)),
		gID,
	)
	if err != nil {
		s.logError(c, "Error al consultar las circulares (staff)", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar las circulares"})
		return
	}
	defer rows.Close()

	circulares := []CircularConLecturas{}
	ids := []int{}
	for rows.Next() {
		var cir CircularConLecturas
		var imagenKey sql.NullString
		if err := rows.Scan(&cir.ID, &cir.Titulo, &cir.Contenido, &cir.CreadoEn, &imagenKey, &cir.ParaTodos, &cir.LeidoPor, &cir.TotalFamilias); err != nil {
			continue
		}
		s.firmarImagenCircular(&cir.Circular, imagenKey)
		cir.Grupos = []GrupoDestino{}
		circulares = append(circulares, cir)
		ids = append(ids, cir.ID)
	}

	// Los grupos se traen aparte, de un solo viaje: meterlos en el SELECT de
	// arriba obligaría a agregarlos por circular y ya hay dos subconsultas
	// de conteo ahí. Si esta parte falla, el listado se manda igual sin las
	// etiquetas de grupo en vez de dejar la pantalla vacía.
	if porCircular, err := s.gruposDePublicaciones("circulares_grupos", "circular_id", ids); err != nil {
		s.logError(c, "No se pudieron consultar los grupos de las circulares", err)
	} else {
		for i := range circulares {
			if grupos := porCircular[circulares[i].ID]; grupos != nil {
				circulares[i].Grupos = grupos
			}
		}
	}

	c.JSON(http.StatusOK, circulares)
}

// firmarImagenCircular centraliza lo que comparten handleListarCircularesStaff
// y handleListarCircularesPadre: si la circular trae imagen, firma su URL;
// si falla la firma, la circular se sigue mostrando (solo sin imagen) en vez
// de tumbar todo el listado por una key rota.
func (s *Server) firmarImagenCircular(cir *Circular, imagenKey sql.NullString) {
	if !imagenKey.Valid {
		return
	}
	if url, err := s.firmarURLFoto(imagenKey.String); err == nil {
		cir.ImagenURL = &url
	} else {
		s.logError(nil, "No se pudo firmar la imagen de la circular", err, "circular_id", cir.ID)
	}
}

// handleListarCircularesPadre es la vista simple que ya existía -- el padre
// NO se marca lector solo por listar (eso lo dispara el frontend, uno por
// uno, para los avisos que de verdad se le muestran en pantalla; ver
// handleMarcarCircularLeida).
func (s *Server) handleListarCircularesPadre(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")

	userID, _ := c.Get("user_id")

	rows, err := s.DB.Query(fmt.Sprintf(
		`SELECT c.id, c.titulo, c.contenido, c.creado_en, c.imagen_s3_key, c.para_todos
         FROM circulares c
         WHERE c.guarderia_id = $1 AND %s
         ORDER BY c.creado_en DESC
         LIMIT 50`,
		condicionVisibleParaPadre("c", "circulares_grupos", "circular_id", 2)),
		gID, userID,
	)
	if err != nil {
		s.logError(c, "Error al consultar las circulares (padre)", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar las circulares"})
		return
	}
	defer rows.Close()

	circulares := []Circular{}
	for rows.Next() {
		var cir Circular
		var imagenKey sql.NullString
		if err := rows.Scan(&cir.ID, &cir.Titulo, &cir.Contenido, &cir.CreadoEn, &imagenKey, &cir.ParaTodos); err != nil {
			continue
		}
		s.firmarImagenCircular(&cir, imagenKey)
		// El papá no necesita saber a qué grupos se dirigió: para él la
		// circular o le toca o no aparece.
		cir.Grupos = []GrupoDestino{}
		circulares = append(circulares, cir)
	}
	c.JSON(http.StatusOK, circulares)
}

// handleMarcarCircularLeida la llama el frontend del padre por cada aviso
// que realmente se muestra en su pantalla (no en el simple listado) -- así
// el conteo que ve staff refleja avisos vistos de verdad, no solo
// consultados por la API.
func (s *Server) handleMarcarCircularLeida(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	userID, _ := c.Get("user_id")
	circularID := c.Param("id")

	// Se comprueba la audiencia, no solo que la circular sea de su
	// guardería: si no, un papá podría marcar como leída una circular
	// dirigida a otro salón e inflar el "leído por" que ve el staff.
	var existe bool
	if err := s.DB.QueryRow(fmt.Sprintf(
		`SELECT EXISTS(SELECT 1 FROM circulares c WHERE c.id = $1 AND c.guarderia_id = $2 AND %s)`,
		condicionVisibleParaPadre("c", "circulares_grupos", "circular_id", 3)),
		circularID, gID, userID,
	).Scan(&existe); err != nil || !existe {
		c.JSON(http.StatusNotFound, gin.H{"error": "Circular no encontrada"})
		return
	}

	if _, err := s.DB.Exec(
		`INSERT INTO circulares_lecturas (circular_id, padre_id) VALUES ($1, $2) ON CONFLICT (circular_id, padre_id) DO NOTHING`,
		circularID, userID,
	); err != nil {
		s.logError(c, "No se pudo registrar la lectura de la circular", err, "circular_id", circularID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo registrar la lectura"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Marcada como leída"})
}

// handleDetalleLecturasCircular regresa la lista de familias que ya leyeron
// una circular (nombre + fecha), para el "quiénes" del staff.
func (s *Server) handleDetalleLecturasCircular(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	circularID := c.Param("id")

	var existe bool
	if err := s.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM circulares WHERE id = $1 AND guarderia_id = $2)`, circularID, gID).Scan(&existe); err != nil || !existe {
		c.JSON(http.StatusNotFound, gin.H{"error": "Circular no encontrada"})
		return
	}

	rows, err := s.DB.Query(
		`SELECT COALESCE(pa.nombre, 'Familia'), cl.leido_en
         FROM circulares_lecturas cl
         LEFT JOIN padres pa ON pa.id = cl.padre_id
         WHERE cl.circular_id = $1
         ORDER BY cl.leido_en DESC`,
		circularID,
	)
	if err != nil {
		s.logError(c, "Error al consultar las lecturas", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar las lecturas"})
		return
	}
	defer rows.Close()

	lecturas := []LecturaCircular{}
	for rows.Next() {
		var l LecturaCircular
		if err := rows.Scan(&l.Nombre, &l.LeidoEn); err != nil {
			continue
		}
		lecturas = append(lecturas, l)
	}
	c.JSON(http.StatusOK, lecturas)
}

// handleCrearCircular recibe multipart/form-data (antes era JSON) para
// poder traer una imagen opcional junto con título/contenido -- mismo
// patrón que leerMensajeConAdjunto en chat.go, mismo bucket privado y mismo
// límite de 10 MB que documentos_nino.
func (s *Server) handleCrearCircular(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	userID, _ := c.Get("user_id")

	titulo := strings.TrimSpace(c.PostForm("titulo"))
	contenido := strings.TrimSpace(c.PostForm("contenido"))
	if titulo == "" || contenido == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El título y el contenido son obligatorios"})
		return
	}

	var imagenKey *string
	if fileHeader, errArchivo := c.FormFile("imagen"); errArchivo == nil {
		if fileHeader.Size > maxTamanoDocumento {
			c.JSON(http.StatusBadRequest, gin.H{"error": "La imagen no puede pesar más de 10 MB"})
			return
		}
		contentType := fileHeader.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "image/") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "El archivo debe ser una imagen"})
			return
		}
		key := fmt.Sprintf("circulares/guarderia_%v/%d_%s", gID, time.Now().UnixNano(), fileHeader.Filename)
		if _, err := s.uploadToS3(fileHeader, key, contentType); err != nil {
			s.logError(c, "No se pudo subir la imagen de la circular a S3", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo subir la imagen"})
			return
		}
		imagenKey = &key
	}

	// A quién va dirigida. Sin grupos elegidos se comporta como siempre:
	// para todas las familias de la guardería.
	grupos, err := s.validarGruposDeGuarderia(c.PostFormArray("grupos"), gID)
	if err != nil {
		if imagenKey != nil {
			go s.borrarDeS3(*imagenKey)
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// La circular y sus grupos se guardan juntos o no se guarda ninguno: una
	// circular dirigida que quedara sin sus filas de grupo se le mostraría a
	// toda la guardería, justo lo contrario de lo que se pidió.
	tx, err := s.DB.Begin()
	if err != nil {
		if imagenKey != nil {
			go s.borrarDeS3(*imagenKey)
		}
		s.logError(c, "No se pudo publicar la circular", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo publicar la circular"})
		return
	}
	defer tx.Rollback()

	var nuevoID int
	if err := tx.QueryRow(
		`INSERT INTO circulares (guarderia_id, titulo, contenido, creado_por, imagen_s3_key, para_todos)
         VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		gID, titulo, contenido, userID, imagenKey, len(grupos) == 0,
	).Scan(&nuevoID); err != nil {
		if imagenKey != nil {
			go s.borrarDeS3(*imagenKey) // la circular no se guardó, no dejamos la imagen huérfana
		}
		s.logError(c, "No se pudo publicar la circular", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo publicar la circular"})
		return
	}

	if err := guardarGruposDestino(tx, "circulares_grupos", "circular_id", nuevoID, grupos); err != nil {
		if imagenKey != nil {
			go s.borrarDeS3(*imagenKey)
		}
		s.logError(c, "No se pudieron guardar los grupos de la circular", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo publicar la circular"})
		return
	}

	if err := tx.Commit(); err != nil {
		if imagenKey != nil {
			go s.borrarDeS3(*imagenKey)
		}
		s.logError(c, "No se pudo publicar la circular", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo publicar la circular"})
		return
	}

	go s.notificarCircular(gID, titulo, contenido, grupos)

	c.JSON(http.StatusCreated, gin.H{"id": nuevoID, "message": "Circular publicada"})
}

func (s *Server) handleEliminarCircular(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	circularID := c.Param("id")

	var imagenKey sql.NullString
	if err := s.DB.QueryRow(`SELECT imagen_s3_key FROM circulares WHERE id = $1 AND guarderia_id = $2`, circularID, gID).Scan(&imagenKey); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Circular no encontrada"})
		return
	}

	res, err := s.DB.Exec(`DELETE FROM circulares WHERE id = $1 AND guarderia_id = $2`, circularID, gID)
	if err != nil {
		s.logError(c, "No se pudo eliminar la circular", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo eliminar la circular"})
		return
	}
	if filas, _ := res.RowsAffected(); filas == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Circular no encontrada"})
		return
	}
	if imagenKey.Valid {
		go s.borrarDeS3(imagenKey.String)
	}
	c.JSON(http.StatusOK, gin.H{"message": "Circular eliminada"})
}
