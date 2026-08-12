package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
)

func nuevoServidorDePrueba(t *testing.T) (*Server, sqlmock.Sqlmock) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { mockDB.Close() })

	srv := New()
	srv.DBAuth = mockDB
	srv.JWTKey = []byte("clave-de-prueba-solo-para-tests")
	return srv, mock
}

func jsonRequest(method, url string, body any) *http.Request {
	var reader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, url, reader)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestCrearPersonal(t *testing.T) {
	cuerpoValido := map[string]string{
		"username": "nueva_staff",
		"password": "Contrasena123!",
		"nombre":   "Nueva Staff",
		"rol":      "staff",
		"pin":      "5678",
	}

	t.Run("admin crea staff -> 201", func(t *testing.T) {
		srv, mock := nuevoServidorDePrueba(t)
		mock.ExpectQuery("INSERT INTO usuarios").
			WithArgs(1, "nueva_staff", sqlmock.AnyArg(), "5678", "staff", sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(9))
		mock.MatchExpectationsInOrder(false)

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/admin/personal", cuerpoValido)
		autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("código = %d; esperado 201 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("staff no puede crear cuentas -> 403", func(t *testing.T) {
		srv, _ := nuevoServidorDePrueba(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/admin/personal", cuerpoValido)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("PIN inválido -> 400", func(t *testing.T) {
		srv, _ := nuevoServidorDePrueba(t)
		r := nuevoRouterDePrueba(srv)
		malo := map[string]string{"username": "x", "password": "Contrasena123!", "rol": "staff", "pin": "12"}
		req := jsonRequest(http.MethodPost, "/admin/personal", malo)
		autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("rol inválido -> 400", func(t *testing.T) {
		srv, _ := nuevoServidorDePrueba(t)
		r := nuevoRouterDePrueba(srv)
		malo := map[string]string{"username": "x", "password": "Contrasena123!", "rol": "papa", "pin": "1234"}
		req := jsonRequest(http.MethodPost, "/admin/personal", malo)
		autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("username duplicado -> 409", func(t *testing.T) {
		srv, mock := nuevoServidorDePrueba(t)
		mock.ExpectQuery("INSERT INTO usuarios").
			WillReturnError(&pq.Error{Code: "23505"})

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/admin/personal", cuerpoValido)
		autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("código = %d; esperado 409 (body: %s)", w.Code, w.Body.String())
		}
	})
}

func TestListarPersonal(t *testing.T) {
	srv, mock := nuevoServidorDePrueba(t)
	columnas := []string{"id", "username", "nombre", "rol", "activo", "created_at"}
	mock.ExpectQuery("SELECT(.|\n)*FROM usuarios(.|\n)*WHERE guarderia_id = \\$1").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows(columnas).
			AddRow(1, "admin_demo", "Admin Demo", "admin", true, "2026-01-01T00:00:00Z").
			AddRow(2, "staff_demo", nil, "staff", true, "2026-01-02T00:00:00Z"))

	r := nuevoRouterDePrueba(srv)
	req := jsonRequest(http.MethodGet, "/admin/personal", nil)
	autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
	}
	var personal []Personal
	if err := json.Unmarshal(w.Body.Bytes(), &personal); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v", err)
	}
	if len(personal) != 2 {
		t.Fatalf("se esperaban 2 cuentas, se recibieron %d", len(personal))
	}
}

func TestActualizarPersonalNoPuedeAutodesactivarse(t *testing.T) {
	srv, _ := nuevoServidorDePrueba(t)
	r := nuevoRouterDePrueba(srv)

	// generarTokenPrueba fija UserID=1 -- probamos que ese mismo admin no
	// pueda desactivarse a sí mismo llamando PUT /admin/personal/1.
	req := jsonRequest(http.MethodPut, "/admin/personal/1", map[string]any{
		"nombre": "Yo Mismo", "rol": "admin", "activo": false,
	})
	autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("código = %d; esperado 400 al intentar autodesactivarse (body: %s)", w.Code, w.Body.String())
	}
}
