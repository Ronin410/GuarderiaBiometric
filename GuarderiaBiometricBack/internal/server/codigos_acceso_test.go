package server

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

// TestGenerarCodigoAcceso cubre el alta del código QR/corto que un papá usa
// para delegar una recogida -- "quiero poder decir quién puede ir por el
// niño sin que esa persona esté enrolada con su rostro".
func TestGenerarCodigoAcceso(t *testing.T) {
	cuerpoValido := map[string]any{
		"hijos_ids":          []int{5},
		"persona_autorizada": "Tía Lupita",
		"dias":               3,
	}

	t.Run("papá, hijo propio, 3 días -> 200 con token y código corto", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)

		mock.ExpectQuery("SELECT EXISTS(.|\n)*FROM tutor_hijos").
			WithArgs("5", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectBegin()
		mock.ExpectQuery("INSERT INTO codigos_acceso").
			WithArgs(1, 1, sqlmock.AnyArg(), sqlmock.AnyArg(), "Tía Lupita", sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(9))
		mock.ExpectExec("INSERT INTO codigos_acceso_hijos").
			WithArgs(9, 5).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/padre/codigos-acceso", cuerpoValido)
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

	t.Run("rol staff/admin -> 403, no llega a tocar la base de datos", func(t *testing.T) {
		for _, rol := range []string{"staff", "admin"} {
			srv, _ := nuevoServidorDePruebaConDB(t)
			r := nuevoRouterDePrueba(srv)
			req := jsonRequest(http.MethodPost, "/padre/codigos-acceso", cuerpoValido)
			autenticarRequestPrueba(t, req, srv.JWTKey, rol, time.Hour)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusForbidden {
				t.Errorf("rol=%s: código = %d; esperado 403 (body: %s)", rol, w.Code, w.Body.String())
			}
		}
	})

	t.Run("hijo que no pertenece al papá -> 403", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)

		mock.ExpectQuery("SELECT EXISTS(.|\n)*FROM tutor_hijos").
			WithArgs("5", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/padre/codigos-acceso", cuerpoValido)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("vigencia fuera de 1/3/7 días -> 400, no llega a tocar la base de datos", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		cuerpo := map[string]any{"hijos_ids": []int{5}, "persona_autorizada": "Tía Lupita", "dias": 2}
		req := jsonRequest(http.MethodPost, "/padre/codigos-acceso", cuerpo)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("sin persona_autorizada -> 400", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		cuerpo := map[string]any{"hijos_ids": []int{5}, "persona_autorizada": "", "dias": 3}
		req := jsonRequest(http.MethodPost, "/padre/codigos-acceso", cuerpo)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})
}

// TestRevocarCodigoAcceso cubre "si quiere revisar algo" aplicado a cortar
// un código antes de que venza solo (se lo mandó a la persona equivocada,
// cambió de opinión).
func TestRevocarCodigoAcceso(t *testing.T) {
	t.Run("código propio y activo -> 200", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)

		mock.ExpectExec("UPDATE codigos_acceso SET revocado_en = NOW\\(\\)").
			WithArgs("9", 1).
			WillReturnResult(sqlmock.NewResult(0, 1))

		r := nuevoRouterDePrueba(srv)
		req := httptest.NewRequest(http.MethodDelete, "/padre/codigos-acceso/9", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("código ajeno o ya revocado -> 404", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)

		mock.ExpectExec("UPDATE codigos_acceso SET revocado_en = NOW\\(\\)").
			WithArgs("9", 1).
			WillReturnResult(sqlmock.NewResult(0, 0))

		r := nuevoRouterDePrueba(srv)
		req := httptest.NewRequest(http.MethodDelete, "/padre/codigos-acceso/9", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("código = %d; esperado 404 (body: %s)", w.Code, w.Body.String())
		}
	})
}

// TestIdentificarPorCodigo cubre el lado del kiosco: escanear/teclear un
// código y que el sistema diga a quién identifica, igual que /identificar
// lo hace con un rostro.
func TestIdentificarPorCodigo(t *testing.T) {
	codigoRequest := func(t *testing.T, jwtKey []byte, codigo string) *http.Request {
		t.Helper()
		req := jsonRequest(http.MethodPost, "/identificar-codigo", map[string]string{"codigo": codigo})
		autenticarRequestPrueba(t, req, jwtKey, "staff", time.Hour)
		return req
	}

	t.Run("código vigente -> 200 con los hijos que autoriza", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)

		vence := time.Now().Add(48 * time.Hour)
		mock.ExpectQuery("SELECT ca.id, ca.padre_id(.|\n)*FROM codigos_acceso").
			WithArgs("ABC123", "ABC123", 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "padre_id", "nombre", "persona_autorizada", "expira_en", "revocado_en"}).
				AddRow(9, 3, "Mamá de Ryan", "Tía Lupita", vence, nil))
		mock.ExpectQuery("SELECT h.id, h.nombre_niño(.|\n)*FROM codigos_acceso_hijos").
			WithArgs(9).
			WillReturnRows(sqlmock.NewRows([]string{"id", "nombre_niño"}).AddRow(5, "Ryan"))

		r := nuevoRouterDePrueba(srv)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, codigoRequest(t, srv.JWTKey, "ABC123"))

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("código inexistente en esta guardería -> 404", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)

		mock.ExpectQuery("SELECT ca.id, ca.padre_id(.|\n)*FROM codigos_acceso").
			WithArgs("XYZ999", "XYZ999", 1).
			WillReturnError(sql.ErrNoRows)

		r := nuevoRouterDePrueba(srv)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, codigoRequest(t, srv.JWTKey, "XYZ999"))

		if w.Code != http.StatusNotFound {
			t.Fatalf("código = %d; esperado 404 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("código revocado -> 410", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)

		vence := time.Now().Add(48 * time.Hour)
		revocado := time.Now().Add(-1 * time.Hour)
		mock.ExpectQuery("SELECT ca.id, ca.padre_id(.|\n)*FROM codigos_acceso").
			WithArgs("ABC123", "ABC123", 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "padre_id", "nombre", "persona_autorizada", "expira_en", "revocado_en"}).
				AddRow(9, 3, "Mamá de Ryan", "Tía Lupita", vence, revocado))

		r := nuevoRouterDePrueba(srv)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, codigoRequest(t, srv.JWTKey, "ABC123"))

		if w.Code != http.StatusGone {
			t.Fatalf("código = %d; esperado 410 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("código vencido -> 410", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)

		vencioHaceRato := time.Now().Add(-48 * time.Hour)
		mock.ExpectQuery("SELECT ca.id, ca.padre_id(.|\n)*FROM codigos_acceso").
			WithArgs("ABC123", "ABC123", 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "padre_id", "nombre", "persona_autorizada", "expira_en", "revocado_en"}).
				AddRow(9, 3, "Mamá de Ryan", "Tía Lupita", vencioHaceRato, nil))

		r := nuevoRouterDePrueba(srv)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, codigoRequest(t, srv.JWTKey, "ABC123"))

		if w.Code != http.StatusGone {
			t.Fatalf("código = %d; esperado 410 (body: %s)", w.Code, w.Body.String())
		}
	})
}

// TestConfirmarAsistenciaConCodigoAcceso cubre que el movimiento se
// revalide server-side al confirmar, no solo al identificar -- un código
// pudo haberse revocado en los segundos entre las dos pantallas.
func TestConfirmarAsistenciaConCodigoAcceso(t *testing.T) {
	cuerpo := func(codigoAccesoID int) map[string]any {
		return map[string]any{
			"padre_id": 3, "hijo_id": 5, "aseado": true, "reporte_golpe": false,
			"observaciones": "", "codigo_acceso_id": codigoAccesoID,
		}
	}

	t.Run("código vigente y autoriza al hijo -> inserta con la nota y codigo_acceso_id", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock: %v", err)
		}
		defer mockDB.Close()
		srv := nuevoServerDePrueba(mockDB)

		vence := time.Now().Add(48 * time.Hour)
		mock.ExpectQuery("SELECT ca.persona_autorizada(.|\n)*FROM codigos_acceso").
			WithArgs(9, 5, 1, 3).
			WillReturnRows(sqlmock.NewRows([]string{"persona_autorizada", "expira_en", "revocado_en", "exists"}).
				AddRow("Tía Lupita", vence, nil, true))
		mock.ExpectQuery("SELECT tipo_movimiento(.|\n)*FROM asistencia").
			WithArgs(5, 1).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectExec("INSERT INTO asistencia").
			WithArgs(3, 5, true, false, "Vía código de acceso — autorizado para: Tía Lupita", "ENTRADA", 1, 1, 9).
			WillReturnResult(sqlmock.NewResult(1, 1))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/confirmar-asistencia", cuerpo(9))
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectativas de sqlmock no cumplidas: %v", err)
		}
	})

	t.Run("código ya revocado -> 400, nunca llega a insertar la asistencia", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock: %v", err)
		}
		defer mockDB.Close()
		srv := nuevoServerDePrueba(mockDB)

		vence := time.Now().Add(48 * time.Hour)
		revocado := time.Now().Add(-1 * time.Minute)
		mock.ExpectQuery("SELECT ca.persona_autorizada(.|\n)*FROM codigos_acceso").
			WithArgs(9, 5, 1, 3).
			WillReturnRows(sqlmock.NewRows([]string{"persona_autorizada", "expira_en", "revocado_en", "exists"}).
				AddRow("Tía Lupita", vence, revocado, true))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/confirmar-asistencia", cuerpo(9))
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectativas de sqlmock no cumplidas: %v", err)
		}
	})
}
