package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func generarToken(t *testing.T, jwtKey []byte, rol string, expiraEn time.Duration) string {
	t.Helper()
	claims := &Claims{
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

func TestAuth(t *testing.T) {
	jwtKey := []byte("clave-de-prueba-solo-para-tests")

	r := gin.New()
	r.GET("/protegido", Auth(jwtKey), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	t.Run("sin cabecera Authorization -> 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protegido", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("código = %d; esperado 401 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("token con firma de otra clave -> 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protegido", nil)
		req.Header.Set("Authorization", "Bearer "+generarToken(t, []byte("otra-clave-distinta"), "admin", time.Hour))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("código = %d; esperado 401 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("token válido deja pasar", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protegido", nil)
		req.Header.Set("Authorization", "Bearer "+generarToken(t, jwtKey, "admin", time.Hour))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("token expirado -> 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protegido", nil)
		req.Header.Set("Authorization", "Bearer "+generarToken(t, jwtKey, "admin", -time.Hour))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("código = %d; esperado 401 para un token ya expirado (body: %s)", w.Code, w.Body.String())
		}
	})
}

func TestRequireStaff(t *testing.T) {
	r := gin.New()
	r.GET("/solo-staff",
		func(c *gin.Context) { c.Set("rol", c.GetHeader("X-Rol-Prueba")); c.Next() },
		RequireStaff(),
		func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) },
	)

	casos := []struct {
		rol            string
		codigoEsperado int
	}{
		{"admin", http.StatusOK},
		{"staff", http.StatusOK},
		{"papa", http.StatusForbidden},
		{"", http.StatusForbidden},
	}

	for _, c := range casos {
		t.Run("rol="+c.rol, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/solo-staff", nil)
			req.Header.Set("X-Rol-Prueba", c.rol)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != c.codigoEsperado {
				t.Errorf("rol %q: código = %d; esperado %d", c.rol, w.Code, c.codigoEsperado)
			}
		})
	}
}
