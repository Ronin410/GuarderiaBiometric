package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"

	"biometrico/internal/ia"
)

// ia_soporte.go -- "Chat de soporte con RAG": el asistente de IA intenta
// responder solo, ANTES de avisarle al dueño de la plataforma, usando la
// documentación real de los manuales de usuario (manual.html/
// manual-papa.html, indexados con cmd/ingest-conocimiento) como única
// fuente de verdad. Único call site: intentarRespuestaAutomaticaSoporte se
// llama desde handleEnviarMensajeSoporte en chat_soporte.go, y solo cuando
// RAGSoporteHabilitado() es true.
//
// A propósito NO se conecta al formulario de prospectos (gente sin cuenta
// todavía, en la página de presentación, ver handleEnviarMensajeProspecto
// en chat_soporte.go): esa conversación es de ventas, no de "cómo uso la
// plataforma" -- la base de conocimiento (los manuales de usuario) no tiene
// nada que contestarles, y ese tipo de charla se beneficia más de un humano
// real respondiendo.
const (
	// limiteFragmentosIA -- cuántos fragmentos como máximo se le pasan al
	// modelo como contexto. Suficiente para cubrir una pregunta que toque
	// más de una sección del manual, sin inflar el prompt de más.
	limiteFragmentosIA = 5

	// umbralSimilitudIA -- por debajo de esto, el fragmento más parecido
	// que encontró la búsqueda ya no cuenta como "relevante" y se escala a
	// soporte humano sin ni llamar al modelo de lenguaje (ahorra esa
	// llamada, y evita que conteste con contexto que no tiene nada que
	// ver con la pregunta). La similitud es 1 - distancia de coseno: 1 =
	// idéntico, 0 = sin relación. Es un punto de partida razonable, no un
	// número medido -- vale la pena afinarlo con preguntas reales una vez
	// que el chat lleve tráfico de verdad.
	umbralSimilitudIA = 0.5

	// modeloIASoporte -- Haiku 4.5: rápido y económico, apropiado para una
	// respuesta corta de soporte contra contexto ya acotado por la
	// búsqueda. No hace falta un modelo de razonamiento más caro para
	// resumir 3-5 fragmentos de manual y contestar una pregunta de uso.
	modeloIASoporte = "claude-haiku-4-5"
)

const mensajeSinContextoIA = "No tengo información suficiente sobre eso en la documentación de Pasitos. Ya le avisé al equipo de soporte para que te ayude directamente -- en un momento te responden por aquí."

const sistemaIASoporte = `Eres el asistente de soporte de Pasitos, una plataforma de administración de guarderías (reconocimiento facial para entrada/salida, bitácora diaria, pagos, menú semanal, encuestas y chat entre guardería y familias).

Tu única función es responder dudas de USO de la plataforma (a papás, maestras o directoras) usando EXCLUSIVAMENTE los fragmentos de documentación que te comparte el usuario a continuación -- nunca inventes pantallas, botones o pasos que no aparezcan ahí.

Si los fragmentos no traen la respuesta, dilo con claridad en vez de adivinar o improvisar.

Responde en español de México, breve y directo (2 a 5 líneas), como si le explicaras a alguien sin conocimientos técnicos.

Escribe en texto plano: la burbuja del chat muestra tu respuesta tal cual, sin interpretar formato. No uses asteriscos para negritas ni almohadillas para títulos -- salen impresos y se ven como un error. Si necesitas enumerar pasos, usa "1." "2." "3." al inicio del renglón, que sí se lee bien.`

type fragmentoConocimiento struct {
	Contenido string
	Fuente    string
	Similitud float64
}

// intentarRespuestaAutomaticaSoporte -- corre SIEMPRE en su propia
// goroutine (ver handleEnviarMensajeSoporte), nunca debe bloquear la
// respuesta HTTP de "mensaje enviado". Si no hay contexto relevante, o
// CUALQUIER paso falla (búsqueda, modelo, base de datos), cae de vuelta al
// aviso normal a la plataforma -- "sin intervención humana" no debe
// significar "el mensaje se pierde si la IA truena": el dueño de la
// plataforma siempre se entera cuando el asistente no pudo responder.
func (s *Server) intentarRespuestaAutomaticaSoporte(convID int, pregunta, etiquetaRol string) {
	fragmentos, err := s.buscarFragmentosRelevantes(pregunta)
	if err != nil {
		s.logError(nil, "intentarRespuestaAutomaticaSoporte: error buscando contexto", err, "conversacion_id", convID)
		s.notificarPlataformaNuevoMensajeSoporteDeConversacion(convID, etiquetaRol)
		return
	}

	if len(fragmentos) == 0 || fragmentos[0].Similitud < umbralSimilitudIA {
		if err := s.insertarMensajeSoporteIA(convID, mensajeSinContextoIA); err != nil {
			s.logError(nil, "intentarRespuestaAutomaticaSoporte: no se pudo guardar el aviso de 'sin contexto'", err, "conversacion_id", convID)
		}
		s.notificarPlataformaNuevoMensajeSoporteDeConversacion(convID, etiquetaRol)
		return
	}

	respuesta, err := s.generarRespuestaIA(pregunta, fragmentos)
	if err != nil {
		s.logError(nil, "intentarRespuestaAutomaticaSoporte: error generando la respuesta", err, "conversacion_id", convID)
		s.notificarPlataformaNuevoMensajeSoporteDeConversacion(convID, etiquetaRol)
		return
	}

	if err := s.insertarMensajeSoporteIA(convID, respuesta); err != nil {
		s.logError(nil, "intentarRespuestaAutomaticaSoporte: no se pudo guardar la respuesta", err, "conversacion_id", convID)
		s.notificarPlataformaNuevoMensajeSoporteDeConversacion(convID, etiquetaRol)
	}
	// Éxito: no se notifica al dueño de la plataforma -- ese es el punto
	// de automatizar esto. La conversación sigue viéndose completa en su
	// inbox de /plataforma cuando quiera revisarla, solo sin interrumpirlo
	// con un push por algo que el asistente ya resolvió solo.
}

// buscarFragmentosRelevantes calcula el embedding de la pregunta y trae los
// fragmentos más parecidos por similitud de coseno (pgvector, operador
// <=>). Regresa ordenado del más al menos parecido.
func (s *Server) buscarFragmentosRelevantes(pregunta string) ([]fragmentoConocimiento, error) {
	embeddings, err := ia.GenerarEmbeddings(s.VoyageAPIKey, []string{pregunta}, "query")
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 || embeddings[0] == nil {
		return nil, fmt.Errorf("voyage no regresó un embedding para la pregunta")
	}

	literal := ia.VectorLiteral(embeddings[0])
	rows, err := s.DB.Query(
		`SELECT contenido, fuente, 1 - (embedding <=> $1::vector) AS similitud
         FROM fragmentos_conocimiento
         ORDER BY embedding <=> $1::vector
         LIMIT $2`,
		literal, limiteFragmentosIA,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fragmentos []fragmentoConocimiento
	for rows.Next() {
		var f fragmentoConocimiento
		if err := rows.Scan(&f.Contenido, &f.Fuente, &f.Similitud); err != nil {
			continue
		}
		fragmentos = append(fragmentos, f)
	}
	return fragmentos, nil
}

// generarRespuestaIA le pasa los fragmentos recuperados + la pregunta a
// Claude y regresa el texto de la respuesta.
func (s *Server) generarRespuestaIA(pregunta string, fragmentos []fragmentoConocimiento) (string, error) {
	var contexto strings.Builder
	for i, f := range fragmentos {
		fmt.Fprintf(&contexto, "[%d] (fuente: %s)\n%s\n\n", i+1, f.Fuente, f.Contenido)
	}

	mensaje := fmt.Sprintf("Fragmentos de documentación relevantes:\n\n%sPregunta del usuario: %s", contexto.String(), pregunta)

	resp, err := s.AnthropicClient.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     modeloIASoporte,
		MaxTokens: 500,
		System:    []anthropic.TextBlockParam{{Text: sistemaIASoporte}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(mensaje)),
		},
	})
	if err != nil {
		return "", err
	}

	var texto strings.Builder
	for _, block := range resp.Content {
		if b, ok := block.AsAny().(anthropic.TextBlock); ok {
			texto.WriteString(b.Text)
		}
	}
	if texto.Len() == 0 {
		return "", fmt.Errorf("claude no regresó texto en la respuesta")
	}
	return texto.String(), nil
}

// insertarMensajeSoporteIA guarda una respuesta del asistente de IA --
// mismo criterio que insertarMensajeSoporte (chat_soporte.go), pero
// separada a propósito en vez de agregarle un parámetro a esa función: así
// ningún caller existente (todos los mensajes humanos, ya cubiertos por
// pruebas) cambia su firma ni sus argumentos esperados.
func (s *Server) insertarMensajeSoporteIA(convID any, contenido string) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`INSERT INTO mensajes_soporte (conversacion_id, autor_rol, contenido, generado_por_ia) VALUES ($1, 'plataforma', $2, true)`,
		convID, contenido,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE conversaciones_soporte SET actualizado_en = now() WHERE id = $1`, convID); err != nil {
		return err
	}
	return tx.Commit()
}
