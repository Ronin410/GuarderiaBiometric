package server

import (
	"database/sql"
	"fmt"
	"strconv"

	"github.com/lib/pq"
)

// destinatarios.go -- "que si alguien quiere hacer una encuesta pueda
// escoger a quién va dirigida, y de la misma forma las circulares".
//
// Circulares y encuestas comparten exactamente el mismo modelo de audiencia,
// así que la lógica vive aquí una sola vez en vez de copiada en los dos
// archivos: una publicación va para TODAS las familias (para_todos = true,
// que es como sigue naciendo cualquiera que no elija grupos) o para los
// grupos listados en su tabla puente.
//
// Un papá ve una publicación dirigida si tiene AL MENOS UN hijo activo en
// alguno de esos grupos. Con dos hijos en salones distintos ve las dos
// listas, que es lo que espera cualquiera.

// GrupoDestino es un grupo al que va dirigida una publicación, tal como lo
// necesita el panel para pintar la etiqueta "Para: Sala Maternal" y para
// filtrar el listado.
type GrupoDestino struct {
	ID     int    `json:"id"`
	Nombre string `json:"nombre"`
}

// validarGruposDeGuarderia convierte los ids que mandó el panel a enteros y
// confirma que TODOS pertenezcan a esta guardería. Sin esta comprobación,
// alguien podría dirigir una circular al id de un grupo de otra guardería:
// no le mostraría datos ajenos (el listado del padre filtra por su propia
// guardería de todos modos), pero sí dejaría la publicación dirigida a un
// grupo inexistente para él, o sea invisible para todos, sin ningún aviso.
//
// Regresa la lista vacía cuando no se eligió ningún grupo, que es como el
// llamador distingue "va para todas las familias".
func (s *Server) validarGruposDeGuarderia(idsTexto []string, gID any) ([]int, error) {
	if len(idsTexto) == 0 {
		return nil, nil
	}

	ids := make([]int, 0, len(idsTexto))
	for _, texto := range idsTexto {
		id, err := strconv.Atoi(texto)
		if err != nil {
			return nil, fmt.Errorf("grupo inválido")
		}
		ids = append(ids, id)
	}

	var encontrados int
	if err := s.DB.QueryRow(
		`SELECT COUNT(*) FROM grupos WHERE guarderia_id = $1 AND id = ANY($2)`,
		gID, pq.Array(ids),
	).Scan(&encontrados); err != nil {
		return nil, err
	}
	if encontrados != len(ids) {
		return nil, fmt.Errorf("alguno de los grupos elegidos no existe en tu guardería")
	}
	return ids, nil
}

// guardarGruposDestino escribe la tabla puente y deja para_todos en false.
// Va dentro de la misma transacción que crea la publicación: una circular
// dirigida que se guardara con para_todos = true por un error a media
// escritura se le mostraría a toda la guardería, justo lo contrario de lo
// que pidió quien la publicó.
func guardarGruposDestino(tx *sql.Tx, tabla, columna string, publicacionID int, grupos []int) error {
	if len(grupos) == 0 {
		return nil
	}
	for _, grupoID := range grupos {
		if _, err := tx.Exec(
			fmt.Sprintf(`INSERT INTO %s (%s, grupo_id) VALUES ($1, $2)`, tabla, columna),
			publicacionID, grupoID,
		); err != nil {
			return err
		}
	}
	return nil
}

// condicionVisibleParaPadre arma el fragmento de SQL que decide si una
// publicación le toca a un papá. `alias` es el alias de la tabla principal
// en la consulta que lo usa, y los dos números son las posiciones de los
// parámetros de la publicación y del padre.
//
// Se genera en vez de escribirse a mano en cada consulta porque circulares y
// encuestas tienen que aplicar EXACTAMENTE el mismo criterio: si uno de los
// dos se desviara, un papá vería en su lista una encuesta que no le tocaba.
func condicionVisibleParaPadre(alias, tabla, columna string, posPadre int) string {
	return fmt.Sprintf(`(%[1]s.para_todos OR EXISTS (
        SELECT 1 FROM %[2]s pg
        JOIN hijos h ON h.grupo_id = pg.grupo_id AND h.activo
        JOIN tutor_hijos th ON th.hijo_id = h.id
        WHERE pg.%[3]s = %[1]s.id AND th.padre_id = $%[4]d
    ))`, alias, tabla, columna, posPadre)
}

// contarFamiliasDestino cuenta a cuántas familias les toca una publicación:
// todas las de la guardería si va para todas, o solo las que tienen un hijo
// activo en los grupos elegidos. Sin esto, el panel diría "3 de 40 familias"
// en una circular que en realidad solo se le mandó a un salón de 8 -- un
// avance del 38% se leería como del 7,5%.
func contarFamiliasDestino(alias, tabla, columna string, posGuarderia int) string {
	return fmt.Sprintf(`CASE WHEN %[1]s.para_todos
        THEN (SELECT COUNT(*) FROM padres p WHERE p.guarderia_id = $%[4]d)
        ELSE (SELECT COUNT(DISTINCT th.padre_id)
              FROM %[2]s pg
              JOIN hijos h ON h.grupo_id = pg.grupo_id AND h.activo
              JOIN tutor_hijos th ON th.hijo_id = h.id
              WHERE pg.%[3]s = %[1]s.id)
    END`, alias, tabla, columna, posGuarderia)
}

// gruposDePublicaciones trae, de un solo viaje, los grupos de todas las
// publicaciones de un listado -- una consulta por circular sería N+1 sobre
// una pantalla que carga 50.
func (s *Server) gruposDePublicaciones(tabla, columna string, ids []int) (map[int][]GrupoDestino, error) {
	porPublicacion := map[int][]GrupoDestino{}
	if len(ids) == 0 {
		return porPublicacion, nil
	}

	rows, err := s.DB.Query(fmt.Sprintf(
		`SELECT pg.%[1]s, g.id, g.nombre
         FROM %[2]s pg
         JOIN grupos g ON g.id = pg.grupo_id
         WHERE pg.%[1]s = ANY($1)
         ORDER BY g.nombre ASC`, columna, tabla), pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var publicacionID int
		var g GrupoDestino
		if err := rows.Scan(&publicacionID, &g.ID, &g.Nombre); err != nil {
			continue
		}
		porPublicacion[publicacionID] = append(porPublicacion[publicacionID], g)
	}
	return porPublicacion, nil
}
