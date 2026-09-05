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
			WillReturnError(&pq.Error{Code: "23505", Constraint: "usuarios_username_key"})

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
	columnas := []string{"id", "username", "nombre", "rol", "activo", "created_at", "permisos"}
	mock.ExpectQuery("SELECT(.|\n)*FROM usuarios(.|\n)*WHERE guarderia_id = \\$1").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows(columnas).
			AddRow(1, "admin_demo", "Admin Demo", "admin", true, "2026-01-01T00:00:00Z", nil).
			AddRow(2, "staff_demo", nil, "staff", true, "2026-01-02T00:00:00Z", "{pagos,menu}"))

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
	if personal[0].Permisos != nil {
		t.Errorf("admin_demo no tiene permisos personalizados en la BD (columna NULL); se esperaba nil, se obtuvo %v", personal[0].Permisos)
	}
	if len(personal[1].Permisos) != 2 || personal[1].Permisos[0] != "pagos" || personal[1].Permisos[1] != "menu" {
		t.Errorf("permisos de staff_demo = %v; esperado [pagos menu]", personal[1].Permisos)
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

func TestActualizarPermisosPersonal(t *testing.T) {
	t.Run("array de áreas válidas -> guarda esa lista exacta", func(t *testing.T) {
		srv, mock := nuevoServidorDePrueba(t)
		r := nuevoRouterDePrueba(srv)

		mock.ExpectExec("UPDATE usuarios SET permisos = (.|\n)*WHERE id = \\$2 AND guarderia_id = \\$3").
			WithArgs(pq.Array([]string{"pagos", "menu"}), "5", 1).
			WillReturnResult(sqlmock.NewResult(0, 1))

		req := jsonRequest(http.MethodPut, "/admin/personal/5/permisos", map[string]any{
			"permisos": []string{"pagos", "menu"},
		})
		autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectativas de sqlmock no cumplidas: %v", err)
		}
	})

	t.Run("permisos null (u omitido) -> vuelve la cuenta a acceso completo", func(t *testing.T) {
		srv, mock := nuevoServidorDePrueba(t)
		r := nuevoRouterDePrueba(srv)

		mock.ExpectExec("UPDATE usuarios SET permisos = NULL WHERE id = \\$1 AND guarderia_id = \\$2").
			WithArgs("5", 1).
			WillReturnResult(sqlmock.NewResult(0, 1))

		req := jsonRequest(http.MethodPut, "/admin/personal/5/permisos", map[string]any{})
		autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectativas de sqlmock no cumplidas: %v", err)
		}
	})

	t.Run("área desconocida -> 400, sin tocar la BD", func(t *testing.T) {
		srv, _ := nuevoServidorDePrueba(t)
		r := nuevoRouterDePrueba(srv)

		req := jsonRequest(http.MethodPut, "/admin/personal/5/permisos", map[string]any{
			"permisos": []string{"pagos", "contabilidad-secreta"},
		})
		autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("staff no puede tocar los permisos de nadie", func(t *testing.T) {
		srv, _ := nuevoServidorDePrueba(t)
		r := nuevoRouterDePrueba(srv)

		req := jsonRequest(http.MethodPut, "/admin/personal/5/permisos", map[string]any{
			"permisos": []string{"pagos"},
		})
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})
}
