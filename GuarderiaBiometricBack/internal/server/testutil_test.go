package server

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"biometrico/internal/middleware"
)

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

// nuevoRouterDePrueba monta un *gin.Engine real sobre el Server dado, igual
// que hace cmd/server/main.go, pero sin arrancar un puerto HTTP — para
// ejercitarlo con httptest.
func nuevoRouterDePrueba(s *Server) *gin.Engine {
	r := gin.New()
	s.RegisterRoutes(r)
	return r
}
