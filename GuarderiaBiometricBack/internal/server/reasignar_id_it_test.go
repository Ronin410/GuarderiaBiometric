package server

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"

	appdb "biometrico/internal/db"
)

// TestResolverChoqueID_Integracion es una prueba de integración de verdad
// (Postgres real, no sqlmock) para handleResolverChoqueID -- la lógica que
// mueve a un padre a un id nuevo cuando su id choca con la cuenta de otra
// persona (admin/staff) por la convención usuarios.id == padres.id. sqlmock
// no sirve aquí: lo que hay que probar es justo si el orden de las
// sentencias (INSERT con face_id temporal -> repuntar FKs -> DELETE ->
// restaurar face_id) de verdad respeta las restricciones reales de
// Postgres (FK sin ON UPDATE CASCADE, UNIQUE en face_id), y eso solo lo
// puede confirmar una base real.
//
// Se salta sola si no hay TEST_DATABASE_URL (ej. en máquinas sin Postgres
// instalado) -- no debe tumbar "go test ./..." en un entorno que no la
// tenga configurada.
func TestResolverChoqueID_Integracion(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL no configurada -- se salta la prueba de integración")
	}

	conexion, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("no se pudo abrir la conexión de prueba: %v", err)
	}
	// Un solo Cleanup para las dos cosas, en el orden correcto (limpiar
	// tablas y HASTA DESPUÉS cerrar la conexión) -- t.Cleanup corre después
	// de que la función ya retornó, así que un defer conexion.Close() aparte
	// se ejecutaría ANTES y dejaría a un t.Cleanup(limpiar) posterior
	// intentando usar una conexión ya cerrada.
	limpiar := func() {
		for _, tabla := range []string{"mensajes_chat", "circulares_lecturas", "encuesta_respuestas", "push_subscripciones", "consentimientos", "asistencia", "tutor_hijos", "hijos", "padres", "usuarios", "guarderias"} {
			if _, err := conexion.Exec("DELETE FROM " + tabla); err != nil {
				t.Errorf("no se pudo limpiar %s: %v", tabla, err)
			}
		}
	}
	t.Cleanup(func() {
		limpiar()
		conexion.Close()
	})

	if err := appdb.RunMigrations(conexion); err != nil {
		t.Fatalf("no se pudieron aplicar las migraciones: %v", err)
	}

	// Limpieza total antes de empezar -- la prueba deja fixtures fijos
	// (guarderia_id=1, padre id=1, admin id=1) para que sea reproducible.
	limpiar()

	srv := New()
	srv.DB = conexion
	srv.DBAuth = conexion // en esta prueba (como en el despliegue real de Render) viven en la misma base física.
	srv.JWTKey = []byte("clave-de-prueba-solo-para-tests")

	// --- Fixture: exactamente el escenario reportado ---
	// guardería 1, admin1 con id=1 (cuenta que "chocó" primero), y un padre
	// (Alejandro) que por el bug histórico de creación de ids terminó
	// también con id=1 en la tabla padres -- con un hijo vinculado,
	// asistencia registrada y su consentimiento, para probar que TODO eso
	// sobrevive el movimiento.
	if _, err := conexion.Exec(`INSERT INTO guarderias (id, nombre, slug) VALUES (1, 'Guardería Sol y Luna', 'demo')`); err != nil {
		t.Fatalf("fixture guarderias: %v", err)
	}
	if _, err := conexion.Exec(`
        INSERT INTO usuarios (id, guarderia_id, username, password_hash, pin_admin, rol, nombre, activo)
        VALUES (1, 1, 'admin1', 'hash', '1234', 'admin', 'Admin Uno', true)`); err != nil {
		t.Fatalf("fixture usuarios (admin1): %v", err)
	}
	if _, err := conexion.Exec(`INSERT INTO padres (id, nombre, face_id, guarderia_id, celular, recibe_whatsapp) VALUES (1, 'Alejandro', 'face-alejandro-123', 1, '6674983913', true)`); err != nil {
		t.Fatalf("fixture padres: %v", err)
	}
	if _, err := conexion.Exec(`INSERT INTO hijos (id, nombre_niño, guarderia_id) VALUES (1, 'Ryan Alejandro Bueno Ordóñez', 1)`); err != nil {
		t.Fatalf("fixture hijos: %v", err)
	}
	if _, err := conexion.Exec(`INSERT INTO tutor_hijos (padre_id, hijo_id, guarderia_id) VALUES (1, 1, 1)`); err != nil {
		t.Fatalf("fixture tutor_hijos: %v", err)
	}
	if _, err := conexion.Exec(`INSERT INTO asistencia (padre_id, hijo_id, guarderia_id, tipo_movimiento) VALUES (1, 1, 1, 'REGISTRO')`); err != nil {
		t.Fatalf("fixture asistencia: %v", err)
	}
	if _, err := conexion.Exec(`INSERT INTO consentimientos (padre_id, padre_nombre_historico, guarderia_id, version_aviso) VALUES (1, 'Alejandro', 1, 'v1')`); err != nil {
		t.Fatalf("fixture consentimientos: %v", err)
	}

	r := nuevoRouterDePrueba(srv)

	// --- Acción: reasignar el id del padre 1 ---
	req := httptest.NewRequest(http.MethodPost, "/padres/1/reasignar-id", nil)
	autenticarRequestPrueba(t, req, srv.JWTKey, "admin", time.Hour)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("código = %d; esperado 200 (body: %s)", w.Code, w.Body.String())
	}

	var nuevoID int
	if err := conexion.QueryRow("SELECT id FROM padres WHERE face_id = 'face-alejandro-123'").Scan(&nuevoID); err != nil {
		t.Fatalf("no se encontró al padre por su face_id original tras la reasignación: %v", err)
	}
	if nuevoID == 1 {
		t.Fatalf("el padre se quedó en el id 1 -- no se movió")
	}
	t.Logf("padre movido del id 1 al id %d", nuevoID)

	// El id viejo ya no debe existir en padres.
	var existeViejo bool
	conexion.QueryRow("SELECT EXISTS(SELECT 1 FROM padres WHERE id = 1)").Scan(&existeViejo)
	if existeViejo {
		t.Errorf("la fila vieja (id=1) de padres debía haberse borrado")
	}

	// El face_id real debe estar restaurado (sin el sufijo temporal).
	var faceIDFinal string
	conexion.QueryRow("SELECT face_id FROM padres WHERE id = $1", nuevoID).Scan(&faceIDFinal)
	if faceIDFinal != "face-alejandro-123" {
		t.Errorf("face_id final = %q; esperado el original sin sufijo temporal", faceIDFinal)
	}

	// tutor_hijos, asistencia y consentimientos deben apuntar al id nuevo.
	var cuentaTutorHijos, cuentaAsistencia, cuentaConsentimientos int
	conexion.QueryRow("SELECT COUNT(*) FROM tutor_hijos WHERE padre_id = $1", nuevoID).Scan(&cuentaTutorHijos)
	conexion.QueryRow("SELECT COUNT(*) FROM asistencia WHERE padre_id = $1", nuevoID).Scan(&cuentaAsistencia)
	conexion.QueryRow("SELECT COUNT(*) FROM consentimientos WHERE padre_id = $1", nuevoID).Scan(&cuentaConsentimientos)
	if cuentaTutorHijos != 1 {
		t.Errorf("tutor_hijos con el id nuevo = %d; esperado 1", cuentaTutorHijos)
	}
	if cuentaAsistencia != 1 {
		t.Errorf("asistencia con el id nuevo = %d; esperado 1", cuentaAsistencia)
	}
	if cuentaConsentimientos != 1 {
		t.Errorf("consentimientos con el id nuevo = %d; esperado 1", cuentaConsentimientos)
	}

	// admin1 (id=1 en usuarios) debe seguir intacto -- este movimiento no
	// debía tocarlo para nada.
	var adminUsername string
	if err := conexion.QueryRow("SELECT username FROM usuarios WHERE id = 1").Scan(&adminUsername); err != nil || adminUsername != "admin1" {
		t.Errorf("la cuenta admin1 (id=1) debía seguir intacta, error=%v username=%q", err, adminUsername)
	}

	// --- La prueba real: ahora SÍ debe poder crear la cuenta del portal ---
	reqCuenta := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/padres/%d/crear-cuenta", nuevoID),
		strings.NewReader(`{"username":"alejandro_papa","password":"Password123"}`))
	reqCuenta.Header.Set("Content-Type", "application/json")
	autenticarRequestPrueba(t, reqCuenta, srv.JWTKey, "admin", time.Hour)
	wCuenta := httptest.NewRecorder()
	r.ServeHTTP(wCuenta, reqCuenta)

	if wCuenta.Code != http.StatusCreated {
		t.Fatalf("crear-cuenta tras reasignar -> código = %d (esperado 201); body: %s", wCuenta.Code, wCuenta.Body.String())
	}

	var rolNuevo string
	if err := conexion.QueryRow("SELECT rol FROM usuarios WHERE id = $1", nuevoID).Scan(&rolNuevo); err != nil || rolNuevo != "papa" {
		t.Errorf("se esperaba una cuenta rol 'papa' en el id nuevo, error=%v rol=%q", err, rolNuevo)
	}
}
