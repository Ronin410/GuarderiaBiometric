package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestListarCalendario(t *testing.T) {
	srv, mock := nuevoServidorDePruebaConDB(t)
	mock.ExpectQuery("SELECT id, titulo, COALESCE\\(descripcion, ''\\), fecha_inicio, fecha_fin, tipo, creado_en FROM eventos_calendario").
		WithArgs(1, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "titulo", "descripcion", "fecha_inicio", "fecha_fin", "tipo", "creado_en"}).
			AddRow(1, "Suspensión de clases", "Junta de consejo técnico", "2026-09-01T00:00:00Z", nil, "suspension", "2026-08-13T10:00:00Z"))

	r := nuevoRouterDePrueba(srv)
	req := jsonRequest(http.MethodGet, "/calendario", nil)
	autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
	}
	var eventos []EventoCalendario
	if err := json.Unmarshal(w.Body.Bytes(), &eventos); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v", err)
	}
	if len(eventos) != 1 || eventos[0].FechaInicio != "2026-09-01" || eventos[0].FechaFin != nil {
		t.Fatalf("se esperaba un evento sin fecha_fin recortado a la fecha, se recibió: %+v", eventos)
	}
}

func TestListarCalendarioComoPadre(t *testing.T) {
	srv, mock := nuevoServidorDePruebaConDB(t)
	mock.ExpectQuery("SELECT id, titulo, COALESCE\\(descripcion, ''\\), fecha_inicio, fecha_fin, tipo, creado_en FROM eventos_calendario").
		WithArgs(1, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "titulo", "descripcion", "fecha_inicio", "fecha_fin", "tipo", "creado_en"}))

	r := nuevoRouterDePrueba(srv)
	req := jsonRequest(http.MethodGet, "/padre/calendario", nil)
	autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
	}
}

func TestCrearEventoCalendario(t *testing.T) {
	t.Run("título vacío -> 400", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/calendario", map[string]string{"titulo": "   ", "fecha_inicio": "2026-09-01"})
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("fecha final antes de la inicial -> 400", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/calendario", map[string]string{
			"titulo": "Vacaciones de invierno", "fecha_inicio": "2026-12-20", "fecha_fin": "2026-12-10",
		})
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("tipo inválido -> 400", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/calendario", map[string]string{
			"titulo": "Algo", "fecha_inicio": "2026-09-01", "tipo": "fiesta",
		})
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("papá no puede crear -> 403", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/calendario", map[string]string{"titulo": "Aviso", "fecha_inicio": "2026-09-01"})
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("staff crea un evento -> 201", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("INSERT INTO eventos_calendario").
			WithArgs(1, "Junta de padres", "Auditorio principal", "2026-09-05", nil, "junta", 1).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(9))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/calendario", map[string]string{
			"titulo": "Junta de padres", "descripcion": "Auditorio principal",
			"fecha_inicio": "2026-09-05", "tipo": "junta",
		})
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("código = %d; esperado 201 (body: %s)", w.Code, w.Body.String())
		}
	})
}

func TestEliminarEventoCalendario(t *testing.T) {
	t.Run("no encontrado -> 404", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectExec("DELETE FROM eventos_calendario").
			WithArgs("99", 1).
			WillReturnResult(sqlmock.NewResult(0, 0))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodDelete, "/calendario/99", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("código = %d; esperado 404 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("staff elimina -> 200", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectExec("DELETE FROM eventos_calendario").
			WithArgs("9", 1).
			WillReturnResult(sqlmock.NewResult(0, 1))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodDelete, "/calendario/9", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})
}
