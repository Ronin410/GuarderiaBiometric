package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
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

	t.Run("sin cookie de sesión -> 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protegido", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("código = %d; esperado 401 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("token con firma de otra clave -> 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protegido", nil)
		req.AddCookie(&http.Cookie{Name: CookieToken, Value: generarToken(t, []byte("otra-clave-distinta"), "admin", time.Hour)})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("código = %d; esperado 401 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("token válido deja pasar", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protegido", nil)
		req.AddCookie(&http.Cookie{Name: CookieToken, Value: generarToken(t, jwtKey, "admin", time.Hour)})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("token expirado -> 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protegido", nil)
		req.AddCookie(&http.Cookie{Name: CookieToken, Value: generarToken(t, jwtKey, "admin", -time.Hour)})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("código = %d; esperado 401 para un token ya expirado (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("POST sin header X-CSRF-Token -> 403", func(t *testing.T) {
		rPost := gin.New()
		rPost.POST("/protegido", Auth(jwtKey), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})
		req := httptest.NewRequest(http.MethodPost, "/protegido", nil)
		req.AddCookie(&http.Cookie{Name: CookieToken, Value: generarToken(t, jwtKey, "admin", time.Hour)})
		req.AddCookie(&http.Cookie{Name: CookieCSRF, Value: "csrf-de-prueba"})
		w := httptest.NewRecorder()
		rPost.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Errorf("código = %d; esperado 403 sin header X-CSRF-Token (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("POST con cookie y header CSRF coincidentes -> 200", func(t *testing.T) {
		rPost := gin.New()
		rPost.POST("/protegido", Auth(jwtKey), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})
		req := httptest.NewRequest(http.MethodPost, "/protegido", nil)
		req.AddCookie(&http.Cookie{Name: CookieToken, Value: generarToken(t, jwtKey, "admin", time.Hour)})
		req.AddCookie(&http.Cookie{Name: CookieCSRF, Value: "csrf-de-prueba"})
		req.Header.Set("X-CSRF-Token", "csrf-de-prueba")
		w := httptest.NewRecorder()
		rPost.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("código = %d; esperado 200 con CSRF coincidente (body: %s)", w.Code, w.Body.String())
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

func TestRequireArea(t *testing.T) {
	// Simula lo que hace Auth(): pone "rol" siempre, y "permisos" solo
	// cuando la cuenta trae permisos personalizados (el caso "sin
	// personalizar" se representa con la ausencia de la clave, no con nil).
	r := gin.New()
	r.GET("/pagos-o-lo-que-sea",
		func(c *gin.Context) {
			c.Set("rol", c.GetHeader("X-Rol-Prueba"))
			if permisosCSV := c.GetHeader("X-Permisos-Prueba"); permisosCSV != "" || c.GetHeader("X-Personalizado-Prueba") == "1" {
				var permisos []string
				if permisosCSV != "" {
					permisos = strings.Split(permisosCSV, ",")
				}
				c.Set("permisos", permisos)
			}
			c.Next()
		},
		RequireArea("pagos"),
		func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) },
	)

	casos := []struct {
		nombre         string
		rol            string
		personalizado  bool
		permisos       string
		codigoEsperado int
	}{
		{"admin siempre pasa", "admin", false, "", http.StatusOK},
		{"admin con permisos personalizados (irrelevante) igual pasa", "admin", true, "menu", http.StatusOK},
		{"staff sin personalizar -> acceso completo (legacy)", "staff", false, "", http.StatusOK},
		{"staff personalizado CON el área -> pasa", "staff", true, "bitacora,pagos", http.StatusOK},
		{"staff personalizado SIN el área -> 403", "staff", true, "bitacora,menu", http.StatusForbidden},
		{"staff personalizado a lista vacía -> 403", "staff", true, "", http.StatusForbidden},
		{"papa nunca pasa, tenga o no personalizado", "papa", false, "", http.StatusForbidden},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/pagos-o-lo-que-sea", nil)
			req.Header.Set("X-Rol-Prueba", c.rol)
			if c.personalizado {
				req.Header.Set("X-Personalizado-Prueba", "1")
				req.Header.Set("X-Permisos-Prueba", c.permisos)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != c.codigoEsperado {
				t.Errorf("código = %d; esperado %d (body: %s)", w.Code, c.codigoEsperado, w.Body.String())
			}
		})
	}
}

func TestRequireAdmin(t *testing.T) {
	r := gin.New()
	r.GET("/solo-admin",
		func(c *gin.Context) { c.Set("rol", c.GetHeader("X-Rol-Prueba")); c.Next() },
		RequireAdmin(),
		func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) },
	)

	casos := []struct {
		rol            string
		codigoEsperado int
	}{
		{"admin", http.StatusOK},
		{"staff", http.StatusForbidden},
		{"papa", http.StatusForbidden},
		{"", http.StatusForbidden},
	}

	for _, c := range casos {
		t.Run("rol="+c.rol, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/solo-admin", nil)
			req.Header.Set("X-Rol-Prueba", c.rol)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != c.codigoEsperado {
				t.Errorf("rol %q: código = %d; esperado %d", c.rol, w.Code, c.codigoEsperado)
			}
		})
	}
}
