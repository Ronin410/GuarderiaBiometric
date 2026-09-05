package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestObtenerMenuSemanal(t *testing.T) {
	t.Run("sin parámetros de fecha -> 400", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/menu-semanal", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("rango válido -> 200 con los días guardados", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		desayuno := "Fruta picada"
		mock.ExpectQuery("SELECT fecha, desayuno, comida, merienda FROM menu_semanal").
			WithArgs(1, "2026-08-10", "2026-08-14").
			WillReturnRows(sqlmock.NewRows([]string{"fecha", "desayuno", "comida", "merienda"}).
				AddRow("2026-08-10T00:00:00Z", desayuno, nil, nil))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/menu-semanal?inicio=2026-08-10&fin=2026-08-14", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		var dias []DiaMenu
		if err := json.Unmarshal(w.Body.Bytes(), &dias); err != nil {
			t.Fatalf("respuesta no es JSON válido: %v", err)
		}
		if len(dias) != 1 || dias[0].Fecha != "2026-08-10" {
			t.Fatalf("se esperaba un día con fecha 2026-08-10, se recibió: %+v", dias)
		}
	})

	t.Run("papá también puede consultar el menú -> 200", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT fecha, desayuno, comida, merienda FROM menu_semanal").
			WithArgs(1, "2026-08-10", "2026-08-14").
			WillReturnRows(sqlmock.NewRows([]string{"fecha", "desayuno", "comida", "merienda"}))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/padre/menu-semanal?inicio=2026-08-10&fin=2026-08-14", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})
}

func TestGuardarDiaMenu(t *testing.T) {
	t.Run("fecha inválida -> 400", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPut, "/menu-semanal/no-es-una-fecha", map[string]string{"desayuno": "Fruta"})
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("papá no puede editar el menú -> 403", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPut, "/menu-semanal/2026-08-10", map[string]string{"desayuno": "Fruta"})
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("staff guarda un día -> 200", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectExec("INSERT INTO menu_semanal").
			WithArgs(1, "2026-08-10", "Fruta picada", "Pollo con verduras", "Galletas").
			WillReturnResult(sqlmock.NewResult(1, 1))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPut, "/menu-semanal/2026-08-10", map[string]string{
			"desayuno": "Fruta picada", "comida": "Pollo con verduras", "merienda": "Galletas",
		})
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})
}
