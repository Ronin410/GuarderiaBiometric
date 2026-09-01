package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestListarEncuestasStaff(t *testing.T) {
	srv, mock := nuevoServidorDePruebaConDB(t)
	mock.ExpectQuery("SELECT e.id, e.titulo, COALESCE\\(e.descripcion, ''\\), e.activa, e.creado_en").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "titulo", "descripcion", "activa", "creado_en", "total_respuestas", "total_familias"}).
			AddRow(1, "Posada navideña", "¿Asistirán?", true, "2026-08-13T10:00:00Z", 3, 8))

	r := nuevoRouterDePrueba(srv)
	req := jsonRequest(http.MethodGet, "/encuestas", nil)
	autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
	}
	var encuestas []EncuestaConConteo
	if err := json.Unmarshal(w.Body.Bytes(), &encuestas); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v", err)
	}
	if len(encuestas) != 1 || encuestas[0].TotalRespuestas != 3 || encuestas[0].TotalFamilias != 8 {
		t.Fatalf("se esperaba 3 de 8, se recibió: %+v", encuestas)
	}
}

func TestCrearEncuesta(t *testing.T) {
	t.Run("papá no puede crear -> 403", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/encuestas", map[string]any{
			"titulo": "Aviso", "preguntas": []map[string]string{{"texto": "¿Vienen?", "tipo": "texto_libre"}},
		})
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("sin preguntas -> 400", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/encuestas", map[string]any{"titulo": "Posada", "preguntas": []any{}})
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("opción múltiple con menos de 2 opciones -> 400", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/encuestas", map[string]any{
			"titulo": "Posada",
			"preguntas": []map[string]any{
				{"texto": "¿Vienen?", "tipo": "opcion_multiple", "opciones": []string{"Sí"}},
			},
		})
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("staff crea con una pregunta de opción múltiple y otra de texto libre -> 201", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("INSERT INTO encuestas").
			WithArgs(1, "Posada navideña", "Confírmanos", 1).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(9))
		mock.ExpectExec("INSERT INTO encuesta_preguntas").
			WithArgs(9, "¿Asistirán?", "opcion_multiple", sqlmock.AnyArg(), 0).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("INSERT INTO encuesta_preguntas").
			WithArgs(9, "Comentarios", "texto_libre", nil, 1).
			WillReturnResult(sqlmock.NewResult(2, 1))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/encuestas", map[string]any{
			"titulo": "Posada navideña", "descripcion": "Confírmanos",
			"preguntas": []map[string]any{
				{"texto": "¿Asistirán?", "tipo": "opcion_multiple", "opciones": []string{"Sí", "No"}},
				{"texto": "Comentarios", "tipo": "texto_libre"},
			},
		})
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("código = %d; esperado 201 (body: %s)", w.Code, w.Body.String())
		}
	})
}

func TestDetalleEncuesta(t *testing.T) {
	t.Run("no encontrada -> 404", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT id, titulo, COALESCE\\(descripcion, ''\\), activa, creado_en FROM encuestas").
			WithArgs("99", 1).
			WillReturnError(sql.ErrNoRows)

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/encuestas/99", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("código = %d; esperado 404 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("staff ve el detalle con resultados agregados -> 200", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT id, titulo, COALESCE\\(descripcion, ''\\), activa, creado_en FROM encuestas").
			WithArgs("1", 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "titulo", "descripcion", "activa", "creado_en"}).
				AddRow(1, "Posada navideña", "Confírmanos", true, "2026-08-13T10:00:00Z"))
		mock.ExpectQuery("SELECT id, texto, tipo, opciones FROM encuesta_preguntas").
			WithArgs("1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "texto", "tipo", "opciones"}).
				AddRow(1, "¿Asistirán?", "opcion_multiple", `["Sí","No"]`))
		mock.ExpectQuery("SELECT respuesta, COUNT\\(\\*\\) FROM encuesta_respuestas").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"respuesta", "count"}).AddRow("Sí", 5).AddRow("No", 2))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/encuestas/1", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		var detalle EncuestaDetalleStaff
		if err := json.Unmarshal(w.Body.Bytes(), &detalle); err != nil {
			t.Fatalf("respuesta no es JSON válido: %v", err)
		}
		if len(detalle.Preguntas) != 1 || detalle.Preguntas[0].ConteoOpciones["Sí"] != 5 || detalle.Preguntas[0].ConteoOpciones["No"] != 2 {
			t.Fatalf("se esperaba Sí=5 No=2, se recibió: %+v", detalle.Preguntas)
		}
	})
}

func TestListarEncuestasComoPadre(t *testing.T) {
	srv, mock := nuevoServidorDePruebaConDB(t)
	mock.ExpectQuery("SELECT id, titulo, COALESCE\\(descripcion, ''\\), activa, creado_en FROM encuestas").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "titulo", "descripcion", "activa", "creado_en"}).
			AddRow(1, "Posada navideña", "Confírmanos", true, "2026-08-13T10:00:00Z"))
	mock.ExpectQuery("SELECT id, texto, tipo, opciones FROM encuesta_preguntas").
		WithArgs("1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "texto", "tipo", "opciones"}).
			AddRow(1, "¿Asistirán?", "opcion_multiple", `["Sí","No"]`))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM encuesta_preguntas").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("SELECT COUNT\\(DISTINCT er.pregunta_id\\)").
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	r := nuevoRouterDePrueba(srv)
	req := jsonRequest(http.MethodGet, "/padre/encuestas", nil)
	autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
	}
	var encuestas []EncuestaParaPadre
	if err := json.Unmarshal(w.Body.Bytes(), &encuestas); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v", err)
	}
	if len(encuestas) != 1 || encuestas[0].YaRespondida || len(encuestas[0].Preguntas) != 1 {
		t.Fatalf("se esperaba 1 encuesta sin responder con 1 pregunta, se recibió: %+v", encuestas)
	}
}

// Ya respondida: el listado debe traer de vuelta lo que el papá puso, para
// que el frontend pueda pintar el formulario deshabilitado con esa
// información cargada (en vez de solo un aviso genérico).
func TestListarEncuestasComoPadreYaRespondida(t *testing.T) {
	srv, mock := nuevoServidorDePruebaConDB(t)
	mock.ExpectQuery("SELECT id, titulo, COALESCE\\(descripcion, ''\\), activa, creado_en FROM encuestas").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "titulo", "descripcion", "activa", "creado_en"}).
			AddRow(1, "Posada navideña", "Confírmanos", true, "2026-08-13T10:00:00Z"))
	mock.ExpectQuery("SELECT id, texto, tipo, opciones FROM encuesta_preguntas").
		WithArgs("1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "texto", "tipo", "opciones"}).
			AddRow(1, "¿Asistirán?", "opcion_multiple", `["Sí","No"]`))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM encuesta_preguntas").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("SELECT COUNT\\(DISTINCT er.pregunta_id\\)").
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("SELECT er.pregunta_id, er.respuesta FROM encuesta_respuestas").
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"pregunta_id", "respuesta"}).AddRow(1, "Sí"))

	r := nuevoRouterDePrueba(srv)
	req := jsonRequest(http.MethodGet, "/padre/encuestas", nil)
	autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
	}
	var encuestas []EncuestaParaPadre
	if err := json.Unmarshal(w.Body.Bytes(), &encuestas); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v", err)
	}
	if len(encuestas) != 1 || !encuestas[0].YaRespondida || len(encuestas[0].Preguntas) != 1 || encuestas[0].Preguntas[0].RespuestaPadre != "Sí" {
		t.Fatalf("se esperaba 1 encuesta respondida con respuesta_padre = \"Sí\", se recibió: %+v", encuestas)
	}
}

func TestResponderEncuesta(t *testing.T) {
	t.Run("encuesta cerrada -> 400", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT activa FROM encuestas").
			WithArgs("1", 1).
			WillReturnRows(sqlmock.NewRows([]string{"activa"}).AddRow(false))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/padre/encuestas/1/respuestas", map[string]any{
			"respuestas": []map[string]any{{"pregunta_id": 1, "respuesta": "Sí"}},
		})
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("papá responde -> 200", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT activa FROM encuestas").
			WithArgs("1", 1).
			WillReturnRows(sqlmock.NewRows([]string{"activa"}).AddRow(true))
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM encuesta_preguntas").
			WithArgs("1").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery("SELECT COUNT\\(DISTINCT er.pregunta_id\\)").
			WithArgs("1", 1).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery("SELECT EXISTS").WithArgs(1, "1").
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectExec("INSERT INTO encuesta_respuestas").
			WithArgs(1, 1, "Sí").
			WillReturnResult(sqlmock.NewResult(1, 1))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/padre/encuestas/1/respuestas", map[string]any{
			"respuestas": []map[string]any{{"pregunta_id": 1, "respuesta": "Sí"}},
		})
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("ya respondió todas las preguntas -> 400 (no se puede reenviar)", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT activa FROM encuestas").
			WithArgs("1", 1).
			WillReturnRows(sqlmock.NewRows([]string{"activa"}).AddRow(true))
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM encuesta_preguntas").
			WithArgs("1").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery("SELECT COUNT\\(DISTINCT er.pregunta_id\\)").
			WithArgs("1", 1).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/padre/encuestas/1/respuestas", map[string]any{
			"respuestas": []map[string]any{{"pregunta_id": 1, "respuesta": "No"}},
		})
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})
}

func TestCerrarEncuesta(t *testing.T) {
	srv, mock := nuevoServidorDePruebaConDB(t)
	mock.ExpectExec("UPDATE encuestas SET activa = false").
		WithArgs("1", 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := nuevoRouterDePrueba(srv)
	req := jsonRequest(http.MethodPut, "/encuestas/1/cerrar", nil)
	autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
	}
}

func TestEliminarEncuesta(t *testing.T) {
	srv, mock := nuevoServidorDePruebaConDB(t)
	mock.ExpectExec("DELETE FROM encuestas").
		WithArgs("1", 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := nuevoRouterDePrueba(srv)
	req := jsonRequest(http.MethodDelete, "/encuestas/1", nil)
	autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
	}
}
