package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestListarAusenciasHijo(t *testing.T) {
	t.Run("niño ajeno -> 403", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("9", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/padre/hijos/9/ausencias", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("papá ve las ausencias de su hijo -> 200", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("5", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery("SELECT id, hijo_id, fecha, COALESCE\\(motivo, ''\\), creado_en FROM ausencias_planificadas").
			WithArgs("5").
			WillReturnRows(sqlmock.NewRows([]string{"id", "hijo_id", "fecha", "motivo", "creado_en"}).
				AddRow(1, 5, "2026-09-01T00:00:00Z", "Cita médica", "2026-08-13T10:00:00Z"))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/padre/hijos/5/ausencias", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		var ausencias []AusenciaPlanificada
		if err := json.Unmarshal(w.Body.Bytes(), &ausencias); err != nil {
			t.Fatalf("respuesta no es JSON válido: %v", err)
		}
		if len(ausencias) != 1 || ausencias[0].Fecha != "2026-09-01" || ausencias[0].Motivo != "Cita médica" {
			t.Fatalf("se esperaba una ausencia recortada a la fecha, se recibió: %+v", ausencias)
		}
	})
}

func TestCrearAusencia(t *testing.T) {
	manana := time.Now().In(zonaMazatlan()).AddDate(0, 0, 1).Format("2006-01-02")
	ayer := time.Now().In(zonaMazatlan()).AddDate(0, 0, -1).Format("2006-01-02")

	t.Run("niño ajeno -> 403", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("9", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/padre/hijos/9/ausencias", map[string]string{"fecha_inicio": manana})
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("fecha pasada -> 400", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("5", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/padre/hijos/5/ausencias", map[string]string{"fecha_inicio": ayer})
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("papá reporta un día -> 201", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("5", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectExec("INSERT INTO ausencias_planificadas").
			WithArgs("5", 1, manana, "Cita médica", 1).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectQuery("SELECT nombre_niño FROM hijos").
			WithArgs("5").
			WillReturnRows(sqlmock.NewRows([]string{"nombre_niño"}).AddRow("Ryan"))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/padre/hijos/5/ausencias", map[string]string{
			"fecha_inicio": manana, "motivo": "Cita médica",
		})
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("código = %d; esperado 201 (body: %s)", w.Code, w.Body.String())
		}
	})
}

func TestCancelarAusencia(t *testing.T) {
	t.Run("no encontrada -> 404", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectExec("DELETE FROM ausencias_planificadas").
			WithArgs("99", 1).
			WillReturnResult(sqlmock.NewResult(0, 0))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodDelete, "/padre/ausencias/99", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("código = %d; esperado 404 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("papá cancela su propia ausencia -> 200", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectExec("DELETE FROM ausencias_planificadas").
			WithArgs("3", 1).
			WillReturnResult(sqlmock.NewResult(0, 1))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodDelete, "/padre/ausencias/3", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})
}

func TestListarAusenciasStaff(t *testing.T) {
	t.Run("papá no puede ver el listado general -> 403", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/ausencias", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("staff ve las ausencias del rango -> 200", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT a.id, a.hijo_id, a.fecha, COALESCE\\(a.motivo, ''\\), a.creado_en, h.nombre_niño").
			WithArgs(1, "2026-09-01", "2026-09-05").
			WillReturnRows(sqlmock.NewRows([]string{"id", "hijo_id", "fecha", "motivo", "creado_en", "hijo_nombre"}).
				AddRow(1, 5, "2026-09-01T00:00:00Z", "Cita médica", "2026-08-13T10:00:00Z", "Valentina Cruz"))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/ausencias?desde=2026-09-01&hasta=2026-09-05", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		var ausencias []AusenciaConNino
		if err := json.Unmarshal(w.Body.Bytes(), &ausencias); err != nil {
			t.Fatalf("respuesta no es JSON válido: %v", err)
		}
		if len(ausencias) != 1 || ausencias[0].HijoNombre != "Valentina Cruz" {
			t.Fatalf("se esperaba una ausencia de Valentina Cruz, se recibió: %+v", ausencias)
		}
	})
}
