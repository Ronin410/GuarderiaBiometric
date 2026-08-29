package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

// nuevoServidorDePruebaPlataforma arma un Server con DB y DBAuth apuntando
// al MISMO mock -- handleListarGuarderiasPlataforma consulta ambas (ver el
// comentario largo en plataforma.go sobre por qué guarderias/usuarios viven
// conceptualmente aparte de hijos/logs_acceso), y en este despliegue real
// (y en la mayoría de las pruebas de este paquete) son la misma base
// física de todos modos.
func nuevoServidorDePruebaPlataforma(t *testing.T) (*Server, sqlmock.Sqlmock) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { mockDB.Close() })

	srv := New()
	srv.DB = mockDB
	srv.DBAuth = mockDB
	srv.PlatformAdminKey = "clave-de-plataforma-de-prueba"
	return srv, mock
}

func TestListarGuarderiasPlataforma(t *testing.T) {
	t.Run("sin llave -> 401", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaPlataforma(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/plataforma/guarderias", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("código = %d; esperado 401 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("llave incorrecta -> 401", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaPlataforma(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/plataforma/guarderias", nil)
		req.Header.Set("X-Platform-Key", "no-es-la-que-es")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("código = %d; esperado 401 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("PLATFORM_ADMIN_KEY sin configurar -> 404", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaPlataforma(t)
		srv.PlatformAdminKey = ""
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/plataforma/guarderias", nil)
		req.Header.Set("X-Platform-Key", "cualquier-cosa")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("código = %d; esperado 404 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("llave correcta -> 200, junta papás/staff/niños/último acceso por guardería", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaPlataforma(t)

		mock.ExpectQuery("SELECT(.|\n)*FROM guarderias(.|\n)*LEFT JOIN usuarios").
			WillReturnRows(sqlmock.NewRows([]string{"id", "nombre", "slug", "direccion", "created_at", "total_papas", "total_staff"}).
				AddRow(1, "Guardería Uno", "guarderia-uno", "Calle Falsa 123", "2026-08-01T10:00:00Z", 12, 3).
				AddRow(2, "Guardería Dos", "guarderia-dos", nil, "2026-08-15T10:00:00Z", 0, 1))

		mock.ExpectQuery("SELECT guarderia_id, COUNT\\(\\*\\) FROM hijos").
			WillReturnRows(sqlmock.NewRows([]string{"guarderia_id", "count"}).
				AddRow(1, 15))

		mock.ExpectQuery("SELECT guarderia_id, MAX\\(creado_en\\) FROM logs_acceso").
			WillReturnRows(sqlmock.NewRows([]string{"guarderia_id", "max"}).
				AddRow(1, "2026-08-29T09:00:00Z"))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/plataforma/guarderias", nil)
		req.Header.Set("X-Platform-Key", "clave-de-plataforma-de-prueba")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		body := w.Body.String()
		// Guardería 1: tiene niños y un último acceso registrado.
		for _, esperado := range []string{`"total_ninos":15`, `"total_papas":12`, `"total_staff":3`, `"ultimo_acceso":"2026-08-29T09:00:00Z"`} {
			if !strings.Contains(body, esperado) {
				t.Errorf("body no trae %q (datos de la guardería 1): %s", esperado, body)
			}
		}
		// Guardería 2: sin niños ni accesos registrados -- 0 y null, no error.
		for _, esperado := range []string{`"total_ninos":0`, `"ultimo_acceso":null`} {
			if !strings.Contains(body, esperado) {
				t.Errorf("body no trae %q (valores en cero/null de la guardería 2): %s", esperado, body)
			}
		}
	})
}
