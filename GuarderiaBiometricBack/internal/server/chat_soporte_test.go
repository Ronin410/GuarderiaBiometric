package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

// nuevoServidorDePruebaSoporte -- mismo criterio que nuevoServidorDePruebaChat:
// DB y DBAuth apuntan al mismo mock, porque en este despliegue real son la
// misma base física.
func nuevoServidorDePruebaSoporte(t *testing.T) (*Server, sqlmock.Sqlmock) {
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
	srv.PlatformAdminKey = "clave-de-plataforma-de-prueba"
	return srv, mock
}

func TestCrearConversacionProspecto(t *testing.T) {
	t.Run("datos válidos -> 201 con token", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaSoporte(t)
		mock.ExpectBegin()
		mock.ExpectQuery("INSERT INTO conversaciones_soporte").
			WithArgs("Directora interesada", "directora@ejemplo.com", sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectExec("INSERT INTO mensajes_soporte").
			WithArgs(1, "prospecto", "¿Cuánto cuesta la plataforma?").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/soporte/prospecto", map[string]string{
			"nombre":  "Directora interesada",
			"email":   "directora@ejemplo.com",
			"mensaje": "¿Cuánto cuesta la plataforma?",
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("código = %d; esperado 201 (body: %s)", w.Code, w.Body.String())
		}
		var resp struct {
			Token string `json:"token"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("respuesta no es JSON válido: %v", err)
		}
		if len(resp.Token) != 64 {
			t.Errorf("se esperaba un token de 64 caracteres hex, se recibió %q (%d caracteres)", resp.Token, len(resp.Token))
		}
	})

	t.Run("correo inválido -> 400", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaSoporte(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/soporte/prospecto", map[string]string{
			"nombre":  "Alguien",
			"email":   "no-es-un-correo",
			"mensaje": "Hola",
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("mensaje vacío -> 400", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaSoporte(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/soporte/prospecto", map[string]string{
			"nombre": "Alguien",
			"email":  "alguien@ejemplo.com",
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("de más de 20 peticiones en la ventana -> 429", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaSoporte(t)
		for i := 0; i < 20; i++ {
			mock.ExpectBegin()
			mock.ExpectQuery("INSERT INTO conversaciones_soporte").
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(i + 1))
			mock.ExpectExec("INSERT INTO mensajes_soporte").WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()
		}
		r := nuevoRouterDePrueba(srv)

		cuerpo := map[string]string{"nombre": "Alguien", "email": "alguien@ejemplo.com", "mensaje": "Hola"}
		for i := 0; i < 20; i++ {
			req := jsonRequest(http.MethodPost, "/soporte/prospecto", cuerpo)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusCreated {
				t.Fatalf("petición %d: código = %d; esperado 201 (body: %s)", i+1, w.Code, w.Body.String())
			}
		}

		req := jsonRequest(http.MethodPost, "/soporte/prospecto", cuerpo)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusTooManyRequests {
			t.Fatalf("petición 21: código = %d; esperado 429 (body: %s)", w.Code, w.Body.String())
		}
	})
}

func TestMensajesProspecto(t *testing.T) {
	t.Run("token inexistente -> 404", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaSoporte(t)
		mock.ExpectQuery("SELECT id FROM conversaciones_soporte WHERE token").
			WithArgs("token-que-no-existe").
			WillReturnError(sql.ErrNoRows)

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/soporte/prospecto/token-que-no-existe/mensajes", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("código = %d; esperado 404 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("token válido -> lista mensajes y marca leídos", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaSoporte(t)
		mock.ExpectQuery("SELECT id FROM conversaciones_soporte WHERE token").
			WithArgs("token-valido").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))
		mock.ExpectQuery("SELECT id, autor_rol, contenido, creado_en FROM mensajes_soporte").
			WithArgs(7).
			WillReturnRows(sqlmock.NewRows([]string{"id", "autor_rol", "contenido", "creado_en"}).
				AddRow(1, "prospecto", "Hola", "2026-08-12T10:00:00Z").
				AddRow(2, "plataforma", "¡Hola! Con gusto te ayudo", "2026-08-12T10:05:00Z"))
		mock.ExpectExec("UPDATE mensajes_soporte SET leido = true").WithArgs(int64(7)).WillReturnResult(sqlmock.NewResult(0, 1))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/soporte/prospecto/token-valido/mensajes", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		var mensajes []MensajeSoporte
		if err := json.Unmarshal(w.Body.Bytes(), &mensajes); err != nil {
			t.Fatalf("respuesta no es JSON válido: %v", err)
		}
		if len(mensajes) != 2 || mensajes[0].EsMio != true || mensajes[1].EsMio != false {
			t.Errorf("es_mio incorrecto para el lector prospecto: %+v", mensajes)
		}
	})
}

func TestEnviarMensajeSoporteAutenticado(t *testing.T) {
	t.Run("papá crea su conversación al primer mensaje", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaSoporte(t)
		mock.ExpectQuery("SELECT id FROM conversaciones_soporte WHERE tipo = \\$1 AND guarderia_id = \\$2 AND usuario_id = \\$3").
			WithArgs("papa", 1, 1).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery("SELECT COALESCE\\(nombre, 'Familia'\\) FROM padres").
			WithArgs(1, 1).
			WillReturnRows(sqlmock.NewRows([]string{"nombre"}).AddRow("Laura Ramirez"))
		mock.ExpectQuery("INSERT INTO conversaciones_soporte").
			WithArgs("papa", 1, 1, "Laura Ramirez").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(5))
		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO mensajes_soporte").
			WithArgs(int64(5), "papa", "Tengo una duda sobre mi pago").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("UPDATE conversaciones_soporte SET actualizado_en").
			WithArgs(int64(5)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/soporte/mis-mensajes", map[string]string{"contenido": "Tengo una duda sobre mi pago"})
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("código = %d; esperado 201 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("staff reutiliza su conversación existente", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaSoporte(t)
		mock.ExpectQuery("SELECT id FROM conversaciones_soporte WHERE tipo = \\$1 AND guarderia_id = \\$2 AND usuario_id = \\$3").
			WithArgs("staff", 1, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(9))
		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO mensajes_soporte").
			WithArgs(int64(9), "staff", "¿Cómo cambio el logo de mi guardería?").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("UPDATE conversaciones_soporte SET actualizado_en").
			WithArgs(int64(9)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/soporte/mis-mensajes", map[string]string{"contenido": "¿Cómo cambio el logo de mi guardería?"})
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("código = %d; esperado 201 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("sin sesión -> 401", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaSoporte(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/soporte/mis-mensajes", map[string]string{"contenido": "Hola"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("código = %d; esperado 401 (body: %s)", w.Code, w.Body.String())
		}
	})
}

// TestObtenerMisMensajesSoporteSinConversacion -- abrir el widget de soporte
// sin haber escrito nunca no debe crear una conversación vacía en el inbox
// de la plataforma: el GET solo busca, nunca inserta.
func TestObtenerMisMensajesSoporteSinConversacion(t *testing.T) {
	srv, mock := nuevoServidorDePruebaSoporte(t)
	mock.ExpectQuery("SELECT id FROM conversaciones_soporte WHERE tipo = \\$1 AND guarderia_id = \\$2 AND usuario_id = \\$3").
		WithArgs("papa", 1, 1).
		WillReturnError(sql.ErrNoRows)

	r := nuevoRouterDePrueba(srv)
	req := jsonRequest(http.MethodGet, "/soporte/mis-mensajes", nil)
	autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
	}
	var mensajes []MensajeSoporte
	if err := json.Unmarshal(w.Body.Bytes(), &mensajes); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v", err)
	}
	if len(mensajes) != 0 {
		t.Errorf("se esperaba un hilo vacío, se recibió: %+v", mensajes)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("se ejecutó alguna query inesperada (¿se creó la conversación en el GET?): %v", err)
	}
}

func TestPlataformaSoporte(t *testing.T) {
	t.Run("sin llave -> 401", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaSoporte(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/plataforma/soporte/conversaciones", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("código = %d; esperado 401 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("con llave -> lista conversaciones con nombre de guardería resuelto", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaSoporte(t)
		mock.ExpectQuery("SELECT cs.id, cs.tipo, cs.guarderia_id").
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "tipo", "guarderia_id", "nombre", "email", "cerrada", "actualizado_en", "ultimo_mensaje", "no_leidos",
			}).
				AddRow(1, "prospecto", nil, "Directora interesada", "directora@ejemplo.com", false, "2026-08-12T10:00:00Z", "¿Cuánto cuesta?", 1).
				AddRow(2, "papa", 1, "Laura Ramirez", nil, false, "2026-08-12T09:00:00Z", "Tengo una duda", 0))
		mock.ExpectQuery("SELECT id, nombre FROM guarderias").
			WillReturnRows(sqlmock.NewRows([]string{"id", "nombre"}).AddRow(1, "Guardería Pasitos"))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/plataforma/soporte/conversaciones", nil)
		req.Header.Set("X-Platform-Key", "clave-de-plataforma-de-prueba")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		var conversaciones []ConversacionSoporte
		if err := json.Unmarshal(w.Body.Bytes(), &conversaciones); err != nil {
			t.Fatalf("respuesta no es JSON válido: %v", err)
		}
		if len(conversaciones) != 2 {
			t.Fatalf("se esperaban 2 conversaciones, se recibieron %d", len(conversaciones))
		}
		if conversaciones[1].GuarderiaNombre != "Guardería Pasitos" {
			t.Errorf("no se resolvió el nombre de la guardería: %+v", conversaciones[1])
		}
	})

	t.Run("responder a una conversación inexistente -> 404", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaSoporte(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("999").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/plataforma/soporte/999/mensajes", map[string]string{"contenido": "Hola"})
		req.Header.Set("X-Platform-Key", "clave-de-plataforma-de-prueba")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("código = %d; esperado 404 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("responder a una conversación existente -> 201", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaSoporte(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("5").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO mensajes_soporte").
			WithArgs("5", "plataforma", "Claro, con gusto te apoyo").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("UPDATE conversaciones_soporte SET actualizado_en").
			WithArgs("5").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/plataforma/soporte/5/mensajes", map[string]string{"contenido": "Claro, con gusto te apoyo"})
		req.Header.Set("X-Platform-Key", "clave-de-plataforma-de-prueba")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("código = %d; esperado 201 (body: %s)", w.Code, w.Body.String())
		}
	})
}
