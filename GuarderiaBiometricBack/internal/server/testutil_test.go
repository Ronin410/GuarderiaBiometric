package server

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"biometrico/internal/middleware"
)

// csrfTokenPrueba es el valor fijo de CSRF usado en pruebas — no necesita
// ser aleatorio como en producción, solo tiene que coincidir entre la
// cookie y el header, que es lo único que middleware.Auth verifica.
const csrfTokenPrueba = "csrf-de-prueba"

func init() {
	gin.SetMode(gin.TestMode)
}

// generarTokenPrueba firma un JWT con la misma forma que /login, para poder
// probar rutas protegidas sin pasar por la base de datos.
func generarTokenPrueba(t *testing.T, jwtKey []byte, rol string, expiraEn time.Duration) string {
	t.Helper()
	claims := &middleware.Claims{
		UserID:      1,
		GuarderiaID: 1,
		Rol:         rol,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiraEn)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	firmado, err := token.SignedString(jwtKey)
	if err != nil {
		t.Fatalf("no se pudo firmar el token de prueba: %v", err)
	}
	return firmado
}

// autenticarRequestPrueba agrega a un *http.Request las cookies que
// middleware.Auth exige (JWT + CSRF) y, si el método modifica datos, el
// header X-CSRF-Token — mismo patrón "double-submit cookie" que usa
// axiosConfig.js en el frontend.
func autenticarRequestPrueba(t *testing.T, req *http.Request, jwtKey []byte, rol string, expiraEn time.Duration) {
	t.Helper()
	req.AddCookie(&http.Cookie{Name: middleware.CookieToken, Value: generarTokenPrueba(t, jwtKey, rol, expiraEn)})
	if req.Method != http.MethodGet && req.Method != http.MethodHead && req.Method != http.MethodOptions {
		req.AddCookie(&http.Cookie{Name: middleware.CookieCSRF, Value: csrfTokenPrueba})
		req.Header.Set("X-CSRF-Token", csrfTokenPrueba)
	}
}

// nuevoRouterDePrueba monta un *gin.Engine real sobre el Server dado, igual
// que hace cmd/server/main.go, pero sin arrancar un puerto HTTP — para
// ejercitarlo con httptest.
func nuevoRouterDePrueba(s *Server) *gin.Engine {
	r := gin.New()
	s.RegisterRoutes(r)
	return r
}
