// Package ia agrupa las llamadas a servicios externos de inteligencia
// artificial que usa el backend: por ahora solo los embeddings de Voyage AI
// (para el RAG del chat de soporte, ver internal/server/ia_soporte.go y
// cmd/ingest-conocimiento). Claude (las respuestas del chat en sí) se llama
// directo con el SDK oficial desde internal/server -- no hace falta
// envolverlo aquí, solo Voyage no tiene SDK de Go.
package ia

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	urlEmbeddingsVoyage = "https://api.voyageai.com/v1/embeddings"

	// ModeloEmbeddings -- el nivel "lite" de Voyage: alcanza de sobra para
	// buscar contra unos cuantos cientos de fragmentos de manuales de
	// usuario (no documentación técnica densa) y es barato, en línea con
	// el presupuesto que pedía la especificación original de esta función.
	ModeloEmbeddings = "voyage-3.5-lite"

	// DimensionesEmbeddings es fija por columna en pgvector (ver la
	// migración 000043_conocimiento_ia) -- si el modelo de embeddings
	// cambia más adelante, la tabla fragmentos_conocimiento necesita su
	// propia migración y reindexar todo desde cero con cmd/ingest-conocimiento.
	DimensionesEmbeddings = 1024
)

type solicitudEmbeddings struct {
	Input           []string `json:"input"`
	Model           string   `json:"model"`
	InputType       string   `json:"input_type"`
	OutputDimension int      `json:"output_dimension"`
}

type datoEmbedding struct {
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

type respuestaEmbeddings struct {
	Data []datoEmbedding `json:"data"`
}

// GenerarEmbeddings llama a la API de Voyage AI y regresa un embedding por
// cada texto de "textos", EN EL MISMO ORDEN (se reordena según el índice
// que regresa Voyage en cada elemento, no se asume que "data" siempre
// venga en el mismo orden que la petición).
//
// inputType debe ser "document" al indexar contenido (ver
// cmd/ingest-conocimiento) o "query" al buscar por una pregunta (ver
// internal/server/ia_soporte.go) -- Voyage optimiza el embedding distinto
// según cuál sea; usar el que no toca no falla, pero degrada la calidad de
// la búsqueda.
func GenerarEmbeddings(apiKey string, textos []string, inputType string) ([][]float32, error) {
	if len(textos) == 0 {
		return nil, nil
	}

	cuerpoJSON, err := json.Marshal(solicitudEmbeddings{
		Input:           textos,
		Model:           ModeloEmbeddings,
		InputType:       inputType,
		OutputDimension: DimensionesEmbeddings,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, urlEmbeddingsVoyage, bytes.NewReader(cuerpoJSON))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	cliente := &http.Client{Timeout: 30 * time.Second}
	resp, err := cliente.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		cuerpo, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("voyage respondió %d: %s", resp.StatusCode, string(cuerpo))
	}

	var respuesta respuestaEmbeddings
	if err := json.NewDecoder(resp.Body).Decode(&respuesta); err != nil {
		return nil, err
	}

	embeddings := make([][]float32, len(textos))
	for _, d := range respuesta.Data {
		if d.Index >= 0 && d.Index < len(embeddings) {
			embeddings[d.Index] = d.Embedding
		}
	}
	return embeddings, nil
}

// VectorLiteral da formato al embedding como literal de pgvector
// ("[0.1,0.2,...]"). Este proyecto usa lib/pq, que a diferencia de pgx no
// trae un tipo nativo para pgvector -- el valor viaja como texto plano y se
// castea a vector del lado de Postgres ($1::vector en cada query que lo usa).
func VectorLiteral(embedding []float32) string {
	partes := make([]string, len(embedding))
	for i, v := range embedding {
		partes[i] = strconv.FormatFloat(float64(v), 'f', -1, 32)
	}
	return "[" + strings.Join(partes, ",") + "]"
}
