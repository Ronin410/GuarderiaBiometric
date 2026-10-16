package server

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

// TestGenerarCodigoKiosco cubre el lado de la tablet -- cualquier staff
// puede pedir un código nuevo (no es exclusivo de admin, es parte de la
// misma pantalla de Recepción que ya usan todos).
func TestGenerarCodigoKiosco(t *testing.T) {
	t.Run("staff -> 200 con token y vigencia", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("INSERT INTO codigos_kiosco").
			WithArgs(1, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))

		r := nuevoRouterDePrueba(srv)
		req := httptest.NewRequest(http.MethodPost, "/admin/codigo-kiosco", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("papá -> 403", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := httptest.NewRequest(http.MethodPost, "/admin/codigo-kiosco", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})
}

// TestEstadoCodigoKiosco cubre el polling que hace la tablet para saber
// cuándo pedir un código nuevo sin esperar a que se cumpla el minuto.
func TestEstadoCodigoKiosco(t *testing.T) {
	t.Run("todavía sin usar ni vencer -> usado:false, expirado:false", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT expira_en, usado_en FROM codigos_kiosco").
			WithArgs("7", 1).
			WillReturnRows(sqlmock.NewRows([]string{"expira_en", "usado_en"}).
				AddRow(time.Now().Add(50*time.Second), nil))

		r := nuevoRouterDePrueba(srv)
		req := httptest.NewRequest(http.MethodGet, "/admin/codigo-kiosco/7", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		body := w.Body.String()
		if !strings.Contains(body, `"usado":false`) || !strings.Contains(body, `"expirado":false`) {
			t.Errorf("body no refleja el estado esperado: %s", body)
		}
	})

	t.Run("ya usado -> usado:true", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT expira_en, usado_en FROM codigos_kiosco").
			WithArgs("7", 1).
			WillReturnRows(sqlmock.NewRows([]string{"expira_en", "usado_en"}).
				AddRow(time.Now().Add(50*time.Second), time.Now()))

		r := nuevoRouterDePrueba(srv)
		req := httptest.NewRequest(http.MethodGet, "/admin/codigo-kiosco/7", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if !strings.Contains(w.Body.String(), `"usado":true`) {
			t.Errorf("esperaba usado:true (body: %s)", w.Body.String())
		}
	})
}

// TestCanjearCodigoKiosco cubre el lado del papá: escanea el QR de la
// tablet y recibe SUS hijos -- nunca los de nadie más.
func TestCanjearCodigoKiosco(t *testing.T) {
	canjearRequest := func(t *testing.T, jwtKey []byte, rol string, token string) *http.Request {
		t.Helper()
		req := jsonRequest(http.MethodPost, "/padre/codigo-kiosco/canjear", map[string]string{"token": token})
		autenticarRequestPrueba(t, req, jwtKey, rol, time.Hour)
		return req
	}

	t.Run("código vigente y sin usar -> 200 con mis hijos", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT id, expira_en, usado_en FROM codigos_kiosco").
			WithArgs("ABC123", 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "expira_en", "usado_en"}).
				AddRow(9, time.Now().Add(50*time.Second), nil))
		mock.ExpectExec("UPDATE codigos_kiosco SET usado_en = NOW\\(\\)").
			WithArgs(1, 9).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery("SELECT h.id, h.nombre_niño(.|\n)*FROM hijos").
			WithArgs(1, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "nombre_niño", "ultimo_estado"}).AddRow(5, "Ryan", "AUSENTE"))

		r := nuevoRouterDePrueba(srv)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, canjearRequest(t, srv.JWTKey, "papa", "ABC123"))

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectativas de sqlmock no cumplidas: %v", err)
		}
	})

	t.Run("staff/admin -> 403, no llega a tocar la base de datos", func(t *testing.T) {
		for _, rol := range []string{"staff", "admin"} {
			srv, _ := nuevoServidorDePruebaConDB(t)
			r := nuevoRouterDePrueba(srv)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, canjearRequest(t, srv.JWTKey, rol, "ABC123"))
			if w.Code != http.StatusForbidden {
				t.Errorf("rol=%s: código = %d; esperado 403", rol, w.Code)
			}
		}
	})

	t.Run("código ya usado -> 410", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT id, expira_en, usado_en FROM codigos_kiosco").
			WithArgs("ABC123", 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "expira_en", "usado_en"}).
				AddRow(9, time.Now().Add(50*time.Second), time.Now().Add(-10*time.Second)))

		r := nuevoRouterDePrueba(srv)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, canjearRequest(t, srv.JWTKey, "papa", "ABC123"))

		if w.Code != http.StatusGone {
			t.Fatalf("código = %d; esperado 410 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("código vencido -> 410", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT id, expira_en, usado_en FROM codigos_kiosco").
			WithArgs("ABC123", 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "expira_en", "usado_en"}).
				AddRow(9, time.Now().Add(-5*time.Second), nil))

		r := nuevoRouterDePrueba(srv)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, canjearRequest(t, srv.JWTKey, "papa", "ABC123"))

		if w.Code != http.StatusGone {
			t.Fatalf("código = %d; esperado 410 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("código de otra guardería -> 404", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT id, expira_en, usado_en FROM codigos_kiosco").
			WithArgs("ABC123", 1).
			WillReturnError(sql.ErrNoRows)

		r := nuevoRouterDePrueba(srv)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, canjearRequest(t, srv.JWTKey, "papa", "ABC123"))

		if w.Code != http.StatusNotFound {
			t.Fatalf("código = %d; esperado 404 (body: %s)", w.Code, w.Body.String())
		}
	})
}

// TestConfirmarAsistenciaPadre cubre el paso final: el papá elige un hijo
// (siempre el suyo, nunca el de otro) y confirma, con padre_id saliendo
// SIEMPRE de su sesión.
func TestConfirmarAsistenciaPadre(t *testing.T) {
	cuerpo := map[string]any{"hijo_id": 5, "codigo_kiosco_id": 9}

	t.Run("hijo propio y código canjeado por mí -> 200", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS(.|\n)*FROM tutor_hijos").
			WithArgs("5", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery("SELECT usado_por FROM codigos_kiosco").
			WithArgs(9, 1).
			WillReturnRows(sqlmock.NewRows([]string{"usado_por"}).AddRow(1))
		mock.ExpectQuery("SELECT tipo_movimiento(.|\n)*FROM asistencia").
			WithArgs(5, 1).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectExec("INSERT INTO asistencia").
			WithArgs(1, 5, "Confirmado por el propio tutor desde su app, vía QR del kiosco", "ENTRADA", 1, 9).
			WillReturnResult(sqlmock.NewResult(1, 1))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/padre/confirmar-asistencia", cuerpo)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectativas de sqlmock no cumplidas: %v", err)
		}
	})

	t.Run("hijo que no es mío -> 403, nunca llega a tocar el código", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS(.|\n)*FROM tutor_hijos").
			WithArgs("5", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/padre/confirmar-asistencia", cuerpo)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("código canjeado por OTRO papá -> 400, no inserta asistencia", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS(.|\n)*FROM tutor_hijos").
			WithArgs("5", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery("SELECT usado_por FROM codigos_kiosco").
			WithArgs(9, 1).
			WillReturnRows(sqlmock.NewRows([]string{"usado_por"}).AddRow(999))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/padre/confirmar-asistencia", cuerpo)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectativas de sqlmock no cumplidas: %v", err)
		}
	})

	t.Run("staff -> 403", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/padre/confirmar-asistencia", cuerpo)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})
}
