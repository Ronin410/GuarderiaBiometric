package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

// TestEliminarPagoEsBorradoSuave cubre el cambio de DELETE a UPDATE: "eliminar
// un pago" ya no hace desaparecer la fila -- la marca con eliminado_en/
// eliminado_por (a qué cuenta se le atribuye) y listo, para que un registro
// financiero nunca desaparezca de la base sin dejar rastro de que existió.
func TestEliminarPagoEsBorradoSuave(t *testing.T) {
	t.Run("pago existente -> UPDATE marca eliminado_en/eliminado_por, no DELETE", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)

		mock.ExpectExec("UPDATE pagos SET eliminado_en = NOW\\(\\), eliminado_por = \\$1").
			WithArgs(1, "77", 1).
			WillReturnResult(sqlmock.NewResult(0, 1))

		r := nuevoRouterDePrueba(srv)
		req := httptest.NewRequest(http.MethodDelete, "/pagos/77", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectativas de sqlmock no cumplidas -- ¿se sigue usando DELETE? %v", err)
		}
	})

	t.Run("pago inexistente o ya eliminado -> 404", func(t *testing.T) {
		srv, mock := nuevoServidorDePruebaConDB(t)

		mock.ExpectExec("UPDATE pagos SET eliminado_en = NOW\\(\\), eliminado_por = \\$1").
			WithArgs(1, "999", 1).
			WillReturnResult(sqlmock.NewResult(0, 0))

		r := nuevoRouterDePrueba(srv)
		req := httptest.NewRequest(http.MethodDelete, "/pagos/999", nil)
		autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("código = %d; esperado 404 (body: %s)", w.Code, w.Body.String())
		}
	})
}

// TestListarPagosExcluyeEliminados cubre que un pago dado de baja (borrado
// suave) siga existiendo en la base pero deje de aparecer en el historial
// que ve el staff -- para cualquier pantalla de la app debe comportarse
// exactamente como si nunca hubiera existido.
func TestListarPagosExcluyeEliminados(t *testing.T) {
	srv, mock := nuevoServidorDePruebaConDB(t)

	mock.ExpectQuery("SELECT(.|\n)*FROM pagos(.|\n)*AND eliminado_en IS NULL").
		WithArgs(1, "", "").
		WillReturnRows(sqlmock.NewRows([]string{"id", "hijo_id", "monto", "concepto", "periodo", "fecha_pago", "metodo_pago", "observaciones"}).
			AddRow(1, 5, 800.0, "Colegiatura", "2026-09", "2026-09-05", "efectivo", ""))

	r := nuevoRouterDePrueba(srv)
	req := httptest.NewRequest(http.MethodGet, "/pagos", nil)
	autenticarRequestPrueba(t, req, srv.JWTKey, "staff", time.Hour)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectativas de sqlmock no cumplidas -- ¿la consulta ya no filtra eliminado_en? %v", err)
	}
}
