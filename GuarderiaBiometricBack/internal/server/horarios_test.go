package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestObtenerTurnos(t *testing.T) {
	t.Run("cuenta de otra guardería -> 404", func(t *testing.T) {
		srv, mock := nuevoServidorDePrueba(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("9", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/admin/horarios/9", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("código = %d; esperado 404 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("staff no puede ver horarios -> 403", func(t *testing.T) {
		srv, _ := nuevoServidorDePrueba(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/admin/horarios/4", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("regresa los 7 días aunque no haya turno capturado -> 200", func(t *testing.T) {
		srv, mock := nuevoServidorDePrueba(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("4", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery("SELECT dia_semana, hora_entrada, hora_salida FROM horarios_personal").
			WithArgs("4").
			// lib/pq entrega TIME con una fecha ficticia por delante -- el
			// handler debe recortarla (soloHora) antes de responder.
			WillReturnRows(sqlmock.NewRows([]string{"dia_semana", "hora_entrada", "hora_salida"}).
				AddRow(0, "0000-01-01T08:00:00Z", "0000-01-01T14:00:00Z"))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/admin/horarios/4", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		var turnos []TurnoDia
		if err := json.Unmarshal(w.Body.Bytes(), &turnos); err != nil {
			t.Fatalf("respuesta no es JSON válido: %v", err)
		}
		if len(turnos) != 7 {
			t.Fatalf("se esperaban 7 días, se recibieron %d", len(turnos))
		}
		if turnos[0].HoraEntrada == nil || *turnos[0].HoraEntrada != "08:00:00" {
			t.Errorf("lunes debería tener hora_entrada 08:00:00, se recibió: %+v", turnos[0])
		}
		if turnos[1].HoraEntrada != nil {
			t.Errorf("martes no tiene turno capturado, se esperaba null, se recibió: %+v", turnos[1])
		}
	})
}

func TestGuardarTurno(t *testing.T) {
	t.Run("día inválido -> 400", func(t *testing.T) {
		srv, _ := nuevoServidorDePrueba(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPut, "/admin/horarios/4/9", map[string]string{"hora_entrada": "08:00", "hora_salida": "14:00"})
		autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("admin guarda un turno -> 200", func(t *testing.T) {
		srv, mock := nuevoServidorDePrueba(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("4", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectExec("INSERT INTO horarios_personal").
			WithArgs("4", 1, 0, "08:00", "14:00").
			WillReturnResult(sqlmock.NewResult(1, 1))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPut, "/admin/horarios/4/0", map[string]string{"hora_entrada": "08:00", "hora_salida": "14:00"})
		autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})
}

func TestGuardarHoras(t *testing.T) {
	t.Run("horas fuera de rango -> 400", func(t *testing.T) {
		srv, mock := nuevoServidorDePrueba(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("4", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPut, "/admin/horas/4/2026-08-12", map[string]float64{"horas_trabajadas": 30})
		autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("admin guarda horas trabajadas -> 200", func(t *testing.T) {
		srv, mock := nuevoServidorDePrueba(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("4", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectExec("INSERT INTO registro_horas").
			WithArgs("4", 1, "2026-08-12", 6.5, "").
			WillReturnResult(sqlmock.NewResult(1, 1))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPut, "/admin/horas/4/2026-08-12", map[string]any{"horas_trabajadas": 6.5, "observaciones": ""})
		autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})
}
