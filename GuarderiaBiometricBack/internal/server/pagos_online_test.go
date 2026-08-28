package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stripe/stripe-go/v82/webhook"
)

// Nota general: session.New() de verdad llama a la API de Stripe por HTTP,
// así que ningún test de aquí ejercita el camino feliz de
// handleCrearCheckoutColegiatura (crear la Checkout Session) -- solo las
// validaciones que se resuelven ANTES de llamar a Stripe. Es el mismo
// criterio que ya usa galeria_test.go/soporte_test.go con firmarURLFoto: lo
// que de verdad habla con un servicio externo se prueba en vivo, no con
// sqlmock. handleWebhookStripe sí se prueba completo porque verificar la
// firma es puro cómputo local (HMAC), sin red de por medio.

func TestConfigPagosOnline(t *testing.T) {
	srv, _ := nuevoServidorDePruebaConDB(t)
	// StripeSecretKey vacía (valor por defecto de un Server nuevo en las
	// pruebas) -- exactamente el estado "sin activar" que pide este alcance.
	r := nuevoRouterDePrueba(srv)

	req := jsonRequest(http.MethodGet, "/pagos-online/config", nil)
	autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v", err)
	}
	if habilitado, _ := resp["habilitado"].(bool); habilitado {
		t.Errorf("habilitado = true; esperado false sin STRIPE_SECRET_KEY configurada")
	}
}

func TestCrearCheckoutColegiatura(t *testing.T) {
	t.Run("Stripe deshabilitado -> 501", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)

		req := jsonRequest(http.MethodPost, "/padre/pagos-online/checkout", map[string]any{"hijo_id": "3", "periodo": "2026-08"})
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotImplemented {
			t.Fatalf("código = %d; esperado 501 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("niño ajeno -> 403", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		srv.StripeSecretKey = "sk_test_clave_de_prueba" // habilita la ruta sin llamar a Stripe -- se corta antes

		mock.ExpectQuery("SELECT EXISTS").WithArgs("99", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/padre/pagos-online/checkout", map[string]any{"hijo_id": "99", "periodo": "2026-08"})
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("sin saldo pendiente -> 400, sin llegar a Stripe", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		srv.StripeSecretKey = "sk_test_clave_de_prueba"

		mock.ExpectQuery("SELECT EXISTS").WithArgs("3", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery("SELECT(.|\n)*FROM hijos(.|\n)*LEFT JOIN pagos").
			WithArgs("3", "2026-08", 1).
			WillReturnRows(sqlmock.NewRows([]string{"nombre_niño", "colegiatura_mensual", "total"}).
				AddRow("Valentina Cruz", 1500.0, 1500.0)) // ya pagado por completo

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/padre/pagos-online/checkout", map[string]any{"hijo_id": "3", "periodo": "2026-08"})
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	// "Quiero que el papá pueda seleccionar cuánto puede pagar" -- puede
	// pedir menos del saldo (pago parcial), pero no más.
	t.Run("monto mayor al saldo pendiente -> 400, sin llegar a Stripe", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		srv.StripeSecretKey = "sk_test_clave_de_prueba"

		mock.ExpectQuery("SELECT EXISTS").WithArgs("3", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery("SELECT(.|\n)*FROM hijos(.|\n)*LEFT JOIN pagos").
			WithArgs("3", "2026-08", 1).
			WillReturnRows(sqlmock.NewRows([]string{"nombre_niño", "colegiatura_mensual", "total"}).
				AddRow("Valentina Cruz", 1500.0, 0.0)) // saldo pendiente: 1500

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/padre/pagos-online/checkout", map[string]any{"hijo_id": "3", "periodo": "2026-08", "monto": 2000})
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("monto por debajo del mínimo ($10) -> 400, sin llegar a Stripe", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		srv.StripeSecretKey = "sk_test_clave_de_prueba"

		mock.ExpectQuery("SELECT EXISTS").WithArgs("3", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery("SELECT(.|\n)*FROM hijos(.|\n)*LEFT JOIN pagos").
			WithArgs("3", "2026-08", 1).
			WillReturnRows(sqlmock.NewRows([]string{"nombre_niño", "colegiatura_mensual", "total"}).
				AddRow("Valentina Cruz", 1500.0, 0.0))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/padre/pagos-online/checkout", map[string]any{"hijo_id": "3", "periodo": "2026-08", "monto": 5})
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})
}

func TestWebhookStripe(t *testing.T) {
	const secreto = "whsec_de_prueba"

	firmarYEnviar := func(t *testing.T, srv *Server, payload []byte) *httptest.ResponseRecorder {
		t.Helper()
		firmado := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{
			Payload: payload, Secret: secreto, Timestamp: time.Now(),
		})
		req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", bytes.NewReader(payload))
		req.Header.Set("Stripe-Signature", firmado.Header)
		w := httptest.NewRecorder()
		nuevoRouterDePrueba(srv).ServeHTTP(w, req)
		return w
	}

	t.Run("webhook no configurado -> 400", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		srv.StripeSecretKey = "sk_test_clave_de_prueba" // habilitado, pero SIN webhook secret
		w := firmarYEnviar(t, srv, []byte(`{}`))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("firma inválida -> 400", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		srv.StripeSecretKey = "sk_test_clave_de_prueba"
		srv.StripeWebhookSecret = secreto

		req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", bytes.NewReader([]byte(`{"type":"checkout.session.completed"}`)))
		req.Header.Set("Stripe-Signature", "t=1,v1=firma-que-no-coincide")
		w := httptest.NewRecorder()
		nuevoRouterDePrueba(srv).ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("evento de un tipo que no nos interesa -> 200, sin tocar la BD", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		srv.StripeSecretKey = "sk_test_clave_de_prueba"
		srv.StripeWebhookSecret = secreto

		w := firmarYEnviar(t, srv, []byte(`{"id":"evt_1","type":"payment_intent.created","data":{"object":{}}}`))
		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("checkout.session.completed pagado -> registra el pago (idempotente)", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		srv.StripeSecretKey = "sk_test_clave_de_prueba"
		srv.StripeWebhookSecret = secreto

		mock.ExpectExec("INSERT INTO pagos").
			WithArgs("1", "3", 1500.0, "Colegiatura", "2026-08", sqlmock.AnyArg(), "cs_test_123", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		payload := []byte(`{
			"id": "evt_1",
			"type": "checkout.session.completed",
			"data": { "object": {
				"id": "cs_test_123",
				"payment_status": "paid",
				"amount_total": 150000,
				"payment_intent": "pi_test_456",
				"metadata": { "guarderia_id": "1", "hijo_id": "3", "periodo": "2026-08", "concepto": "Colegiatura" }
			}}
		}`)
		w := firmarYEnviar(t, srv, payload)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectativas de sqlmock no cumplidas: %v", err)
		}
	})

	t.Run("checkout.session.completed sin pagar (ej. modo setup) -> 200, sin insertar", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		srv.StripeSecretKey = "sk_test_clave_de_prueba"
		srv.StripeWebhookSecret = secreto

		payload := []byte(`{"id":"evt_2","type":"checkout.session.completed","data":{"object":{"id":"cs_test_999","payment_status":"unpaid"}}}`)
		w := firmarYEnviar(t, srv, payload)
		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("checkout.session.completed sin metadata esperada -> 200, sin insertar", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		srv.StripeSecretKey = "sk_test_clave_de_prueba"
		srv.StripeWebhookSecret = secreto

		payload := []byte(`{"id":"evt_3","type":"checkout.session.completed","data":{"object":{"id":"cs_test_888","payment_status":"paid","metadata":{}}}}`)
		w := firmarYEnviar(t, srv, payload)
		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})
}
