package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

// multipartRequest arma un POST multipart con un campo "tipo" y, opcionalmente,
// un archivo en el campo "archivo" -- para probar el caso de "no seleccioné
// archivo" sin necesitar tocar S3 (los handlers validan esto antes de subir
// nada, así que srv.S3 puede quedar nil en la prueba).
func multipartRequest(url, tipo string, incluirArchivo bool) *http.Request {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	w.WriteField("tipo", tipo)
	if incluirArchivo {
		fw, _ := w.CreateFormFile("archivo", "acta.pdf")
		fw.Write([]byte("contenido de prueba"))
	}
	w.Close()

	req := httptest.NewRequest(http.MethodPost, url, body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestListarDocumentos(t *testing.T) {
	t.Run("niño de otra guardería -> 404", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("9", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/hijos/9/documentos", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("código = %d; esperado 404 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("niño sin documentos subidos -> catálogo completo en null", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("3", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery("SELECT tipo, nombre_archivo, s3_key, subido_en FROM documentos_nino").
			WithArgs("3").
			WillReturnRows(sqlmock.NewRows([]string{"tipo", "nombre_archivo", "s3_key", "subido_en"}))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/hijos/3/documentos", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		var docs []DocumentoNino
		if err := json.Unmarshal(w.Body.Bytes(), &docs); err != nil {
			t.Fatalf("respuesta no es JSON válido: %v", err)
		}
		if len(docs) != len(ordenTiposDocumento) {
			t.Fatalf("se esperaban %d tipos en el catálogo, se recibieron %d", len(ordenTiposDocumento), len(docs))
		}
		for _, d := range docs {
			if d.NombreArchivo != nil || d.URL != nil {
				t.Errorf("tipo %q: se esperaba sin subir (nulls), se recibió %+v", d.Tipo, d)
			}
		}
	})
}

func TestSubirDocumento(t *testing.T) {
	t.Run("niño de otra guardería -> 404", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("9", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		r := nuevoRouterDePrueba(srv)
		req := multipartRequest("/hijos/9/documentos", "curp", true)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("código = %d; esperado 404 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("tipo inválido -> 400", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("3", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		r := nuevoRouterDePrueba(srv)
		req := multipartRequest("/hijos/3/documentos", "pasaporte", true)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("sin archivo adjunto -> 400", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("3", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		r := nuevoRouterDePrueba(srv)
		req := multipartRequest("/hijos/3/documentos", "curp", false)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("papá no puede subir documentos -> 403", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := multipartRequest("/hijos/3/documentos", "curp", true)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})
}

func TestEliminarDocumento(t *testing.T) {
	t.Run("niño de otra guardería -> 404", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("9", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodDelete, "/hijos/9/documentos/curp", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("código = %d; esperado 404 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("documento no subido todavía -> 404", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("3", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery("SELECT s3_key FROM documentos_nino").
			WithArgs("3", "curp").
			WillReturnError(sql.ErrNoRows)

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodDelete, "/hijos/3/documentos/curp", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("código = %d; esperado 404 (body: %s)", w.Code, w.Body.String())
		}
	})
}
