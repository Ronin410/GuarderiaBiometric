package server

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

// multipartMensajeRequest arma un POST multipart con un campo "contenido" y,
// opcionalmente, un archivo en "archivo" -- los endpoints de enviar mensaje
// ya no aceptan JSON (ver leerMensajeConAdjunto en chat.go), justo para
// poder mandar texto y adjunto en la misma petición.
func multipartMensajeRequest(url, contenido string, incluirArchivo bool) *http.Request {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	w.WriteField("contenido", contenido)
	if incluirArchivo {
		fw, _ := w.CreateFormFile("archivo", "foto.jpg")
		fw.Write([]byte("contenido de prueba"))
	}
	w.Close()

	req := httptest.NewRequest(http.MethodPost, url, body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestListarConversaciones(t *testing.T) {
	srv, mock := nuevoServidorDePruebaConDB(t)
	mock.ExpectQuery("SELECT DISTINCT ON \\(m.padre_id\\)").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"padre_id", "nombre", "contenido", "creado_en"}).
			AddRow(3, "Laura Ramirez", "Hola, ¿cómo va todo?", "2026-08-12T10:00:00Z").
			AddRow(5, "Carlos Torres", "Gracias por la información", "2026-08-11T09:00:00Z"))
	mock.ExpectQuery("SELECT padre_id, COUNT\\(\\*\\) FROM mensajes_chat").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"padre_id", "count"}).AddRow(3, 2))

	r := nuevoRouterDePrueba(srv)
	req := jsonRequest(http.MethodGet, "/chat/conversaciones", nil)
	autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
	}
	var conversaciones []ConversacionResumen
	if err := json.Unmarshal(w.Body.Bytes(), &conversaciones); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v", err)
	}
	if len(conversaciones) != 2 {
		t.Fatalf("se esperaban 2 conversaciones, se recibieron %d", len(conversaciones))
	}
	if conversaciones[0].PadreID != 3 || conversaciones[0].NoLeidos != 2 {
		t.Errorf("la conversación más reciente debería ser padre_id=3 con 2 no leídos, se recibió: %+v", conversaciones[0])
	}
}

func TestObtenerMensajesStaff(t *testing.T) {
	t.Run("familia de otra guardería -> 404", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("9", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/chat/9/mensajes", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("código = %d; esperado 404 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("papá no puede ver el inbox de staff -> 403", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/chat/conversaciones", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("marca los mensajes del papá como leídos y calcula es_mio -> 200", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("3", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery("SELECT id, autor_rol, contenido, creado_en, adjunto_s3_key, adjunto_nombre, adjunto_tipo FROM mensajes_chat").
			WithArgs(1, "3").
			WillReturnRows(sqlmock.NewRows([]string{"id", "autor_rol", "contenido", "creado_en", "adjunto_s3_key", "adjunto_nombre", "adjunto_tipo"}).
				AddRow(1, "papa", "Hola", "2026-08-12T10:00:00Z", nil, nil, nil).
				AddRow(2, "staff", "Hola, ¿en qué te ayudo?", "2026-08-12T10:05:00Z", nil, nil, nil))
		mock.ExpectExec("UPDATE mensajes_chat SET leido = true").
			WithArgs(1, "3").
			WillReturnResult(sqlmock.NewResult(0, 1))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/chat/3/mensajes", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		var mensajes []MensajeChat
		if err := json.Unmarshal(w.Body.Bytes(), &mensajes); err != nil {
			t.Fatalf("respuesta no es JSON válido: %v", err)
		}
		if mensajes[0].EsMio {
			t.Errorf("el mensaje del papá no debería marcarse como es_mio=true para staff: %+v", mensajes[0])
		}
		if !mensajes[1].EsMio {
			t.Errorf("el mensaje de staff debería marcarse como es_mio=true para staff: %+v", mensajes[1])
		}
	})
}

func TestEnviarMensajeStaff(t *testing.T) {
	t.Run("mensaje vacío sin adjunto -> 400", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("3", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		r := nuevoRouterDePrueba(srv)
		req := multipartMensajeRequest("/chat/3/mensajes", "   ", false)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("staff envía un mensaje -> 201", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("3", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectExec("INSERT INTO mensajes_chat").
			WithArgs(1, "3", 1, "staff", "Claro, con gusto te ayudo.", nil, nil, nil).
			WillReturnResult(sqlmock.NewResult(1, 1))

		r := nuevoRouterDePrueba(srv)
		req := multipartMensajeRequest("/chat/3/mensajes", "Claro, con gusto te ayudo.", false)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("código = %d; esperado 201 (body: %s)", w.Code, w.Body.String())
		}
	})
}

func TestChatPadre(t *testing.T) {
	t.Run("papá obtiene su propio hilo -> 200", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT id, autor_rol, contenido, creado_en, adjunto_s3_key, adjunto_nombre, adjunto_tipo FROM mensajes_chat").
			WithArgs(1, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "autor_rol", "contenido", "creado_en", "adjunto_s3_key", "adjunto_nombre", "adjunto_tipo"}).
				AddRow(1, "papa", "Hola", "2026-08-12T10:00:00Z", nil, nil, nil))
		mock.ExpectExec("UPDATE mensajes_chat SET leido = true").
			WithArgs(1, 1).
			WillReturnResult(sqlmock.NewResult(0, 0))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/padre/chat", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("papá envía un mensaje -> 201", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectExec("INSERT INTO mensajes_chat").
			WithArgs(1, 1, 1, "Buenas tardes, tengo una duda.", nil, nil, nil).
			WillReturnResult(sqlmock.NewResult(1, 1))

		r := nuevoRouterDePrueba(srv)
		req := multipartMensajeRequest("/padre/chat", "Buenas tardes, tengo una duda.", false)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("código = %d; esperado 201 (body: %s)", w.Code, w.Body.String())
		}
	})
}
