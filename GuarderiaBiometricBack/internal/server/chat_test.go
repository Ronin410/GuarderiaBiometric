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

// nuevoServidorDePruebaChat arma un Server con DB y DBAuth apuntando al
// MISMO mock -- el chat ahora toca ambas: mensajes_chat/padres viven en DB,
// usuarios (para el selector de contactos y para validar personalId) vive
// en DBAuth. En este despliegue real (y en la mayoría de las pruebas de
// este paquete) son la misma base física de todos modos.
func nuevoServidorDePruebaChat(t *testing.T) (*Server, sqlmock.Sqlmock) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { mockDB.Close() })

	srv := New()
	srv.DB = mockDB
	srv.DBAuth = mockDB
	srv.JWTKey = []byte("clave-de-prueba-solo-para-tests")
	return srv, mock
}

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

func TestListarContactosChatPadre(t *testing.T) {
	srv, mock := nuevoServidorDePruebaChat(t)
	mock.ExpectQuery("SELECT id, COALESCE\\(nombre, username\\), rol(.|\n)*FROM usuarios").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "nombre", "rol"}).
			AddRow(1, "Directora Ana", "admin").
			AddRow(2, "Maestra Beatriz", "staff"))

	r := nuevoRouterDePrueba(srv)
	req := jsonRequest(http.MethodGet, "/padre/chat/contactos", nil)
	autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
	}
	var contactos []ContactoChat
	if err := json.Unmarshal(w.Body.Bytes(), &contactos); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v", err)
	}
	if len(contactos) != 2 {
		t.Fatalf("se esperaban 2 contactos, se recibieron %d", len(contactos))
	}
}

func TestListarConversaciones(t *testing.T) {
	t.Run("staff normal solo ve las suyas, sin resolver personal_nombre", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaChat(t)
		// generarTokenPrueba siempre pone UserID: 1, sin importar el rol --
		// por eso "personal_id propio" es 1 en todas estas pruebas.
		mock.ExpectQuery("SELECT DISTINCT ON \\(m.padre_id, m.personal_id\\)").
			WithArgs(1, false, 1).
			WillReturnRows(sqlmock.NewRows([]string{"padre_id", "nombre", "personal_id", "contenido", "creado_en"}).
				AddRow(3, "Laura Ramirez", 1, "Hola, ¿cómo va todo?", "2026-08-12T10:00:00Z"))
		mock.ExpectQuery("SELECT padre_id, personal_id, COUNT\\(\\*\\) FROM mensajes_chat").
			WithArgs(1, false, 1).
			WillReturnRows(sqlmock.NewRows([]string{"padre_id", "personal_id", "count"}).AddRow(3, 1, 1))

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
		if len(conversaciones) != 1 || conversaciones[0].PersonalID != 1 || conversaciones[0].NoLeidos != 1 {
			t.Errorf("se esperaba 1 conversación de personal_id=1 con 1 no leído, se recibió: %+v", conversaciones)
		}
	})

	t.Run("admin ve todas y resuelve el nombre de cada staff", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaChat(t)
		mock.ExpectQuery("SELECT DISTINCT ON \\(m.padre_id, m.personal_id\\)").
			WithArgs(1, true, 1).
			WillReturnRows(sqlmock.NewRows([]string{"padre_id", "nombre", "personal_id", "contenido", "creado_en"}).
				AddRow(3, "Laura Ramirez", 2, "Hola, ¿cómo va todo?", "2026-08-12T10:00:00Z"))
		mock.ExpectQuery("SELECT padre_id, personal_id, COUNT\\(\\*\\) FROM mensajes_chat").
			WithArgs(1, true, 1).
			WillReturnRows(sqlmock.NewRows([]string{"padre_id", "personal_id", "count"}))
		mock.ExpectQuery("SELECT id, COALESCE\\(nombre, username\\) FROM usuarios").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "nombre"}).AddRow(2, "Maestra Beatriz"))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/chat/conversaciones", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		var conversaciones []ConversacionResumen
		if err := json.Unmarshal(w.Body.Bytes(), &conversaciones); err != nil {
			t.Fatalf("respuesta no es JSON válido: %v", err)
		}
		if len(conversaciones) != 1 || conversaciones[0].PersonalNombre != "Maestra Beatriz" {
			t.Errorf("se esperaba personal_nombre resuelto a 'Maestra Beatriz', se recibió: %+v", conversaciones)
		}
	})
}

func TestListarFamiliasChat(t *testing.T) {
	srv, mock := nuevoServidorDePruebaChat(t)
	mock.ExpectQuery("SELECT id, COALESCE\\(nombre, 'Familia'\\) FROM padres").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "nombre"}).
			AddRow(3, "Laura Ramirez").
			AddRow(4, "Carlos Pérez"))

	r := nuevoRouterDePrueba(srv)
	req := jsonRequest(http.MethodGet, "/chat/familias", nil)
	autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
	}
	var familias []FamiliaChat
	if err := json.Unmarshal(w.Body.Bytes(), &familias); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v", err)
	}
	// A propósito lista TODAS las familias, sin filtrar por si ya tienen
	// algún mensaje -- "aunque nunca hayan hablado".
	if len(familias) != 2 {
		t.Fatalf("se esperaban 2 familias, se recibieron %d", len(familias))
	}
}

func TestContarNoLeidos(t *testing.T) {
	t.Run("staff normal solo cuenta lo suyo", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaChat(t)
		mock.ExpectQuery("SELECT COUNT\\(DISTINCT \\(padre_id, personal_id\\)\\) FROM mensajes_chat").
			WithArgs(1, false, 1).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/chat/no-leidos", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		var respuesta struct {
			NoLeidos int `json:"no_leidos"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &respuesta); err != nil {
			t.Fatalf("respuesta no es JSON válido: %v", err)
		}
		if respuesta.NoLeidos != 2 {
			t.Errorf("no_leidos = %d; esperado 2", respuesta.NoLeidos)
		}
	})

	t.Run("admin cuenta todos los hilos de la guardería", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaChat(t)
		mock.ExpectQuery("SELECT COUNT\\(DISTINCT \\(padre_id, personal_id\\)\\) FROM mensajes_chat").
			WithArgs(1, true, 1).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/chat/no-leidos", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		var respuesta struct {
			NoLeidos int `json:"no_leidos"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &respuesta); err != nil {
			t.Fatalf("respuesta no es JSON válido: %v", err)
		}
		if respuesta.NoLeidos != 5 {
			t.Errorf("no_leidos = %d; esperado 5", respuesta.NoLeidos)
		}
	})
}

func TestObtenerMensajesStaff(t *testing.T) {
	t.Run("familia de otra guardería -> 404", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaChat(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("9", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/chat/9/1/mensajes", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("código = %d; esperado 404 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("papá no puede ver el inbox de staff -> 403", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaChat(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/chat/conversaciones", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("staff intenta ver el hilo de OTRO miembro del staff -> 403, sin llegar a la BD", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaChat(t)
		r := nuevoRouterDePrueba(srv)
		// autenticado como el usuario id=2 (ver autenticarRequestPrueba), pide
		// el hilo dirigido a personalId=99 -- no es ni admin ni el dueño.
		req := jsonRequest(http.MethodGet, "/chat/3/99/mensajes", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("marca los mensajes del papá como leídos y calcula es_mio -> 200", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaChat(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("3", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery("SELECT id, autor_rol, contenido, creado_en, adjunto_s3_key, adjunto_nombre, adjunto_tipo FROM mensajes_chat").
			WithArgs(1, "3", "1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "autor_rol", "contenido", "creado_en", "adjunto_s3_key", "adjunto_nombre", "adjunto_tipo"}).
				AddRow(1, "papa", "Hola", "2026-08-12T10:00:00Z", nil, nil, nil).
				AddRow(2, "staff", "Hola, ¿en qué te ayudo?", "2026-08-12T10:05:00Z", nil, nil, nil))
		mock.ExpectExec("UPDATE mensajes_chat SET leido = true").
			WithArgs(1, "3", "1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		r := nuevoRouterDePrueba(srv)
		// personalId=1 -- coincide con el UserID:1 que siempre pone
		// generarTokenPrueba, así que este staff SÍ es dueño del hilo.
		req := jsonRequest(http.MethodGet, "/chat/3/1/mensajes", nil)
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

	t.Run("admin sí puede ver el hilo de otro miembro del staff -> 200", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaChat(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("3", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery("SELECT id, autor_rol, contenido, creado_en, adjunto_s3_key, adjunto_nombre, adjunto_tipo FROM mensajes_chat").
			WithArgs(1, "3", "2").
			WillReturnRows(sqlmock.NewRows([]string{"id", "autor_rol", "contenido", "creado_en", "adjunto_s3_key", "adjunto_nombre", "adjunto_tipo"}))
		mock.ExpectExec("UPDATE mensajes_chat SET leido = true").
			WithArgs(1, "3", "2").
			WillReturnResult(sqlmock.NewResult(0, 0))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/chat/3/2/mensajes", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})
}

func TestEnviarMensajeStaff(t *testing.T) {
	t.Run("mensaje vacío sin adjunto -> 400", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaChat(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("3", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		r := nuevoRouterDePrueba(srv)
		req := multipartMensajeRequest("/chat/3/1/mensajes", "   ", false)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("staff intenta escribir en el hilo de otro -> 403, sin llegar a la BD", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaChat(t)
		r := nuevoRouterDePrueba(srv)
		req := multipartMensajeRequest("/chat/3/99/mensajes", "Hola", false)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("staff envía un mensaje en su propio hilo -> 201", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaChat(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("3", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectExec("INSERT INTO mensajes_chat").
			WithArgs(1, "3", "1", 1, "staff", "Claro, con gusto te ayudo.", nil, nil, nil).
			WillReturnResult(sqlmock.NewResult(1, 1))

		r := nuevoRouterDePrueba(srv)
		req := multipartMensajeRequest("/chat/3/1/mensajes", "Claro, con gusto te ayudo.", false)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("código = %d; esperado 201 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("admin responde dentro del hilo de un staff -> se guarda con personal_id del hilo, no del admin", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaChat(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("3", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectExec("INSERT INTO mensajes_chat").
			WithArgs(1, "3", "2", 1, "admin", "Cubriendo a Beatriz hoy.", nil, nil, nil).
			WillReturnResult(sqlmock.NewResult(1, 1))

		r := nuevoRouterDePrueba(srv)
		req := multipartMensajeRequest("/chat/3/2/mensajes", "Cubriendo a Beatriz hoy.", false)
		autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("código = %d; esperado 201 (body: %s)", w.Code, w.Body.String())
		}
	})
}

func TestChatPadre(t *testing.T) {
	t.Run("contacto que no existe en su guardería -> 404", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaChat(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("99", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/padre/chat/99", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("código = %d; esperado 404 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("papá obtiene su hilo con un miembro del staff -> 200", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaChat(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("2", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery("SELECT id, autor_rol, contenido, creado_en, adjunto_s3_key, adjunto_nombre, adjunto_tipo FROM mensajes_chat").
			WithArgs(1, 1, "2").
			WillReturnRows(sqlmock.NewRows([]string{"id", "autor_rol", "contenido", "creado_en", "adjunto_s3_key", "adjunto_nombre", "adjunto_tipo"}).
				AddRow(1, "papa", "Hola", "2026-08-12T10:00:00Z", nil, nil, nil))
		mock.ExpectExec("UPDATE mensajes_chat SET leido = true").
			WithArgs(1, 1, "2").
			WillReturnResult(sqlmock.NewResult(0, 0))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/padre/chat/2", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("papá envía un mensaje a un miembro del staff -> 201", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaChat(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("2", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectExec("INSERT INTO mensajes_chat").
			WithArgs(1, 1, "2", 1, "Buenas tardes, tengo una duda.", nil, nil, nil).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectQuery("SELECT COALESCE\\(nombre, 'Un tutor'\\) FROM padres").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"nombre"}).AddRow("Laura Ramirez"))

		r := nuevoRouterDePrueba(srv)
		req := multipartMensajeRequest("/padre/chat/2", "Buenas tardes, tengo una duda.", false)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("código = %d; esperado 201 (body: %s)", w.Code, w.Body.String())
		}
	})
}
