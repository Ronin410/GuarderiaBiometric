package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"biometrico/internal/middleware"
)

// tiposPreguntaValidos son los tipos que acepta encuesta_preguntas.tipo
// (mismo juego que el CHECK de la migración).
var tiposPreguntaValidos = map[string]bool{
	"opcion_multiple": true,
	"texto_libre":     true,
}

// Encuesta es un cuestionario que staff publica para todos los padres de la
// guardería -- "Encuestas para familias" del PDF de referencia.
type Encuesta struct {
	ID          int    `json:"id"`
	Titulo      string `json:"titulo"`
	Descripcion string `json:"descripcion"`
	Activa      bool   `json:"activa"`
	CreadoEn    string `json:"creado_en"`
}

// EncuestaConConteo es lo que ve staff en el listado: la encuesta más
// cuántas familias han participado (respondieron al menos una pregunta),
// sobre el total de familias de la guardería.
type EncuestaConConteo struct {
	Encuesta
	TotalRespuestas int `json:"total_respuestas"`
	TotalFamilias   int `json:"total_familias"`
}

// PreguntaEncuesta es una pregunta tal como la ve el padre para responder:
// sin resultados agregados.
type PreguntaEncuesta struct {
	ID       int      `json:"id"`
	Texto    string   `json:"texto"`
	Tipo     string   `json:"tipo"`
	Opciones []string `json:"opciones,omitempty"`
	// RespuestaPadre solo se llena en el listado del padre cuando ya
	// respondió esa encuesta -- así el frontend puede mostrar el
	// formulario deshabilitado con lo que él mismo puso, en vez de solo
	// un aviso de "ya respondiste". Va vacío (y se omite) en el detalle
	// de staff, que no llama a esa parte del código.
	RespuestaPadre string `json:"respuesta_padre,omitempty"`
}

// PreguntaConResultados es lo que ve staff en el detalle: la pregunta más
// el conteo por opción (opcion_multiple) o la lista de respuestas
// (texto_libre).
type PreguntaConResultados struct {
	PreguntaEncuesta
	ConteoOpciones  map[string]int `json:"conteo_opciones,omitempty"`
	RespuestasTexto []string       `json:"respuestas_texto,omitempty"`
}

// EncuestaDetalleStaff es la encuesta completa con resultados, para el
// detalle que consulta staff.
type EncuestaDetalleStaff struct {
	Encuesta
	Preguntas []PreguntaConResultados `json:"preguntas"`
}

// EncuestaParaPadre es la encuesta tal como la ve el padre: sin resultados
// ajenos, con sus propias preguntas para responder y si ya participó.
type EncuestaParaPadre struct {
	Encuesta
	Preguntas    []PreguntaEncuesta `json:"preguntas"`
	YaRespondida bool               `json:"ya_respondida"`
}

func (s *Server) registrarRutasEncuestas(r *gin.Engine) {
	auth := middleware.Auth(s.JWTKey)
	staff := middleware.RequireStaff()

	r.GET("/encuestas", auth, staff, s.handleListarEncuestasStaff)
	r.POST("/encuestas", auth, staff, s.handleCrearEncuesta)
	r.GET("/encuestas/:id", auth, staff, s.handleDetalleEncuesta)
	r.PUT("/encuestas/:id/cerrar", auth, staff, s.handleCerrarEncuesta)
	r.DELETE("/encuestas/:id", auth, staff, s.handleEliminarEncuesta)

	r.GET("/padre/encuestas", auth, s.handleListarEncuestasPadre)
	r.POST("/padre/encuestas/:id/respuestas", auth, s.handleResponderEncuesta)
}

// handleListarEncuestasStaff cuenta "cuántas familias han participado" como
// el número de padres distintos que respondieron AL MENOS una pregunta de
// la encuesta -- una aproximación práctica a "completaron la encuesta",
// suficiente para el propósito de ver a simple vista qué tanta respuesta
// tuvo cada una, sin la complejidad de exigir que hayan respondido todas.
func (s *Server) handleListarEncuestasStaff(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")

	rows, err := s.DB.Query(
		`SELECT e.id, e.titulo, COALESCE(e.descripcion, ''), e.activa, e.creado_en,
                COUNT(DISTINCT er.padre_id),
                (SELECT COUNT(*) FROM padres p WHERE p.guarderia_id = $1)
         FROM encuestas e
         LEFT JOIN encuesta_preguntas ep ON ep.encuesta_id = e.id
         LEFT JOIN encuesta_respuestas er ON er.pregunta_id = ep.id
         WHERE e.guarderia_id = $1
         GROUP BY e.id
         ORDER BY e.creado_en DESC`,
		gID,
	)
	if err != nil {
		s.logError(c, "Error al consultar las encuestas (staff)", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar las encuestas"})
		return
	}
	defer rows.Close()

	encuestas := []EncuestaConConteo{}
	for rows.Next() {
		var enc EncuestaConConteo
		if err := rows.Scan(&enc.ID, &enc.Titulo, &enc.Descripcion, &enc.Activa, &enc.CreadoEn, &enc.TotalRespuestas, &enc.TotalFamilias); err != nil {
			continue
		}
		encuestas = append(encuestas, enc)
	}
	c.JSON(http.StatusOK, encuestas)
}

func (s *Server) handleCrearEncuesta(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	userID, _ := c.Get("user_id")

	var input struct {
		Titulo      string `json:"titulo"`
		Descripcion string `json:"descripcion"`
		Preguntas   []struct {
			Texto    string   `json:"texto"`
			Tipo     string   `json:"tipo"`
			Opciones []string `json:"opciones"`
		} `json:"preguntas"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	titulo := strings.TrimSpace(input.Titulo)
	if titulo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El título es obligatorio"})
		return
	}
	if len(input.Preguntas) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Agrega al menos una pregunta"})
		return
	}
	for i, p := range input.Preguntas {
		if strings.TrimSpace(p.Texto) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("La pregunta %d no puede estar vacía", i+1)})
			return
		}
		if !tiposPreguntaValidos[p.Tipo] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tipo de pregunta inválido"})
			return
		}
		if p.Tipo == "opcion_multiple" && len(p.Opciones) < 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("La pregunta %d necesita al menos 2 opciones", i+1)})
			return
		}
	}

	var encuestaID int
	if err := s.DB.QueryRow(
		`INSERT INTO encuestas (guarderia_id, titulo, descripcion, creado_por) VALUES ($1, $2, $3, $4) RETURNING id`,
		gID, titulo, strings.TrimSpace(input.Descripcion), userID,
	).Scan(&encuestaID); err != nil {
		s.logError(c, "No se pudo crear la encuesta", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo crear la encuesta"})
		return
	}

	for i, p := range input.Preguntas {
		var opcionesJSON *string
		if p.Tipo == "opcion_multiple" {
			b, err := json.Marshal(p.Opciones)
			if err != nil {
				s.logError(c, "No se pudieron guardar las opciones de una pregunta", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudieron guardar las opciones de una pregunta"})
				return
			}
			texto := string(b)
			opcionesJSON = &texto
		}
		if _, err := s.DB.Exec(
			`INSERT INTO encuesta_preguntas (encuesta_id, texto, tipo, opciones, orden) VALUES ($1, $2, $3, $4, $5)`,
			encuestaID, strings.TrimSpace(p.Texto), p.Tipo, opcionesJSON, i,
		); err != nil {
			s.logError(c, "No se pudo guardar una pregunta de la encuesta", err, "pregunta_num", i+1, "encuesta_id", encuestaID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo guardar una de las preguntas"})
			return
		}
	}

	go s.notificarCircular(gID, "Nueva encuesta: "+titulo, "Tu opinión nos importa -- respóndela desde tu portal de familia.")

	c.JSON(http.StatusCreated, gin.H{"id": encuestaID, "message": "Encuesta publicada"})
}

func (s *Server) handleDetalleEncuesta(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	encuestaID := c.Param("id")

	enc, err := s.obtenerEncuesta(encuestaID, gID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Encuesta no encontrada"})
		return
	} else if err != nil {
		s.logError(c, "Error al consultar la encuesta", err, "encuesta_id", encuestaID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar la encuesta"})
		return
	}

	preguntas, err := s.obtenerPreguntas(encuestaID)
	if err != nil {
		s.logError(c, "Error al consultar las preguntas", err, "encuesta_id", encuestaID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar las preguntas"})
		return
	}

	conResultados := make([]PreguntaConResultados, len(preguntas))
	for i, p := range preguntas {
		conResultados[i].PreguntaEncuesta = p
		if p.Tipo == "opcion_multiple" {
			conteo := map[string]int{}
			filas, err := s.DB.Query(`SELECT respuesta, COUNT(*) FROM encuesta_respuestas WHERE pregunta_id = $1 GROUP BY respuesta`, p.ID)
			if err == nil {
				for filas.Next() {
					var opcion string
					var n int
					if err := filas.Scan(&opcion, &n); err == nil {
						conteo[opcion] = n
					}
				}
				filas.Close()
			}
			conResultados[i].ConteoOpciones = conteo
		} else {
			var textos []string
			filas, err := s.DB.Query(`SELECT respuesta FROM encuesta_respuestas WHERE pregunta_id = $1 ORDER BY creado_en DESC`, p.ID)
			if err == nil {
				for filas.Next() {
					var t string
					if err := filas.Scan(&t); err == nil {
						textos = append(textos, t)
					}
				}
				filas.Close()
			}
			conResultados[i].RespuestasTexto = textos
		}
	}

	c.JSON(http.StatusOK, EncuestaDetalleStaff{Encuesta: enc, Preguntas: conResultados})
}

func (s *Server) handleCerrarEncuesta(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	encuestaID := c.Param("id")

	res, err := s.DB.Exec(`UPDATE encuestas SET activa = false WHERE id = $1 AND guarderia_id = $2`, encuestaID, gID)
	if err != nil {
		s.logError(c, "No se pudo cerrar la encuesta", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo cerrar la encuesta"})
		return
	}
	if filas, _ := res.RowsAffected(); filas == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Encuesta no encontrada"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Encuesta cerrada"})
}

func (s *Server) handleEliminarEncuesta(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	encuestaID := c.Param("id")

	res, err := s.DB.Exec(`DELETE FROM encuestas WHERE id = $1 AND guarderia_id = $2`, encuestaID, gID)
	if err != nil {
		s.logError(c, "No se pudo eliminar la encuesta", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo eliminar la encuesta"})
		return
	}
	if filas, _ := res.RowsAffected(); filas == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Encuesta no encontrada"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Encuesta eliminada"})
}

// handleListarEncuestasPadre regresa las encuestas activas con sus
// preguntas (para responder) y si el padre ya participó en cada una.
func (s *Server) handleListarEncuestasPadre(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	userID, _ := c.Get("user_id")

	rows, err := s.DB.Query(
		`SELECT id, titulo, COALESCE(descripcion, ''), activa, creado_en
         FROM encuestas WHERE guarderia_id = $1 AND activa = true
         ORDER BY creado_en DESC`,
		gID,
	)
	if err != nil {
		s.logError(c, "Error al consultar las encuestas (padre)", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar las encuestas"})
		return
	}
	encuestas := []EncuestaParaPadre{}
	for rows.Next() {
		var enc Encuesta
		if err := rows.Scan(&enc.ID, &enc.Titulo, &enc.Descripcion, &enc.Activa, &enc.CreadoEn); err != nil {
			continue
		}
		encuestas = append(encuestas, EncuestaParaPadre{Encuesta: enc})
	}
	rows.Close()

	for i := range encuestas {
		preguntas, err := s.obtenerPreguntas(fmt.Sprint(encuestas[i].ID))
		if err != nil {
			continue
		}
		encuestas[i].Preguntas = preguntas

		var totalPreguntas, respondidas int
		s.DB.QueryRow(`SELECT COUNT(*) FROM encuesta_preguntas WHERE encuesta_id = $1`, encuestas[i].ID).Scan(&totalPreguntas)
		s.DB.QueryRow(
			`SELECT COUNT(DISTINCT er.pregunta_id) FROM encuesta_respuestas er
             JOIN encuesta_preguntas ep ON ep.id = er.pregunta_id
             WHERE ep.encuesta_id = $1 AND er.padre_id = $2`,
			encuestas[i].ID, userID,
		).Scan(&respondidas)
		encuestas[i].YaRespondida = totalPreguntas > 0 && respondidas >= totalPreguntas

		// Si ya la respondió, se le manda de vuelta lo que él mismo puso
		// en cada pregunta, para que el frontend pinte el formulario
		// deshabilitado con esa información cargada (en vez de un
		// formulario vacío o solo un aviso).
		if encuestas[i].YaRespondida {
			filas, err := s.DB.Query(
				`SELECT er.pregunta_id, er.respuesta FROM encuesta_respuestas er
                 JOIN encuesta_preguntas ep ON ep.id = er.pregunta_id
                 WHERE ep.encuesta_id = $1 AND er.padre_id = $2`,
				encuestas[i].ID, userID,
			)
			if err == nil {
				propias := map[int]string{}
				for filas.Next() {
					var preguntaID int
					var respuesta string
					if err := filas.Scan(&preguntaID, &respuesta); err == nil {
						propias[preguntaID] = respuesta
					}
				}
				filas.Close()
				for j := range encuestas[i].Preguntas {
					encuestas[i].Preguntas[j].RespuestaPadre = propias[encuestas[i].Preguntas[j].ID]
				}
			}
		}
	}

	c.JSON(http.StatusOK, encuestas)
}

func (s *Server) handleResponderEncuesta(c *gin.Context) {
	gID, _ := c.Get("guarderia_id")
	userID, _ := c.Get("user_id")
	encuestaID := c.Param("id")

	var activa bool
	err := s.DB.QueryRow(`SELECT activa FROM encuestas WHERE id = $1 AND guarderia_id = $2`, encuestaID, gID).Scan(&activa)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Encuesta no encontrada"})
		return
	} else if err != nil {
		s.logError(c, "Error al consultar la encuesta (responder)", err, "encuesta_id", encuestaID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al consultar la encuesta"})
		return
	}
	if !activa {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Esta encuesta ya está cerrada"})
		return
	}

	// Solo se puede responder una vez: si ya contestó todas las preguntas
	// de esta encuesta, se rechaza el reenvío en vez de sobrescribir sus
	// respuestas (el ON CONFLICT de abajo actualizaría en silencio si no
	// se frenara aquí).
	var totalPreguntas, respondidas int
	s.DB.QueryRow(`SELECT COUNT(*) FROM encuesta_preguntas WHERE encuesta_id = $1`, encuestaID).Scan(&totalPreguntas)
	s.DB.QueryRow(
		`SELECT COUNT(DISTINCT er.pregunta_id) FROM encuesta_respuestas er
         JOIN encuesta_preguntas ep ON ep.id = er.pregunta_id
         WHERE ep.encuesta_id = $1 AND er.padre_id = $2`,
		encuestaID, userID,
	).Scan(&respondidas)
	if totalPreguntas > 0 && respondidas >= totalPreguntas {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ya respondiste esta encuesta, no se puede modificar"})
		return
	}

	var input struct {
		Respuestas []struct {
			PreguntaID int    `json:"pregunta_id"`
			Respuesta  string `json:"respuesta"`
		} `json:"respuestas"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}
	if len(input.Respuestas) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Responde al menos una pregunta"})
		return
	}

	for _, r := range input.Respuestas {
		respuesta := strings.TrimSpace(r.Respuesta)
		if respuesta == "" {
			continue
		}
		// La pregunta debe pertenecer a ESTA encuesta -- evita que alguien
		// mande un pregunta_id de otra encuesta (o de otra guardería) junto
		// con el id de esta.
		var perteneceAEncuesta bool
		if err := s.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM encuesta_preguntas WHERE id = $1 AND encuesta_id = $2)`, r.PreguntaID, encuestaID).Scan(&perteneceAEncuesta); err != nil || !perteneceAEncuesta {
			continue
		}
		s.DB.Exec(
			`INSERT INTO encuesta_respuestas (pregunta_id, padre_id, respuesta) VALUES ($1, $2, $3)
             ON CONFLICT (pregunta_id, padre_id) DO UPDATE SET respuesta = EXCLUDED.respuesta, creado_en = CURRENT_TIMESTAMP`,
			r.PreguntaID, userID, respuesta,
		)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Respuestas guardadas"})
}

// obtenerEncuesta trae los datos base de una encuesta, verificando que
// pertenezca a la guardería que hace la petición.
func (s *Server) obtenerEncuesta(encuestaID string, gID any) (Encuesta, error) {
	var enc Encuesta
	err := s.DB.QueryRow(
		`SELECT id, titulo, COALESCE(descripcion, ''), activa, creado_en FROM encuestas WHERE id = $1 AND guarderia_id = $2`,
		encuestaID, gID,
	).Scan(&enc.ID, &enc.Titulo, &enc.Descripcion, &enc.Activa, &enc.CreadoEn)
	return enc, err
}

// obtenerPreguntas trae las preguntas de una encuesta, desmarshaleando el
// JSON de opciones -- compartido entre el detalle de staff y el listado
// del padre.
func (s *Server) obtenerPreguntas(encuestaID string) ([]PreguntaEncuesta, error) {
	rows, err := s.DB.Query(
		`SELECT id, texto, tipo, opciones FROM encuesta_preguntas WHERE encuesta_id = $1 ORDER BY orden ASC`,
		encuestaID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	preguntas := []PreguntaEncuesta{}
	for rows.Next() {
		var p PreguntaEncuesta
		var opcionesRaw *string
		if err := rows.Scan(&p.ID, &p.Texto, &p.Tipo, &opcionesRaw); err != nil {
			continue
		}
		if opcionesRaw != nil {
			json.Unmarshal([]byte(*opcionesRaw), &p.Opciones)
		}
		preguntas = append(preguntas, p)
	}
	return preguntas, nil
}
