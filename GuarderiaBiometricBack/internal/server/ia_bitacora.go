package server

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// ia_bitacora.go -- lo que el asistente de soporte puede consultar sobre los
// hijos de QUIEN le está escribiendo.
//
// Antes, a un papá que preguntaba "¿cómo le fue hoy a mi hijo?" el asistente
// le contestaba un instructivo de dónde mirar, porque lo único que tenía era
// el manual. Ahora puede responder con lo que de verdad pasó ese día.
//
// Tres límites, a propósito:
//   - Solo corre para conversaciones de tipo "papa". Un miembro del staff
//     preguntando por un niño tiene su propio panel, con su propio control de
//     permisos; abrirle esta puerta al asistente sería otra vía de acceso a
//     los datos de CUALQUIER niño, sin pasar por ese control.
//   - Solo trae los hijos ligados a esa cuenta en tutor_hijos, y activos.
//   - Solo se llama cuando el modelo decide que hace falta (es una tool, ver
//     ia_soporte.go), no en cada mensaje: preguntar cómo se instala la app no
//     tiene por qué mandarle a nadie la bitácora de un niño.

// nombreHerramientaBitacora es el nombre con el que el modelo la invoca.
const nombreHerramientaBitacora = "consultar_bitacora"

const descripcionHerramientaBitacora = `Consulta la bitácora del día de los hijos del papá que está escribiendo: entrada y salida, qué tanto comió en cada tiempo, si durmió siesta, si trae algún golpe reportado y las notas de la maestra.

Úsala cuando el papá pregunte por cómo le fue a su hijo, qué comió, si durmió, a qué hora llegó o salió, o cualquier cosa del día de su hijo. NO la uses para preguntas sobre cómo funciona la plataforma.`

// legibleComida traduce lo que guarda seguimiento_diario a algo que se pueda
// leer en una frase. "pendiente" es el valor por default de la columna: no es
// que no haya comido, es que la maestra todavía no lo captura -- decir "no
// comió" ahí sería inventarle a un papá algo que nadie reportó.
func legibleComida(valor string) string {
	switch strings.ToLower(strings.TrimSpace(valor)) {
	case "no_comio", "nada":
		return "no comió"
	case "poco":
		return "comió poco"
	case "todo", "bien":
		return "comió bien"
	case "", "pendiente":
		return "sin capturar todavía"
	default:
		return valor
	}
}

// bitacoraDeHijosDePapa arma el texto que se le devuelve al modelo como
// resultado de la herramienta. Va en prosa y no en JSON porque el modelo lo
// va a resumir para un papá, y porque así los valores ya vienen traducidos
// ("comió poco" en vez de "poco") y no puede malinterpretarlos.
func (s *Server) bitacoraDeHijosDePapa(padreID any, guarderiaID any, fecha string) (string, error) {
	if strings.TrimSpace(fecha) == "" {
		fecha = hoyEnZonaLocal(time.Now())
	}

	rows, err := s.DB.Query(`
        SELECT h.id, h.nombre_niño
        FROM hijos h
        JOIN tutor_hijos th ON th.hijo_id = h.id
        WHERE th.padre_id = $1 AND h.guarderia_id = $2 AND h.activo = true
        ORDER BY h.nombre_niño ASC`, padreID, guarderiaID)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	type hijo struct {
		id     int
		nombre string
	}
	var hijos []hijo
	for rows.Next() {
		var h hijo
		if err := rows.Scan(&h.id, &h.nombre); err != nil {
			continue
		}
		hijos = append(hijos, h)
	}
	if len(hijos) == 0 {
		return "Esta cuenta no tiene ningún niño activo ligado.", nil
	}

	var texto strings.Builder
	fmt.Fprintf(&texto, "Bitácora del %s:\n", fecha)

	for _, h := range hijos {
		fmt.Fprintf(&texto, "\n%s:\n", h.nombre)

		// horasAsistenciaDelDia devuelve punteros: nil cuando no hubo ese
		// movimiento, que es distinto de una hora vacía.
		entrada, salida := s.horasAsistenciaDelDia(fmt.Sprint(h.id), fecha)
		switch {
		case entrada == nil:
			texto.WriteString("- No tiene registrada entrada ese día (no asistió, o todavía no lo registran en recepción).\n")
		case salida == nil:
			fmt.Fprintf(&texto, "- Entró a las %s. Todavía no registra salida.\n", *entrada)
		default:
			fmt.Fprintf(&texto, "- Entró a las %s y salió a las %s.\n", *entrada, *salida)
		}

		var desayuno, comida, merienda, esfinter, observaciones sql.NullString
		var durmio sql.NullBool
		err := s.DB.QueryRow(`
            SELECT desayuno, comida, merienda, esfinter, observaciones, durmio
            FROM seguimiento_diario WHERE hijo_id = $1 AND fecha = $2`,
			h.id, fecha,
		).Scan(&desayuno, &comida, &merienda, &esfinter, &observaciones, &durmio)

		if err == sql.ErrNoRows {
			texto.WriteString("- La maestra todavía no ha llenado la bitácora de ese día.\n")
		} else if err != nil {
			return "", err
		} else {
			fmt.Fprintf(&texto, "- Desayuno: %s. Comida: %s. Merienda: %s.\n",
				legibleComida(desayuno.String), legibleComida(comida.String), legibleComida(merienda.String))
			if durmio.Valid && durmio.Bool {
				texto.WriteString("- Sí durmió siesta.\n")
			} else {
				texto.WriteString("- No durmió siesta.\n")
			}
			if esfinter.Valid && strings.TrimSpace(esfinter.String) != "" {
				fmt.Fprintf(&texto, "- Pañal/baño: %s.\n", strings.TrimSpace(esfinter.String))
			}
			if observaciones.Valid && strings.TrimSpace(observaciones.String) != "" {
				fmt.Fprintf(&texto, "- Notas de la maestra: %s\n", strings.TrimSpace(observaciones.String))
			}
		}

		// El golpe y el aseo se capturan en el movimiento de asistencia, no
		// en seguimiento_diario -- por eso se consultan aparte.
		var aseado, golpe sql.NullBool
		var obsAsistencia sql.NullString
		errAsist := s.DB.QueryRow(`
            SELECT aseado, reporte_golpe, observaciones
            FROM asistencia
            WHERE hijo_id = $1 AND (fecha_hora AT TIME ZONE 'America/Mazatlan')::date = $2::date
            ORDER BY fecha_hora DESC LIMIT 1`, h.id, fecha,
		).Scan(&aseado, &golpe, &obsAsistencia)
		if errAsist == nil {
			if golpe.Valid && golpe.Bool {
				texto.WriteString("- SÍ tiene reportado un golpe ese día.\n")
			}
			if aseado.Valid && !aseado.Bool {
				texto.WriteString("- Quedó marcado como no aseado en su último registro.\n")
			}
			if obsAsistencia.Valid && strings.TrimSpace(obsAsistencia.String) != "" {
				fmt.Fprintf(&texto, "- Nota de recepción: %s\n", strings.TrimSpace(obsAsistencia.String))
			}
		}
	}

	return texto.String(), nil
}
