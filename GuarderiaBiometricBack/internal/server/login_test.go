package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"golang.org/x/crypto/bcrypt"

	"biometrico/internal/middleware"
)

func loginRequest(username, password string) *http.Request {
	return loginRequestConTipo(username, password, "")
}

// loginRequestConTipo -- igual que loginRequest, pero mandando el selector
// "Staff/Admin" vs "Soy Papá" de la pantalla de login (campo "tipo").
func loginRequestConTipo(username, password, tipo string) *http.Request {
	body, _ := json.Marshal(map[string]string{"username": username, "password": password, "tipo": tipo})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestLogin(t *testing.T) {
	hashCorrecto, err := bcrypt.GenerateFromPassword([]byte("Correcta123!"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("no se pudo generar el hash de prueba: %v", err)
	}

	columnas := []string{"id", "guarderia_id", "password_hash", "rol", "pin_admin", "nombre", "slug", "activo", "permisos", "bloqueada"}

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
				AddRow(1, 1, string(hashCorrecto), "admin", "1234", "Guardería Demo", "demo", true, nil, false))

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
				AddRow(1, 1, string(hashCorrecto), "admin", "1234", "Guardería Demo", "demo", true, nil, false))

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
				AddRow(2, 1, string(hashCorrecto), "staff", "1234", "Guardería Demo", "demo", false, nil, false))

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

	// "Aunque deje la opción de staff/admin, si pongo el usuario y
	// contraseña del papá me deja iniciar sesión como papá" -- el selector
	// de la pantalla de login debe rechazar credenciales válidas de un rol
	// que no coincide con la pestaña elegida, en vez de dejar entrar de
	// todos modos.
	t.Run("credenciales de papá con tipo=staff -> 401", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock: %v", err)
		}
		defer mockDB.Close()

		srv := New()
		srv.DBAuth = mockDB
		srv.JWTKey = []byte("clave-de-prueba-solo-para-tests")

		mock.ExpectQuery("SELECT(.|\n)*FROM usuarios(.|\n)*WHERE u.username = \\$1").
			WithArgs("papa_demo").
			WillReturnRows(sqlmock.NewRows(columnas).
				AddRow(3, 1, string(hashCorrecto), "papa", "1234", "Guardería Demo", "demo", true, nil, false))

		r := nuevoRouterDePrueba(srv)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, loginRequestConTipo("papa_demo", "Correcta123!", "staff"))

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("código = %d; esperado 401 (body: %s)", w.Code, w.Body.String())
		}
		for _, ck := range w.Result().Cookies() {
			if ck.Name == middleware.CookieToken && ck.Value != "" {
				t.Errorf("no debe emitirse cookie de sesión cuando el tipo no coincide con el rol, se encontró: %v", ck)
			}
		}
	})

	t.Run("credenciales de admin con tipo=papa -> 401", func(t *testing.T) {
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
				AddRow(1, 1, string(hashCorrecto), "admin", "1234", "Guardería Demo", "demo", true, nil, false))

		r := nuevoRouterDePrueba(srv)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, loginRequestConTipo("admin_demo", "Correcta123!", "papa"))

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("código = %d; esperado 401 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("tipo coincide con el rol -> 200", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock: %v", err)
		}
		defer mockDB.Close()

		srv := New()
		srv.DBAuth = mockDB
		srv.JWTKey = []byte("clave-de-prueba-solo-para-tests")

		mock.ExpectQuery("SELECT(.|\n)*FROM usuarios(.|\n)*WHERE u.username = \\$1").
			WithArgs("staff_demo").
			WillReturnRows(sqlmock.NewRows(columnas).
				AddRow(2, 1, string(hashCorrecto), "staff", "1234", "Guardería Demo", "demo", true, nil, false))

		r := nuevoRouterDePrueba(srv)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, loginRequestConTipo("staff_demo", "Correcta123!", "staff"))

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("guardería bloqueada -> 403", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock: %v", err)
		}
		defer mockDB.Close()

		srv := New()
		srv.DBAuth = mockDB
		srv.JWTKey = []byte("clave-de-prueba-solo-para-tests")

		mock.ExpectQuery("SELECT(.|\n)*FROM usuarios(.|\n)*WHERE u.username = \\$1").
			WithArgs("admin_moroso").
			WillReturnRows(sqlmock.NewRows(columnas).
				AddRow(1, 1, string(hashCorrecto), "admin", "1234", "Guardería Morosa", "morosa", true, nil, true))

		r := nuevoRouterDePrueba(srv)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, loginRequest("admin_moroso", "Correcta123!"))

		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
		for _, ck := range w.Result().Cookies() {
			if ck.Name == middleware.CookieToken && ck.Value != "" {
				t.Errorf("una guardería bloqueada no debe recibir cookie de sesión, se encontró: %v", ck)
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

// TestMe cubre la sesión deslizante de /me: los papás reciben un JWT
// renovado con la ventana completa de 90 días en cada llamada (que ocurre
// cada vez que abren la app), mientras que admin/staff conservan su sesión
// corta de siempre sin renovación automática.
func TestMe(t *testing.T) {
	columnasMe := []string{"username", "nombre", "slug", "bloqueada"}

	t.Run("papá -> se renueva a ~90 días y llega cookie nueva", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock: %v", err)
		}
		defer mockDB.Close()

		srv := New()
		srv.DBAuth = mockDB
		srv.JWTKey = []byte("clave-de-prueba-solo-para-tests")

		mock.ExpectQuery("SELECT(.|\n)*FROM usuarios(.|\n)*WHERE u.id = \\$1").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows(columnasMe).AddRow("papa_demo", "Guardería Demo", "demo", false))

		r := nuevoRouterDePrueba(srv)
		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		// El token original casi está por expirar (1 hora) -- justo el caso
		// que la sesión deslizante debe evitar que le explote en la cara.
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", 1*time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("respuesta no es JSON válido: %v", err)
		}

		expiraEn, _ := resp["expires_at"].(float64)
		faltan := time.Until(time.Unix(int64(expiraEn), 0))
		if faltan < 89*24*time.Hour {
			t.Errorf("expires_at debería reflejar la renovación a ~90 días, pero faltan solo %v", faltan)
		}

		var cookieRenovada *http.Cookie
		for _, ck := range w.Result().Cookies() {
			if ck.Name == middleware.CookieToken {
				cookieRenovada = ck
			}
		}
		if cookieRenovada == nil {
			t.Fatalf("se esperaba un Set-Cookie nuevo para %s tras renovar la sesión del papá", middleware.CookieToken)
		}
		if cookieRenovada.MaxAge < 89*24*60*60 {
			t.Errorf("MaxAge de la cookie renovada = %d; esperado ~90 días", cookieRenovada.MaxAge)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectativas de sqlmock no cumplidas: %v", err)
		}
	})

	t.Run("admin -> NO se renueva, conserva su sesión corta", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock: %v", err)
		}
		defer mockDB.Close()

		srv := New()
		srv.DBAuth = mockDB
		srv.JWTKey = []byte("clave-de-prueba-solo-para-tests")

		mock.ExpectQuery("SELECT(.|\n)*FROM usuarios(.|\n)*WHERE u.id = \\$1").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows(columnasMe).AddRow("admin_demo", "Guardería Demo", "demo", false))

		r := nuevoRouterDePrueba(srv)
		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "admin", 1*time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("respuesta no es JSON válido: %v", err)
		}

		expiraEn, _ := resp["expires_at"].(float64)
		faltan := time.Until(time.Unix(int64(expiraEn), 0))
		if faltan > 2*time.Hour {
			t.Errorf("admin/staff no debe renovarse: faltan %v para expirar, se esperaba ~1 hora (la del token original)", faltan)
		}

		for _, ck := range w.Result().Cookies() {
			if ck.Name == middleware.CookieToken {
				t.Errorf("admin/staff no debe recibir una cookie de sesión nueva en /me, se encontró: %v", ck)
			}
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectativas de sqlmock no cumplidas: %v", err)
		}
	})

	// "Bloquear el acceso a una guardería" también corta a alguien que ya
	// tenía sesión abierta: la próxima vez que la app llame /me (recarga,
	// reapertura), aunque el JWT siga siendo válido.
	t.Run("guardería bloqueada -> 403, ni siquiera intenta renovar", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock: %v", err)
		}
		defer mockDB.Close()

		srv := New()
		srv.DBAuth = mockDB
		srv.JWTKey = []byte("clave-de-prueba-solo-para-tests")

		mock.ExpectQuery("SELECT(.|\n)*FROM usuarios(.|\n)*WHERE u.id = \\$1").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows(columnasMe).AddRow("papa_demo", "Guardería Morosa", "morosa", true))

		r := nuevoRouterDePrueba(srv)
		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", 1*time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
		for _, ck := range w.Result().Cookies() {
			if ck.Name == middleware.CookieToken {
				t.Errorf("no debe emitirse una cookie renovada para una guardería bloqueada, se encontró: %v", ck)
			}
		}
	})
}
