package server

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

// La consulta de hijos es la barrera de privacidad de toda esta función: si
// dejara de cruzar por tutor_hijos, o de filtrar por guardería y por activo,
// el asistente podría contarle a un papá el día de un niño ajeno. Esta prueba
// falla si alguien afloja cualquiera de esas tres condiciones.
func TestBitacoraDeHijosDePapaSoloTraeLosSuyos(t *testing.T) {
	srv, mock := nuevoServidorDePruebaConDB(t)

	mock.ExpectQuery("FROM hijos h(.|\n)*JOIN tutor_hijos th ON th.hijo_id = h.id(.|\n)*WHERE th.padre_id = \\$1 AND h.guarderia_id = \\$2 AND h.activo = true").
		WithArgs(7, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "nombre_niño"}).AddRow(3, "Ryan"))
	mock.ExpectQuery("FROM asistencia").
		WithArgs("3", "2026-09-04").
		WillReturnRows(sqlmock.NewRows([]string{"entrada", "salida"}).AddRow(nil, nil))
	mock.ExpectQuery("FROM seguimiento_diario").
		WithArgs(3, "2026-09-04").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("FROM asistencia").
		WithArgs(3, "2026-09-04").
		WillReturnError(sql.ErrNoRows)

	texto, err := srv.bitacoraDeHijosDePapa(7, 1, "2026-09-04")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if !strings.Contains(texto, "Ryan") {
		t.Fatalf("se esperaba el nombre del niño, se recibió: %q", texto)
	}
	if !strings.Contains(texto, "todavía no ha llenado la bitácora") {
		t.Fatalf("un día sin bitácora debe decirse tal cual, se recibió: %q", texto)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("consultas no coinciden: %v", err)
	}
}

// Una cuenta sin hijos ligados no debe provocar una consulta más ni un error:
// se le contesta al modelo que no hay nada que reportar.
func TestBitacoraDeHijosDePapaSinHijos(t *testing.T) {
	srv, mock := nuevoServidorDePruebaConDB(t)
	mock.ExpectQuery("FROM hijos h").
		WillReturnRows(sqlmock.NewRows([]string{"id", "nombre_niño"}))

	texto, err := srv.bitacoraDeHijosDePapa(7, 1, "2026-09-04")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if !strings.Contains(texto, "no tiene ningún niño activo") {
		t.Fatalf("se esperaba el aviso de cuenta sin niños, se recibió: %q", texto)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("consultas no coinciden: %v", err)
	}
}

// "pendiente" es el valor por default de la columna: significa que la maestra
// todavía no lo captura, NO que el niño no comió. Confundirlos le reportaría
// a un papá que su hijo no probó bocado cuando nadie dijo eso.
func TestLegibleComidaNoInventaQueNoComio(t *testing.T) {
	casos := map[string]string{
		"pendiente": "sin capturar todavía",
		"":          "sin capturar todavía",
		"no_comio":  "no comió",
		"poco":      "comió poco",
		"todo":      "comió bien",
	}
	for valor, esperado := range casos {
		if got := legibleComida(valor); got != esperado {
			t.Errorf("legibleComida(%q) = %q; esperado %q", valor, got, esperado)
		}
	}
}

// Sin fecha se consulta el día de hoy en la zona de la guardería, no el UTC
// del servidor: de noche, un servidor en UTC ya está en el día siguiente y le
// contestaría a un papá que su hijo no asistió.
func TestBitacoraDeHijosDePapaUsaHoyLocalSinFecha(t *testing.T) {
	srv, mock := nuevoServidorDePruebaConDB(t)
	hoy := hoyEnZonaLocal(time.Now())
	mock.ExpectQuery("FROM hijos h").
		WillReturnRows(sqlmock.NewRows([]string{"id", "nombre_niño"}).AddRow(3, "Ryan"))
	mock.ExpectQuery("FROM asistencia").WithArgs("3", hoy).
		WillReturnRows(sqlmock.NewRows([]string{"entrada", "salida"}).AddRow(nil, nil))
	mock.ExpectQuery("FROM seguimiento_diario").WithArgs(3, hoy).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("FROM asistencia").WithArgs(3, hoy).
		WillReturnError(sql.ErrNoRows)

	if _, err := srv.bitacoraDeHijosDePapa(7, 1, ""); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("consultas no coinciden: %v", err)
	}
}
