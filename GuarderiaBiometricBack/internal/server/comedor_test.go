package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestListarPedidosComedorHijo(t *testing.T) {
	t.Run("niño ajeno -> 403", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("9", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/padre/hijos/9/pedidos-comedor", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("papá ve las excepciones de su hijo -> 200", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("5", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery("SELECT id, hijo_id, fecha, desayuno, comida, merienda, COALESCE\\(notas, ''\\) FROM pedidos_comedor").
			WithArgs("5", sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "hijo_id", "fecha", "desayuno", "comida", "merienda", "notas"}).
				AddRow(1, 5, "2026-09-01T00:00:00Z", false, true, true, "Alérgico a los cacahuates"))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/padre/hijos/5/pedidos-comedor", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		var pedidos []PedidoComedor
		if err := json.Unmarshal(w.Body.Bytes(), &pedidos); err != nil {
			t.Fatalf("respuesta no es JSON válido: %v", err)
		}
		if len(pedidos) != 1 || pedidos[0].Fecha != "2026-09-01" || pedidos[0].Desayuno {
			t.Fatalf("se esperaba una excepción sin desayuno, se recibió: %+v", pedidos)
		}
	})
}

func TestGuardarPedidoComedor(t *testing.T) {
	t.Run("niño ajeno -> 403", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("9", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPut, "/padre/hijos/9/pedidos-comedor/2026-09-01", map[string]bool{"desayuno": false, "comida": true, "merienda": true})
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("sin excepciones -> borra el pedido y vuelve al default", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("5", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectExec("DELETE FROM pedidos_comedor").
			WithArgs("5", "2026-09-01").
			WillReturnResult(sqlmock.NewResult(0, 1))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPut, "/padre/hijos/5/pedidos-comedor/2026-09-01", map[string]any{
			"desayuno": true, "comida": true, "merienda": true, "notas": "",
		})
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("con excepción -> upsert", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT EXISTS").WithArgs("5", 1).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectExec("INSERT INTO pedidos_comedor").
			WithArgs("5", 1, "2026-09-01", false, true, true, "Ya desayunó en casa", 1).
			WillReturnResult(sqlmock.NewResult(1, 1))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodPut, "/padre/hijos/5/pedidos-comedor/2026-09-01", map[string]any{
			"desayuno": false, "comida": true, "merienda": true, "notas": "Ya desayunó en casa",
		})
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
	})
}

func TestResumenComedor(t *testing.T) {
	t.Run("papá no puede ver el resumen -> 403", func(t *testing.T) {
		srv, _ := nuevoServidorDePruebaConDB(t)
		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/pedidos-comedor", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "papa", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("código = %d; esperado 403 (body: %s)", w.Code, w.Body.String())
		}
	})

	t.Run("staff ve el resumen con conteos descontando excepciones -> 200", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM hijos").WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(8))
		mock.ExpectQuery("SELECT p.id, p.hijo_id, p.fecha, p.desayuno, p.comida, p.merienda, COALESCE\\(p.notas, ''\\), h.nombre_niño").
			WithArgs(1, "2026-09-01").
			WillReturnRows(sqlmock.NewRows([]string{"id", "hijo_id", "fecha", "desayuno", "comida", "merienda", "notas", "hijo_nombre"}).
				AddRow(1, 5, "2026-09-01T00:00:00Z", false, true, true, "Alérgico a los cacahuates", "Valentina Cruz"))

		r := nuevoRouterDePrueba(srv)
		req := jsonRequest(http.MethodGet, "/pedidos-comedor?fecha=2026-09-01", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		var resumen ResumenComedorDia
		if err := json.Unmarshal(w.Body.Bytes(), &resumen); err != nil {
			t.Fatalf("respuesta no es JSON válido: %v", err)
		}
		if resumen.TotalNinos != 8 || resumen.Resumen.Desayuno != 7 || resumen.Resumen.Comida != 8 {
			t.Fatalf("se esperaba desayuno=7 (8-1) y comida=8, se recibió: %+v", resumen.Resumen)
		}
		if len(resumen.Excepciones) != 1 || resumen.Excepciones[0].HijoNombre != "Valentina Cruz" {
			t.Fatalf("se esperaba una excepción de Valentina Cruz, se recibió: %+v", resumen.Excepciones)
		}
	})
}
