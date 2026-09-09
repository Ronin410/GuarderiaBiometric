package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func nuevoServidorDePruebaPushPlataforma(t *testing.T) (*Server, sqlmock.Sqlmock) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { mockDB.Close() })

	srv := New()
	srv.DB = mockDB
	srv.DBAuth = mockDB
	srv.PlatformAdminKey = "clave-de-plataforma-de-prueba"
	return srv, mock
}

func suscripcionPruebaBody() map[string]any {
	return map[string]any{
		"endpoint": "https://fcm.googleapis.com/fcm/send/abc123",
		"keys": map[string]string{
			"p256dh": "clave-p256dh",
			"auth":   "clave-auth",
		},
	}
}

func TestSuscribirPushPlataforma(t *testing.T) {
	t.Run("sin llave -> 401", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaPushPlataforma(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/plataforma/push/suscribir", suscripcionPruebaBody())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("código = %d; esperado 401 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("sin VAPID configurado -> 503", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaPushPlataforma(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/plataforma/push/suscribir", suscripcionPruebaBody())
		req.Header.Set("X-Platform-Key", "clave-de-plataforma-de-prueba")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("código = %d; esperado 503 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("con llave y VAPID -> guarda la suscripción", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaPushPlataforma(t)
		srv.VapidPublicKey = "clave-publica-de-prueba"
		srv.VapidPrivateKey = "clave-privada-de-prueba"
		mock.ExpectExec("INSERT INTO push_suscripciones_plataforma").
			WithArgs("https://fcm.googleapis.com/fcm/send/abc123", "clave-p256dh", "clave-auth").
			WillReturnResult(sqlmock.NewResult(1, 1))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/plataforma/push/suscribir", suscripcionPruebaBody())
		req.Header.Set("X-Platform-Key", "clave-de-plataforma-de-prueba")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("suscripción incompleta -> 400", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaPushPlataforma(t)
		srv.VapidPublicKey = "clave-publica-de-prueba"
		srv.VapidPrivateKey = "clave-privada-de-prueba"

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/plataforma/push/suscribir", map[string]any{"endpoint": ""})
		req.Header.Set("X-Platform-Key", "clave-de-plataforma-de-prueba")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})
}

func TestEliminarSuscripcionPushPlataforma(t *testing.T) {
	srv, mock := nuevoServidorDePruebaPushPlataforma(t)
	mock.ExpectExec("DELETE FROM push_suscripciones_plataforma WHERE endpoint").
		WithArgs("https://fcm.googleapis.com/fcm/send/abc123").
		WillReturnResult(sqlmock.NewResult(0, 1))

	r := nuevoRouterDePrueba(srv)
	req := jsonRequest(http.MethodDelete, "/plataforma/push/suscribir", map[string]string{"endpoint": "https://fcm.googleapis.com/fcm/send/abc123"})
	req.Header.Set("X-Platform-Key", "clave-de-plataforma-de-prueba")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
	}
}
