package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

// TestObtenerSeguimientoIncluyeAsistencia cubre "acá no aparece que el niño
// entró o ya salió" -- /seguimiento/:hijo_id ahora también trae la hora de
// entrada/salida que registró el kiosco ese día, aparte de lo que el staff
// escribió a mano en la bitácora.
func TestObtenerSeguimientoIncluyeAsistencia(t *testing.T) {
	t.Run("con entrada y salida ese día -> las incluye en HH:MM", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)

		mock.ExpectQuery("SELECT(.|\n)*FROM seguimiento_diario(.|\n)*WHERE hijo_id = \\$1 AND fecha = \\$2").
			WithArgs("5", "2026-08-29").
			WillReturnRows(sqlmock.NewRows([]string{"id", "hijo_id", "fecha", "desayuno", "comida", "merienda", "esfinter", "observaciones", "durmio"}).
				AddRow(10, 5, "2026-08-29", "Nada", "Poco", "Bien", "Pipi, Popo", "", false))
		mock.ExpectQuery("SELECT url FROM fotos_seguimiento").
			WithArgs(10).
			WillReturnRows(sqlmock.NewRows([]string{"url"}))
		mock.ExpectQuery("SELECT(.|\n)*FROM asistencia(.|\n)*WHERE hijo_id = \\$1 AND fecha_hora::date = \\$2::date").
			WithArgs("5", "2026-08-29").
			WillReturnRows(sqlmock.NewRows([]string{"entrada", "salida"}).
				AddRow(time.Date(2026, 8, 29, 8, 15, 0, 0, time.UTC), time.Date(2026, 8, 29, 14, 30, 0, 0, time.UTC)))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/seguimiento/5?fecha=2026-08-29", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		body := w.Body.String()
		if !strings.Contains(body, `"hora_entrada":`) || !strings.Contains(body, `"hora_salida":`) {
			t.Errorf("body no trae hora_entrada/hora_salida: %s", body)
		}
	})

	t.Run("sin ningún movimiento ese día -> hora_entrada y hora_salida en null, sin error", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)

		mock.ExpectQuery("SELECT(.|\n)*FROM seguimiento_diario(.|\n)*WHERE hijo_id = \\$1 AND fecha = \\$2").
			WithArgs("5", "2026-08-29").
			WillReturnRows(sqlmock.NewRows([]string{"id", "hijo_id", "fecha", "desayuno", "comida", "merienda", "esfinter", "observaciones", "durmio"}).
				AddRow(10, 5, "2026-08-29", "Nada", "Poco", "Bien", "Pipi, Popo", "", false))
		mock.ExpectQuery("SELECT url FROM fotos_seguimiento").
			WithArgs(10).
			WillReturnRows(sqlmock.NewRows([]string{"url"}))
		mock.ExpectQuery("SELECT(.|\n)*FROM asistencia(.|\n)*WHERE hijo_id = \\$1 AND fecha_hora::date = \\$2::date").
			WithArgs("5", "2026-08-29").
			WillReturnRows(sqlmock.NewRows([]string{"entrada", "salida"}).AddRow(nil, nil))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/seguimiento/5?fecha=2026-08-29", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		body := w.Body.String()
		if !strings.Contains(body, `"hora_entrada":null`) || !strings.Contains(body, `"hora_salida":null`) {
			t.Errorf("body debería traer hora_entrada/hora_salida en null: %s", body)
		}
	})
}
