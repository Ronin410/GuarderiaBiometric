// Comando ingest-conocimiento: llena/actualiza fragmentos_conocimiento (la
// base de conocimiento del chat de soporte con IA, ver
// internal/server/ia_soporte.go) a partir de los manuales de usuario HTML
// (GuarderiaBiometricFront/public/manual.html y manual-papa.html).
//
// No corre como parte del servidor -- se ejecuta a mano cada vez que los
// manuales cambian de contenido:
//
//	go run ./cmd/ingest-conocimiento \
//	    -db "$DATABASE_URL" -voyage-key "$VOYAGE_API_KEY" \
//	    ../GuarderiaBiometricFront/public/manual.html \
//	    ../GuarderiaBiometricFront/public/manual-papa.html
//
// Por archivo: reemplaza TODOS los fragmentos que ya existan de ese mismo
// archivo (por nombre, ver reindexarArchivo) e inserta los que encuentre
// ahora -- reindexar dos veces seguidas con el mismo manual no duplica
// nada.
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	_ "github.com/lib/pq"

	"biometrico/internal/ia"
)

// reBloque encuentra, EN ORDEN dentro del documento, cada encabezado de
// sección (h2), subsección (h3.cat-title) y tarjeta individual (article.manual)
// de los manuales -- ver GuarderiaBiometricFront/public/manual.html /
// manual-papa.html. Exactamente uno de los tres grupos viene lleno según
// cuál haya hecho match.
var reBloque = regexp.MustCompile(`(?s)<h2[^>]*>(.*?)</h2>|<h3 class="cat-title"[^>]*>(.*?)</h3>|<article class="manual">(.*?)</article>`)
var reH4 = regexp.MustCompile(`(?s)<h4[^>]*>(.*?)</h4>`)
var reEtiqueta = regexp.MustCompile(`<[^>]+>`)
var reEspacios = regexp.MustCompile(`\s+`)

type fragmentoParaIngestar struct {
	Contenido string
	Fuente    string
}

// limpiarTexto quita las etiquetas HTML, desescapa entidades (&aacute;,
// &amp;, etc.) y colapsa los espacios en blanco -- deja el texto plano tal
// como lo leería alguien en pantalla, sin marcado.
func limpiarTexto(s string) string {
	s = reEtiqueta.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	s = reEspacios.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// extraerFragmentos lee un manual HTML y regresa un fragmento por cada
// <article class="manual"> que encuentre, con su "fuente" armada como
// "archivo.html › Sección › Subsección › Título de la tarjeta" -- así el
// asistente puede citar de dónde sacó la respuesta, y reindexar un archivo
// se puede limpiar buscando por su nombre (ver reindexarArchivo).
func extraerFragmentos(rutaArchivo string) ([]fragmentoParaIngestar, error) {
	datos, err := os.ReadFile(rutaArchivo)
	if err != nil {
		return nil, err
	}
	contenidoHTML := string(datos)
	nombreArchivo := filepath.Base(rutaArchivo)

	var fragmentos []fragmentoParaIngestar
	var h2Actual, h3Actual string

	for _, m := range reBloque.FindAllStringSubmatch(contenidoHTML, -1) {
		switch {
		case m[1] != "":
			h2Actual = limpiarTexto(m[1])
			h3Actual = ""
		case m[2] != "":
			h3Actual = limpiarTexto(m[2])
		case m[3] != "":
			articulo := m[3]
			var h4 string
			if hm := reH4.FindStringSubmatch(articulo); hm != nil {
				h4 = limpiarTexto(hm[1])
			}
			contenido := limpiarTexto(articulo)
			if contenido == "" {
				continue
			}

			partesFuente := []string{nombreArchivo}
			for _, p := range []string{h2Actual, h3Actual, h4} {
				if p != "" {
					partesFuente = append(partesFuente, p)
				}
			}
			fragmentos = append(fragmentos, fragmentoParaIngestar{
				Contenido: contenido,
				Fuente:    strings.Join(partesFuente, " › "),
			})
		}
	}
	return fragmentos, nil
}

// reindexarArchivo borra los fragmentos existentes de ESTE archivo (por
// prefijo de "fuente", que siempre empieza con el nombre del archivo) y los
// vuelve a insertar con embeddings frescos -- deja la base al día con el
// contenido actual del manual, sin ir acumulando versiones viejas de
// secciones que ya cambiaron o se borraron.
func reindexarArchivo(db *sql.DB, voyageKey, ruta string) error {
	fragmentos, err := extraerFragmentos(ruta)
	if err != nil {
		return fmt.Errorf("no se pudo leer/parsear: %w", err)
	}
	if len(fragmentos) == 0 {
		return fmt.Errorf(`no se encontró ningún <article class="manual"> -- ¿es el archivo correcto?`)
	}

	nombreArchivo := filepath.Base(ruta)
	fmt.Printf("%s: %d fragmentos encontrados, calculando embeddings...\n", nombreArchivo, len(fragmentos))

	textos := make([]string, len(fragmentos))
	for i, f := range fragmentos {
		textos[i] = f.Contenido
	}

	embeddings, err := ia.GenerarEmbeddings(voyageKey, textos, "document")
	if err != nil {
		return fmt.Errorf("voyage: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM fragmentos_conocimiento WHERE fuente LIKE $1`, nombreArchivo+"%"); err != nil {
		return fmt.Errorf("no se pudieron borrar los fragmentos anteriores: %w", err)
	}

	for i, f := range fragmentos {
		if embeddings[i] == nil {
			return fmt.Errorf("voyage no regresó un embedding para el fragmento %d (%q)", i, f.Fuente)
		}
		if _, err := tx.Exec(
			`INSERT INTO fragmentos_conocimiento (contenido, embedding, fuente) VALUES ($1, $2::vector, $3)`,
			f.Contenido, ia.VectorLiteral(embeddings[i]), f.Fuente,
		); err != nil {
			return fmt.Errorf("no se pudo insertar el fragmento %q: %w", f.Fuente, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("%s: listo, %d fragmentos actualizados.\n", nombreArchivo, len(fragmentos))
	return nil
}

func main() {
	dbURL := flag.String("db", os.Getenv("DATABASE_URL"), "Cadena de conexión a Postgres (o variable DATABASE_URL)")
	voyageKey := flag.String("voyage-key", os.Getenv("VOYAGE_API_KEY"), "API key de Voyage AI (o variable VOYAGE_API_KEY)")
	flag.Parse()

	archivos := flag.Args()
	if len(archivos) == 0 {
		fmt.Fprintln(os.Stderr, "uso: ingest-conocimiento [-db ...] [-voyage-key ...] archivo1.html archivo2.html ...")
		os.Exit(1)
	}
	if *dbURL == "" {
		fmt.Fprintln(os.Stderr, "falta -db o la variable DATABASE_URL")
		os.Exit(1)
	}
	if *voyageKey == "" {
		fmt.Fprintln(os.Stderr, "falta -voyage-key o la variable VOYAGE_API_KEY")
		os.Exit(1)
	}

	db, err := sql.Open("postgres", *dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "no se pudo abrir la conexión: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "no se pudo conectar a Postgres: %v\n", err)
		os.Exit(1)
	}

	for _, ruta := range archivos {
		if err := reindexarArchivo(db, *voyageKey, ruta); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", ruta, err)
			os.Exit(1)
		}
	}
}
