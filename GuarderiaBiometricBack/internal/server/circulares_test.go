package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestListarCirculares(t *testing.T) {
	srv, mock := nuevoServidorDePruebaConDB(t)
	mock.ExpectQuery("SELECT id, titulo, contenido, creado_en FROM circulares").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "titulo", "contenido", "creado_en"}).
			AddRow(1, "Suspensión de clases", "El viernes no hay clases por junta de consejo técnico.", "2026-08-10T12:00:00Z"))

	r := nuevoRouterDePrueba(srv)
	req := jsonRequest(http.MethodGet, "/circulares", nil)
	autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
	}
	var circulares []Circular
	if err := json.Unmarshal(w.Body.Bytes(), &circulares); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v", err)
	}
	if len(circulares) != 1 || circulares[0].Titulo != "Suspensión de clases" {
		t.Fatalf("se esperaba una circular con ese título, se recibió: %+v", circulares)
	}
}

func TestListarCircularesComoPadre(t *testing.T) {
	srv, mock := nuevoServidorDePruebaConDB(t)
	mock.ExpectQuery("SELECT id, titulo, contenido, creado_en FROM circulares").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "titulo", "contenido", "creado_en"}))

	r := nuevoRouterDePrueba(srv)
	req := jsonRequest(http.MethodGet, "/padre/circulares", nil)
	autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
	}
}

func TestCrearCircular(t *testing.T) {
	t.Run("título vacío -> 400", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/circulares", map[string]string{"titulo": "   ", "contenido": "algo"})
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("papá no puede publicar -> 403", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/circulares", map[string]string{"titulo": "Aviso", "contenido": "Texto"})
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("staff publica -> 201", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("INSERT INTO circulares").
			WithArgs(1, "Aviso importante", "Mañana hay junta de padres a las 5pm.", 1).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/circulares", map[string]string{
			"titulo": "Aviso importante", "contenido": "Mañana hay junta de padres a las 5pm.",
		})
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("código = %d; esperado 201 (body: %s)", w.Code, w.Body.String())
		}
	})
}

func TestEliminarCircular(t *testing.T) {
	t.Run("no encontrada -> 404", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectExec("DELETE FROM circulares").
			WithArgs("99", 1).
			WillReturnResult(sqlmock.NewResult(0, 0))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodDelete, "/circulares/99", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("código = %d; esperado 404 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("staff elimina -> 200", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectExec("DELETE FROM circulares").
			WithArgs("7", 1).
			WillReturnResult(sqlmock.NewResult(0, 1))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodDelete, "/circulares/7", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})
}
