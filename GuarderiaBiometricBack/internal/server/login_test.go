package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"golang.org/x/crypto/bcrypt"

	"biometrico/internal/middleware"
)

func loginRequest(username, password string) *http.Request {
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestLogin(t *testing.T) {
	hashCorrecto, err := bcrypt.GenerateFromPassword([]byte("Correcta123!"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("no se pudo generar el hash de prueba: %v", err)
	}

	columnas := []string{"id", "guarderia_id", "password_hash", "rol", "pin_admin", "nombre", "slug", "activo", "permisos"}

	t.Run("credenciales correctas -> 200 con token", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock: %v", err)
		}
		defer mockDB.Close()

		srv := New()
		srv.DBAuth = mockDB
		srv.JWTKey = []byte("clave-de-prueba-solo-para-tests")

		mock.ExpectQuery("SELECT(.|\n)*FROM usuarios(.|\n)*WHERE u.username = \\$1").
			WithArgs("admin_demo").
			WillReturnRows(sqlmock.NewRows(columnas).
				AddRow(1, 1, string(hashCorrecto), "admin", "1234", "Guardería Demo", "demo", true, nil))

		r := nuevoRouterDePrueba(srv)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, loginRequest("admin_demo", "Correcta123!"))

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("respuesta no es JSON válido: %v", err)
		}
		if resp["token"] != nil {
			t.Errorf("el JWT ya NO debe ir en el body (vive en una cookie httpOnly), se encontró: %v", resp["token"])
		}
		if resp["pin_admin"] != nil {
			t.Errorf("el pin_admin NUNCA debe exponerse en la respuesta de /login, se encontró: %v", resp["pin_admin"])
		}

		cookies := w.Result().Cookies()
		var tieneCookieToken, tieneCookieCSRF bool
		for _, ck := range cookies {
			if ck.Name == middleware.CookieToken {
				tieneCookieToken = true
				if !ck.HttpOnly {
					t.Errorf("la cookie %s debe ser httpOnly", middleware.CookieToken)
				}
				if ck.Value == "" {
					t.Errorf("la cookie %s no debe estar vacía", middleware.CookieToken)
				}
			}
			if ck.Name == middleware.CookieCSRF {
				tieneCookieCSRF = true
				if ck.HttpOnly {
					t.Errorf("la cookie %s NO debe ser httpOnly (el frontend la tiene que leer)", middleware.CookieCSRF)
				}
			}
		}
		if !tieneCookieToken {
			t.Errorf("se esperaba un Set-Cookie para %s, cookies recibidas: %v", middleware.CookieToken, cookies)
		}
		if !tieneCookieCSRF {
			t.Errorf("se esperaba un Set-Cookie para %s, cookies recibidas: %v", middleware.CookieCSRF, cookies)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectativas de sqlmock no cumplidas: %v", err)
		}
	})

	t.Run("contraseña incorrecta -> 401", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock: %v", err)
		}
		defer mockDB.Close()

		srv := New()
		srv.DBAuth = mockDB
		srv.JWTKey = []byte("clave-de-prueba-solo-para-tests")

		mock.ExpectQuery("SELECT(.|\n)*FROM usuarios(.|\n)*WHERE u.username = \\$1").
			WithArgs("admin_demo").
			WillReturnRows(sqlmock.NewRows(columnas).
				AddRow(1, 1, string(hashCorrecto), "admin", "1234", "Guardería Demo", "demo", true, nil))

		r := nuevoRouterDePrueba(srv)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, loginRequest("admin_demo", "esta-no-es-la-contraseña"))

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("código = %d; esperado 401 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("cuenta desactivada -> 401", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock: %v", err)
		}
		defer mockDB.Close()

		srv := New()
		srv.DBAuth = mockDB
		srv.JWTKey = []byte("clave-de-prueba-solo-para-tests")

		mock.ExpectQuery("SELECT(.|\n)*FROM usuarios(.|\n)*WHERE u.username = \\$1").
			WithArgs("staff_baja").
			WillReturnRows(sqlmock.NewRows(columnas).
				AddRow(2, 1, string(hashCorrecto), "staff", "1234", "Guardería Demo", "demo", false, nil))

		r := nuevoRouterDePrueba(srv)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, loginRequest("staff_baja", "Correcta123!"))

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("código = %d; esperado 401 (body: %s)", w.Code, w.Body.String())
		}
		cookies := w.Result().Cookies()
		for _, ck := range cookies {
			if ck.Name == middleware.CookieToken && ck.Value != "" {
				t.Errorf("una cuenta desactivada no debe recibir cookie de sesión, se encontró: %v", ck)
			}
		}
	})

	t.Run("usuario inexistente -> 401", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock: %v", err)
		}
		defer mockDB.Close()

		srv := New()
		srv.DBAuth = mockDB
		srv.JWTKey = []byte("clave-de-prueba-solo-para-tests")

		mock.ExpectQuery("SELECT(.|\n)*FROM usuarios(.|\n)*WHERE u.username = \\$1").
			WithArgs("no_existe").
			WillReturnError(sql.ErrNoRows)

		r := nuevoRouterDePrueba(srv)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, loginRequest("no_existe", "cualquiera"))

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("código = %d; esperado 401 (body: %s)", w.Code, w.Body.String())
		}
	})
}
