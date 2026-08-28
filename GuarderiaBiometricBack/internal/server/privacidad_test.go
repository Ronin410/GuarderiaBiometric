package server

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

// multipartAvisoRequest arma un PUT multipart con "texto" y, opcionalmente,
// un archivo en "pdf" -- handleActualizarAviso ya no acepta JSON-solo-texto
// (ver el comentario largo ahí sobre por qué). El archivo que arma
// CreateFormFile siempre trae Content-Type "application/octet-stream" (así
// funciona mime/multipart en Go, sin importar el nombre), así que este
// helper también sirve para probar el rechazo de "el archivo debe ser un
// PDF" sin necesitar un PDF de verdad.
func multipartAvisoRequest(texto string, incluirArchivo bool) *http.Request {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	w.WriteField("texto", texto)
	if incluirArchivo {
		fw, _ := w.CreateFormFile("pdf", "aviso.pdf")
		fw.Write([]byte("contenido de prueba"))
	}
	w.Close()

	req := httptest.NewRequest(http.MethodPut, "/admin/aviso-privacidad", body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestObtenerAvisoPrivacidad(t *testing.T) {
	t.Run("solo texto configurado -> configurado=true, sin pdf_url", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT COALESCE\\(aviso_privacidad_texto").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"texto", "version", "pdf_key"}).
				AddRow("Texto del aviso", "v2", nil))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/aviso-privacidad", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		if bytes.Contains(w.Body.Bytes(), []byte("pdf_url")) {
			t.Errorf("no debería incluir pdf_url cuando no hay PDF configurado, body: %s", w.Body.String())
		}
	})

	t.Run("nada configurado -> configurado=false", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT COALESCE\\(aviso_privacidad_texto").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"texto", "version", "pdf_key"}).
				AddRow("", "v1", nil))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/aviso-privacidad", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		if !bytes.Contains(w.Body.Bytes(), []byte(`"configurado":false`)) {
			t.Errorf("se esperaba configurado=false, body: %s", w.Body.String())
		}
	})
}

func TestActualizarAvisoPrivacidad(t *testing.T) {
	t.Run("ni texto ni pdf -> 400", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := multipartAvisoRequest("   ", false)
		autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("archivo que no es PDF -> 400", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := multipartAvisoRequest("", true)
		autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("papá no puede configurar el aviso -> 403", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := multipartAvisoRequest("Texto nuevo", false)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})

	// El caso que en verdad importa aquí: staff NUNCA debe poder tocar
	// Configuración, ni siquiera con permisos personalizados -- por eso
	// esta ruta usa RequireAdmin() y no RequireArea("configuracion").
	t.Run("staff NUNCA puede configurar el aviso, aunque tenga permisos personalizados -> 403", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := multipartAvisoRequest("Texto nuevo", false)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("admin guarda texto -> siguiente versión, sin PDF nuevo", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		// Sin PDF anterior en este caso a propósito -- si lo hubiera,
		// handleActualizarAviso dispara un borrarDeS3(...) en goroutine para
		// limpiarlo, y srv.S3 es nil en esta prueba (no hay credenciales de
		// AWS reales que configurar aquí), lo que haría panic. Mismo
		// criterio que ya usan documentos_test.go/chat_test.go: no ejercitar
		// una llamada real a S3 en pruebas unitarias.
		mock.ExpectQuery("SELECT aviso_privacidad_version, aviso_privacidad_pdf_s3_key").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"version", "pdf_key"}).AddRow("v1", nil))
		mock.ExpectExec("UPDATE guarderias SET aviso_privacidad_texto").
			WithArgs("Texto nuevo", "v2", nil, 1).
			WillReturnResult(sqlmock.NewResult(0, 1))

		r := nuevoRouterDePrueba(srv)
		req := multipartAvisoRequest("Texto nuevo", false)
		autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})
}
