package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestRegistrarTokenExpo(t *testing.T) {
	t.Run("papá -> guarda con padre_id", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectExec("INSERT INTO push_tokens_expo").
			WithArgs(1, nil, 1, "ExponentPushToken[abc123]").
			WillReturnResult(sqlmock.NewResult(1, 1))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/push/expo/registrar", map[string]string{"token": "ExponentPushToken[abc123]"})
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("staff -> guarda con personal_id", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectExec("INSERT INTO push_tokens_expo").
			WithArgs(nil, 1, 1, "ExponentPushToken[abc123]").
			WillReturnResult(sqlmock.NewResult(1, 1))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/push/expo/registrar", map[string]string{"token": "ExponentPushToken[abc123]"})
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("sin token -> 400", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/push/expo/registrar", map[string]string{"token": ""})
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("sin sesión -> 401", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/push/expo/registrar", map[string]string{"token": "ExponentPushToken[abc123]"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("código = %d; esperado 401 (body: %s)", w.Code, w.Body.String())
		}
	})
}

func TestEliminarTokenExpo(t *testing.T) {
	srv, mock := nuevoServidorDePruebaConDB(t)
	mock.ExpectExec("DELETE FROM push_tokens_expo WHERE token").
		WithArgs("ExponentPushToken[abc123]").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := nuevoRouterDePrueba(srv)
	req := jsonRequest(http.MethodDelete, "/push/expo/registrar", map[string]string{"token": "ExponentPushToken[abc123]"})
	autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
	}
}
