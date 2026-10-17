package server

import (
	"bytes"
	"database/sql"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

// formularioSeguimientoDePrueba arma el mismo multipart/form-data que manda
// el frontend a /seguimiento (soporta subir fotos junto con el texto, ver
// handleGuardarSeguimiento) -- sin campo "fotos", así que
// form.File["fotos"] sale vacío y el ciclo de subida a S3 no corre.
func formularioSeguimientoDePrueba(t *testing.T, campos map[string]string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for campo, valor := range campos {
		if err := w.WriteField(campo, valor); err != nil {
			t.Fatalf("no se pudo escribir el campo %q del formulario: %v", campo, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("no se pudo cerrar el formulario multipart: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/seguimiento", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

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

// TestGuardarSeguimientoRegistraHistorial cubre lo que antes no dejaba
// rastro: guardar la bitácora de un niño dos veces el mismo día (una
// corrección) debe dejar attribuido quién guardó cada versión -- no solo
// pisar el valor anterior en seguimiento_diario, sino además dejar una
// fila nueva en seguimiento_diario_historial por cada guardado.
func TestGuardarSeguimientoRegistraHistorial(t *testing.T) {
	srv, mock := nuevoServidorDePruebaConDB(t)

	mock.ExpectQuery("INSERT INTO seguimiento_diario(.|\n)*ON CONFLICT(.|\n)*RETURNING id").
		WithArgs("5", 1, sqlmock.AnyArg(), "Todo", "Poco", "Nada", "Seco", "Sin novedad", true, 1, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))
	mock.ExpectExec("INSERT INTO seguimiento_diario_historial").
		WithArgs(10, "5", 1, sqlmock.AnyArg(), "Todo", "Poco", "Nada", "Seco", "Sin novedad", true, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT p.id, p.celular(.|\n)*FROM padres").
		WithArgs("5").
		WillReturnError(sql.ErrNoRows)

	r := nuevoRouterDePrueba(srv)
	req := formularioSeguimientoDePrueba(t, map[string]string{
		"hijo_id":       "5",
		"desayuno":      "Todo",
		"comida":        "Poco",
		"merienda":      "Nada",
		"esfinter":      "Seco",
		"observaciones": "Sin novedad",
		"durmio":        "true",
	})
	autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectativas de sqlmock no cumplidas -- ¿no se guardó el historial? %v", err)
	}
}
