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

// multipartCircularRequest arma un POST multipart con título/contenido y,
// opcionalmente, una imagen en "imagen" -- handleCrearCircular ya no acepta
// JSON (ver el comentario largo ahí sobre por qué).
func multipartCircularRequest(titulo, contenido string, incluirImagen bool) *http.Request {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	w.WriteField("titulo", titulo)
	w.WriteField("contenido", contenido)
	if incluirImagen {
		fw, _ := w.CreateFormFile("imagen", "foto.jpg")
		fw.Write([]byte("contenido de prueba"))
	}
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/circulares", body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestListarCirculares(t *testing.T) {
	srv, mock := nuevoServidorDePruebaConDB(t)
	mock.ExpectQuery("SELECT c.id, c.titulo, c.contenido, c.creado_en").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "titulo", "contenido", "creado_en", "imagen_s3_key", "leido_por", "total_familias"}).
			AddRow(1, "Suspensión de clases", "El viernes no hay clases por junta de consejo técnico.", "2026-08-10T12:00:00Z", nil, 3, 8))

	r := nuevoRouterDePrueba(srv)
	req := jsonRequest(http.MethodGet, "/circulares", nil)
	autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
	}
	var circulares []CircularConLecturas
	if err := json.Unmarshal(w.Body.Bytes(), &circulares); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v", err)
	}
	if len(circulares) != 1 || circulares[0].Titulo != "Suspensión de clases" {
		t.Fatalf("se esperaba una circular con ese título, se recibió: %+v", circulares)
	}
	if circulares[0].LeidoPor != 3 || circulares[0].TotalFamilias != 8 {
		t.Fatalf("se esperaba leido_por=3 y total_familias=8, se recibió: %+v", circulares[0])
	}
}

func TestListarCircularesComoPadre(t *testing.T) {
	srv, mock := nuevoServidorDePruebaConDB(t)
	mock.ExpectQuery("SELECT id, titulo, contenido, creado_en, imagen_s3_key FROM circulares").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "titulo", "contenido", "creado_en", "imagen_s3_key"}))

	r := nuevoRouterDePrueba(srv)
	req := jsonRequest(http.MethodGet, "/padre/circulares", nil)
	autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
	}
}

func TestMarcarCircularLeida(t *testing.T) {
	t.Run("circular de otra guardería -> 404", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("9", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/padre/circulares/9/leido", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("código = %d; esperado 404 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("papá marca como leída -> 200", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("1", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectExec("INSERT INTO circulares_lecturas").
			WithArgs("1", 1).
			WillReturnResult(sqlmock.NewResult(1, 1))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPost, "/padre/circulares/1/leido", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})
}

func TestDetalleLecturasCircular(t *testing.T) {
	t.Run("papá no puede ver el detalle -> 403", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/circulares/1/lecturas", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("staff ve el detalle -> 200", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("1", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery("SELECT COALESCE\\(pa.nombre, 'Familia'\\), cl.leido_en").
			WithArgs("1").
			WillReturnRows(sqlmock.NewRows([]string{"nombre", "leido_en"}).
				AddRow("Laura Ramirez", "2026-08-12T10:00:00Z"))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/circulares/1/lecturas", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		var lecturas []LecturaCircular
		if err := json.Unmarshal(w.Body.Bytes(), &lecturas); err != nil {
			t.Fatalf("respuesta no es JSON válido: %v", err)
		}
		if len(lecturas) != 1 || lecturas[0].Nombre != "Laura Ramirez" {
			t.Fatalf("se esperaba una lectura de Laura Ramirez, se recibió: %+v", lecturas)
		}
	})
}

func TestCrearCircular(t *testing.T) {
	t.Run("título vacío -> 400", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := multipartCircularRequest("   ", "algo", false)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("código = %d; esperado 400 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("papá no puede publicar -> 403", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := multipartCircularRequest("Aviso", "Texto", false)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("staff publica -> 201", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("INSERT INTO circulares").
			WithArgs(1, "Aviso importante", "Mañana hay junta de padres a las 5pm.", 1, nil).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))

		r := nuevoRouterDePrueba(srv)
		req := multipartCircularRequest("Aviso importante", "Mañana hay junta de padres a las 5pm.", false)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("código = %d; esperado 201 (body: %s)", w.Code, w.Body.String())
		}
	})
}

func TestEliminarCircular(t *testing.T) {
	t.Run("no encontrada -> 404", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT imagen_s3_key FROM circulares").
			WithArgs("99", 1).
			WillReturnError(sql.ErrNoRows)

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodDelete, "/circulares/99", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("código = %d; esperado 404 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("staff elimina -> 200", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT imagen_s3_key FROM circulares").
			WithArgs("7", 1).
			WillReturnRows(sqlmock.NewRows([]string{"imagen_s3_key"}).AddRow(nil))
		mock.ExpectExec("DELETE FROM circulares").
			WithArgs("7", 1).
			WillReturnResult(sqlmock.NewResult(0, 1))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodDelete, "/circulares/7", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})
}
