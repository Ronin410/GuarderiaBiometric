package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func confirmarAsistenciaRequest(t *testing.T, hijoID int) *http.Request {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"padre_id": 1, "hijo_id": hijoID, "aseado": true, "reporte_golpe": false, "observaciones": "",
	})
	req := httptest.NewRequest(http.MethodPost, "/confirmar-asistencia", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+generarTokenPrueba(t, "staff", time.Hour))
	return req
}

func TestConfirmarAsistencia(t *testing.T) {
	jwtKey = []byte("clave-de-prueba-solo-para-tests")

	t.Run("sin movimiento previo hoy -> ENTRADA", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock: %v", err)
		}
		defer mockDB.Close()
		db = mockDB

		mock.ExpectQuery("SELECT tipo_movimiento(.|\n)*FROM asistencia").
			WithArgs(42, 1).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectExec("INSERT INTO asistencia").
			WithArgs(1, 42, true, false, "", "ENTRADA", 1).
			WillReturnResult(sqlmock.NewResult(1, 1))

		r := setupRouter()
		w := httptest.NewRecorder()
		r.ServeHTTP(w, confirmarAsistenciaRequest(t, 42))

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["tipo"] != "ENTRADA" {
			t.Errorf("tipo = %v; esperado ENTRADA", resp["tipo"])
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectativas de sqlmock no cumplidas: %v", err)
		}
	})

	t.Run("última entrada hoy -> SALIDA", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock: %v", err)
		}
		defer mockDB.Close()
		db = mockDB

		mock.ExpectQuery("SELECT tipo_movimiento(.|\n)*FROM asistencia").
			WithArgs(42, 1).
			WillReturnRows(sqlmock.NewRows([]string{"tipo_movimiento"}).AddRow("ENTRADA"))
		mock.ExpectExec("INSERT INTO asistencia").
			WithArgs(1, 42, true, false, "", "SALIDA", 1).
			WillReturnResult(sqlmock.NewResult(1, 1))

		r := setupRouter()
		w := httptest.NewRecorder()
		r.ServeHTTP(w, confirmarAsistenciaRequest(t, 42))

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["tipo"] != "SALIDA" {
			t.Errorf("tipo = %v; esperado SALIDA", resp["tipo"])
		}
	})

	t.Run("JSON inválido -> 400", func(t *testing.T) {
		mockDB, _, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock: %v", err)
		}
		defer mockDB.Close()
		db = mockDB

		r := setupRouter()
		req := httptest.NewRequest(http.MethodPost, "/confirmar-asistencia", bytes.NewReader([]byte("no-es-json")))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+generarTokenPrueba(t, "staff", time.Hour))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("sin token -> 401, no llega a tocar la base de datos", func(t *testing.T) {
		mockDB, _, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock: %v", err)
		}
		defer mockDB.Close()
		db = mockDB

		r := setupRouter()
		body, _ := json.Marshal(map[string]any{"hijo_id": 42})
		req := httptest.NewRequest(http.MethodPost, "/confirmar-asistencia", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("código = %d; esperado 401 (body: %s)", w.Code, w.Body.String())
		}
	})
}

// TestIdentificarRequiereImagen cubre la validación de entrada de /identificar
// que no depende de Rekognition (rekClient es un *rekognition.Client concreto del
// SDK de AWS, sin interfaz propia — mockearlo de verdad para probar el camino feliz
// de la identificación facial requiere la inyección de dependencias que llega con
// la Fase 2 de este mismo lote; por ahora se prueba solo lo que es seguro probar
// sin red real: la validación de entrada y el 401 sin token.
func TestIdentificarRequiereImagen(t *testing.T) {
	jwtKey = []byte("clave-de-prueba-solo-para-tests")
	mockDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer mockDB.Close()
	db = mockDB

	r := setupRouter()

	t.Run("sin token -> 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/identificar", bytes.NewReader([]byte(`{}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("código = %d; esperado 401 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("JSON inválido -> 400, nunca llega a Rekognition", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/identificar", bytes.NewReader([]byte("no-es-json")))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+generarTokenPrueba(t, "staff", time.Hour))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})
}
