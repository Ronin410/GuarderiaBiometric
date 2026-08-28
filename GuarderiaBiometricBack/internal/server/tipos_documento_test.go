package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
)

func TestSlugificar(t *testing.T) {
	casos := []struct {
		entrada  string
		esperado string
	}{
		{"Acta de Nacimiento", "acta_de_nacimiento"},
		{"CURP", "curp"},
		{"Constancia de Estudios (última)", "constancia_de_estudios_ultima"},
		{"  Comprobante de Domicilio  ", "comprobante_de_domicilio"},
		{"Cartilla de Vacunación / Niño", "cartilla_de_vacunacion_nino"},
		{"¡¿???!", ""},
	}
	for _, c := range casos {
		if got := slugificar(c.entrada); got != c.esperado {
			t.Errorf("slugificar(%q) = %q; esperado %q", c.entrada, got, c.esperado)
		}
	}
}

func TestListarTiposDocumento(t *testing.T) {
	srv, mock := nuevoServidorDePruebaConDB(t)
	mock.ExpectQuery("SELECT(.|\n)*FROM tipos_documento(.|\n)*WHERE t.guarderia_id = \\$1").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "clave", "nombre", "orden", "count"}).
			AddRow(1, "acta_nacimiento", "Acta de Nacimiento", 0, 3).
			AddRow(2, "constancia_estudios", "Constancia de Estudios", 1, 0))

	r := nuevoRouterDePrueba(srv)
	req := jsonRequest(http.MethodGet, "/admin/tipos-documento", nil)
	autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
	}
}

// TestStaffNuncaAccedeATiposDocumento cubre el pedido explícito de "el
// staff no tenga acceso al apartado de sistema del menú nunca" -- a
// diferencia del resto de áreas (RequireArea), esto usa RequireAdmin() sin
// importar permisos personalizados, así que ni siquiera hace falta simular
// una cuenta con permisos -- staff se rechaza siempre.
func TestStaffNuncaAccedeATiposDocumento(t *testing.T) {
	srv, _ := nuevoServidorDePruebaConDB(t)
	r := nuevoRouterDePrueba(srv)
	req := jsonRequest(http.MethodGet, "/admin/tipos-documento", nil)
	autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("código = %d; esperado 403 -- staff nunca debe entrar a Configuración (body: %s)", w.Code, w.Body.String())
	}
}

func TestCrearTipoDocumento(t *testing.T) {
	t.Run("nombre vacío -> 400", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/admin/tipos-documento", map[string]string{"nombre": "   "})
		autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("papá no puede crear tipos de documento -> 403", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/admin/tipos-documento", map[string]string{"nombre": "Constancia de Estudios"})
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("staff no puede crear tipos de documento (RequireAdmin, no permisos) -> 403", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/admin/tipos-documento", map[string]string{"nombre": "Constancia de Estudios"})
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("admin crea un tipo nuevo -> 201, deriva la clave del nombre", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("INSERT INTO tipos_documento").
			WithArgs(1, "constancia_de_estudios", "Constancia de Estudios").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/admin/tipos-documento", map[string]string{"nombre": "Constancia de Estudios"})
		autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("código = %d; esperado 201 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("nombre que ya existe en esta guardería -> 409", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("INSERT INTO tipos_documento").
			WithArgs(1, "curp", "CURP").
			WillReturnError(&pq.Error{Code: "23505", Constraint: "tipos_documento_guarderia_id_clave_key"})

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/admin/tipos-documento", map[string]string{"nombre": "CURP"})
		autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusConflict {
			t.Fatalf("código = %d; esperado 409 (body: %s)", w.Code, w.Body.String())
		}
	})
}

func TestEliminarTipoDocumentoEnUso(t *testing.T) {
	srv, mock := nuevoServidorDePruebaConDB(t)
	mock.ExpectQuery("SELECT clave FROM tipos_documento").
		WithArgs("1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"clave"}).AddRow("curp"))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM documentos_nino").
		WithArgs(1, "curp").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	r := nuevoRouterDePrueba(srv)
	req := jsonRequest(http.MethodDelete, "/admin/tipos-documento/1", nil)
	autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("código = %d; esperado 409 al borrar un tipo con documentos subidos (body: %s)", w.Code, w.Body.String())
	}
}

func TestEliminarTipoDocumentoSinUso(t *testing.T) {
	srv, mock := nuevoServidorDePruebaConDB(t)
	mock.ExpectQuery("SELECT clave FROM tipos_documento").
		WithArgs("1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"clave"}).AddRow("otro"))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM documentos_nino").
		WithArgs(1, "otro").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectExec("DELETE FROM tipos_documento").
		WithArgs("1", 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := nuevoRouterDePrueba(srv)
	req := jsonRequest(http.MethodDelete, "/admin/tipos-documento/1", nil)
	autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("código = %d; esperado 200 al borrar un tipo sin uso (body: %s)", w.Code, w.Body.String())
	}
}
