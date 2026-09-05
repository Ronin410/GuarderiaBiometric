package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func nuevoServidorDePruebaConDB(t *testing.T) (*Server, sqlmock.Sqlmock) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { mockDB.Close() })

	srv := New()
	srv.DB = mockDB
	srv.JWTKey = []byte("clave-de-prueba-solo-para-tests")
	return srv, mock
}

func TestListarGrupos(t *testing.T) {
	srv, mock := nuevoServidorDePruebaConDB(t)
	mock.ExpectQuery("SELECT(.|\n)*FROM grupos(.|\n)*WHERE g.guarderia_id = \\$1").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "nombre", "count"}).
			AddRow(1, "Sala Maternal", 3).
			AddRow(2, "Preescolar A", 0))

	r := nuevoRouterDePrueba(srv)
	req := jsonRequest(http.MethodGet, "/admin/grupos", nil)
	autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
	}
}

func TestCrearGrupo(t *testing.T) {
	t.Run("nombre vacío -> 400", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/admin/grupos", map[string]string{"nombre": "   "})
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("papá no puede crear grupos -> 403", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/admin/grupos", map[string]string{"nombre": "Sala Maternal"})
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("staff crea grupo -> 201", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("INSERT INTO grupos").
			WithArgs(1, "Sala Maternal").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/admin/grupos", map[string]string{"nombre": "Sala Maternal"})
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("código = %d; esperado 201 (body: %s)", w.Code, w.Body.String())
		}
	})
}

func TestEliminarGrupoConNinos(t *testing.T) {
	srv, mock := nuevoServidorDePruebaConDB(t)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM hijos WHERE grupo_id = \\$1 AND activo").
		WithArgs("1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	r := nuevoRouterDePrueba(srv)
	req := jsonRequest(http.MethodDelete, "/admin/grupos/1", nil)
	autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("código = %d; esperado 409 al borrar un grupo con niños asignados (body: %s)", w.Code, w.Body.String())
	}
}

func TestEliminarGrupoVacio(t *testing.T) {
	srv, mock := nuevoServidorDePruebaConDB(t)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM hijos WHERE grupo_id = \\$1 AND activo").
		WithArgs("1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectExec("DELETE FROM grupos").
		WithArgs("1", 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := nuevoRouterDePrueba(srv)
	req := jsonRequest(http.MethodDelete, "/admin/grupos/1", nil)
	autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("código = %d; esperado 200 al borrar un grupo vacío (body: %s)", w.Code, w.Body.String())
	}
}
