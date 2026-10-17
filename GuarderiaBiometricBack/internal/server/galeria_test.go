package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

// Nota: no se prueba el camino con fotos reales -- obtenerGaleria llama
// firmarURLFoto, que necesita un cliente S3 real (mismo motivo por el que
// soporte_test.go solo prueba extraerKeyS3, la parte pura, y no
// firmarURLFoto). Aquí se cubre el enrutamiento/permisos y el caso sin
// fotos (0 filas nunca llega a firmar nada); el camino con fotos reales se
// verifica en vivo con Postgres real.

func TestGaleriaStaff(t *testing.T) {
	t.Run("niño de otra guardería -> 404", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("9", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/hijos/9/galeria", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("código = %d; esperado 404 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("papá no puede usar la ruta de staff -> 403", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/hijos/5/galeria", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("niño sin fotos -> 200 con lista vacía", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("5", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery("SELECT f.id, f.url, s.fecha FROM fotos_seguimiento").
			WithArgs("5").
			WillReturnRows(sqlmock.NewRows([]string{"id", "url", "fecha"}))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/hijos/5/galeria", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		var fotos []FotoGaleria
		if err := json.Unmarshal(w.Body.Bytes(), &fotos); err != nil {
			t.Fatalf("respuesta no es JSON válido: %v", err)
		}
		if len(fotos) != 0 {
			t.Fatalf("se esperaba una galería vacía, se recibió: %+v", fotos)
		}
	})
}

func TestGaleriaPadre(t *testing.T) {
	t.Run("niño ajeno -> 403", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("9", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/padre/hijos/9/galeria", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("papá ve la galería (vacía) de su hijo -> 200", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("5", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery("SELECT f.id, f.url, s.fecha FROM fotos_seguimiento").
			WithArgs("5").
			WillReturnRows(sqlmock.NewRows([]string{"id", "url", "fecha"}))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/padre/hijos/5/galeria", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})
}
