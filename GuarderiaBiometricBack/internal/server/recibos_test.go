package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestObtenerRecibo(t *testing.T) {
	columnas := []string{"id", "monto", "concepto", "periodo", "fecha_pago", "metodo_pago", "observaciones", "hijo_id", "nombre_niño", "guarderia_nombre", "direccion"}

	t.Run("staff ve cualquier recibo de su guardería -> 200", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT(.|\n)*FROM pagos p(.|\n)*WHERE p.id = \\$1 AND p.guarderia_id = \\$2").
			WithArgs("5", 1).
			WillReturnRows(sqlmock.NewRows(columnas).
				AddRow(5, 1500.0, "Colegiatura", "2026-08", "2026-08-05T00:00:00Z", "efectivo", "", 3, "Sofia Ramirez", "Guardería Demo", "Av. Insurgentes 45"))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/pagos/5/recibo", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		var rec ReciboPago
		if err := json.Unmarshal(w.Body.Bytes(), &rec); err != nil {
			t.Fatalf("respuesta no es JSON válido: %v", err)
		}
		if rec.Folio != "REC-000005" {
			t.Errorf("folio = %q; esperado REC-000005", rec.Folio)
		}
		if rec.FechaPago != "2026-08-05" {
			t.Errorf("fecha_pago = %q; esperado recortada a la fecha", rec.FechaPago)
		}
	})

	t.Run("papá dueño del pago -> 200", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT(.|\n)*FROM pagos p(.|\n)*WHERE p.id = \\$1 AND p.guarderia_id = \\$2").
			WithArgs("5", 1).
			WillReturnRows(sqlmock.NewRows(columnas).
				AddRow(5, 1500.0, "Colegiatura", "2026-08", "2026-08-05T00:00:00Z", "efectivo", "", 3, "Sofia Ramirez", "Guardería Demo", nil))
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(3, 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/padre/pagos/5/recibo", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("papá NO dueño del pago -> 403", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT(.|\n)*FROM pagos p(.|\n)*WHERE p.id = \\$1 AND p.guarderia_id = \\$2").
			WithArgs("5", 1).
			WillReturnRows(sqlmock.NewRows(columnas).
				AddRow(5, 1500.0, "Colegiatura", "2026-08", "2026-08-05T00:00:00Z", "efectivo", "", 3, "Sofia Ramirez", "Guardería Demo", nil))
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(3, 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/padre/pagos/5/recibo", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})
}

func TestEnviarRecordatorios(t *testing.T) {
	t.Run("sin periodo -> 400", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/pagos/recordatorio", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("papá no puede disparar recordatorios -> 403", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/pagos/recordatorio?periodo=2026-08", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("push no configurado -> 503", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/pagos/recordatorio?periodo=2026-08", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("código = %d; esperado 503 sin VAPID configurado (body: %s)", w.Code, w.Body.String())
		}
	})
}
