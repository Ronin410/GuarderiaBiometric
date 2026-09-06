package server

import (
	"context"
	"encoding/json"
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

Respondes dos tipos de cosas:

1. Dudas de USO de la plataforma (a papás, maestras o directoras), usando EXCLUSIVAMENTE los fragmentos de documentación que te comparte el usuario -- nunca inventes pantallas, botones o pasos que no aparezcan ahí. Si los fragmentos no traen la respuesta, dilo con claridad en vez de adivinar o improvisar.

2. Cuando quien te escribe es un papá y pregunta por SU HIJO -- cómo le fue, qué comió, si durmió, a qué hora llegó o salió, si trae algún golpe -- usa la herramienta consultar_bitacora y contéstale con lo que de verdad dice la bitácora de ese día. NO le expliques dónde mirarlo: quiere saber cómo le fue, no cómo consultarlo. Si además le sirve saber dónde verlo, dilo en una línea al final, no como respuesta principal.

Al contar el día de un niño, ve al grano y en orden natural: cómo comió, si durmió, cualquier nota de la maestra, y las horas de entrada y salida. Menciona SIEMPRE un golpe reportado, aunque no te lo hayan preguntado. Lo que la bitácora marque como "sin capturar todavía" repórtalo como que la maestra todavía no lo llena, nunca como que el niño no comió.

Si la herramienta no devuelve datos de ese día, dilo tal cual (que todavía no llenan la bitácora o que no registra entrada) y sugiere preguntarle directamente a la guardería por el chat. Nunca inventes cómo estuvo un niño.

No tienes acceso a nada más de la guardería: ni pagos, ni expedientes, ni datos de otros niños. Si te preguntan por algo así, dilo y remite a la guardería.

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
// autorSoporte identifica a quien escribió, para saber si se le puede
// ofrecer al modelo la herramienta de bitácora. PadreID es no-nil solo
// cuando quien escribe es un papá con cuenta: es la única audiencia a la que
// se le consultan datos de niños, y siempre los suyos (ver ia_bitacora.go).
type autorSoporte struct {
	PadreID     any
	GuarderiaID any
}

func (s *Server) intentarRespuestaAutomaticaSoporte(convID int, pregunta, etiquetaRol string, autor autorSoporte) {
	fragmentos, err := s.buscarFragmentosRelevantes(pregunta)
	if err != nil {
		s.logError(nil, "intentarRespuestaAutomaticaSoporte: error buscando contexto", err, "conversacion_id", convID)
		s.notificarPlataformaNuevoMensajeSoporteDeConversacion(convID, etiquetaRol)
		return
	}

	// Sin contexto relevante en los manuales se escala a un humano... salvo
	// que quien escribe sea un papá: su pregunta puede no estar en ningún
	// manual y aun así tener respuesta en la bitácora de su hijo ("¿comió
	// bien hoy?"), que el modelo puede consultar con la herramienta.
	if len(fragmentos) == 0 || fragmentos[0].Similitud < umbralSimilitudIA {
		if autor.PadreID == nil {
			if err := s.insertarMensajeSoporteIA(convID, mensajeSinContextoIA); err != nil {
				s.logError(nil, "intentarRespuestaAutomaticaSoporte: no se pudo guardar el aviso de 'sin contexto'", err, "conversacion_id", convID)
			}
			s.notificarPlataformaNuevoMensajeSoporteDeConversacion(convID, etiquetaRol)
			return
		}
		fragmentos = nil
	}

	respuesta, err := s.generarRespuestaIA(pregunta, fragmentos, autor)
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
// Claude y regresa el texto de la respuesta. Cuando quien escribe es un papá
// se le ofrece además la herramienta de bitácora: el modelo decide si la
// pregunta la necesita, así que un "¿cómo instalo la app?" nunca provoca que
// salgan datos de un niño hacia la API.
func (s *Server) generarRespuestaIA(pregunta string, fragmentos []fragmentoConocimiento, autor autorSoporte) (string, error) {
	var contexto strings.Builder
	for i, f := range fragmentos {
		fmt.Fprintf(&contexto, "[%d] (fuente: %s)\n%s\n\n", i+1, f.Fuente, f.Contenido)
	}

	mensaje := "Pregunta del usuario: " + pregunta
	if contexto.Len() > 0 {
		mensaje = fmt.Sprintf("Fragmentos de documentación relevantes:\n\n%s%s", contexto.String(), mensaje)
	}

	params := anthropic.MessageNewParams{
		Model:     modeloIASoporte,
		MaxTokens: 500,
		System:    []anthropic.TextBlockParam{{Text: sistemaIASoporte}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(mensaje)),
		},
	}

	if autor.PadreID != nil {
		params.Tools = []anthropic.ToolUnionParam{{OfTool: &anthropic.ToolParam{
			Name:        nombreHerramientaBitacora,
			Description: anthropic.String(descripcionHerramientaBitacora),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]any{
					"fecha": map[string]any{
						"type":        "string",
						"description": "Día a consultar en formato YYYY-MM-DD. Omítelo para el día de hoy.",
					},
				},
			},
		}}}
	}

	resp, err := s.AnthropicClient.Messages.New(context.Background(), params)
	if err != nil {
		return "", err
	}

	// Una sola vuelta de herramienta: la consulta devuelve el día completo de
	// todos los hijos de esa cuenta, así que no hay nada que encadenar. Si el
	// modelo pidiera una segunda, se queda con lo que ya tiene en vez de
	// alargar la espera de alguien mirando los puntitos del chat.
	if resp.StopReason == anthropic.StopReasonToolUse {
		resp, err = s.responderConBitacora(params, resp, autor)
		if err != nil {
			return "", err
		}
	}

	return textoDeRespuesta(resp)
}

// responderConBitacora ejecuta la herramienta que pidió el modelo y le manda
// el resultado de vuelta para que redacte la respuesta final.
func (s *Server) responderConBitacora(params anthropic.MessageNewParams, resp *anthropic.Message, autor autorSoporte) (*anthropic.Message, error) {
	bloquesAsistente := []anthropic.ContentBlockParamUnion{}
	resultados := []anthropic.ContentBlockParamUnion{}

	for _, bloque := range resp.Content {
		switch b := bloque.AsAny().(type) {
		case anthropic.TextBlock:
			if strings.TrimSpace(b.Text) != "" {
				bloquesAsistente = append(bloquesAsistente, anthropic.NewTextBlock(b.Text))
			}
		case anthropic.ToolUseBlock:
			bloquesAsistente = append(bloquesAsistente, anthropic.NewToolUseBlock(b.ID, b.Input, b.Name))

			if b.Name != nombreHerramientaBitacora {
				// No debería pasar (es la única herramienta que se ofrece),
				// pero el protocolo exige un resultado por cada tool_use: sin
				// él la siguiente llamada falla con un 400.
				resultados = append(resultados, anthropic.NewToolResultBlock(b.ID, "Herramienta desconocida.", true))
				continue
			}

			var entrada struct {
				Fecha string `json:"fecha"`
			}
			// Un JSON que no parsea no es motivo para tumbar la respuesta:
			// se consulta el día de hoy, que es lo que se pregunta el 99% de
			// las veces.
			if err := json.Unmarshal(b.Input, &entrada); err != nil {
				s.logError(nil, "generarRespuestaIA: no se pudo leer la fecha que pidió el modelo", err)
			}

			texto, err := s.bitacoraDeHijosDePapa(autor.PadreID, autor.GuarderiaID, entrada.Fecha)
			if err != nil {
				s.logError(nil, "generarRespuestaIA: no se pudo consultar la bitácora", err)
				resultados = append(resultados, anthropic.NewToolResultBlock(b.ID, "No se pudo consultar la bitácora en este momento.", true))
				continue
			}
			resultados = append(resultados, anthropic.NewToolResultBlock(b.ID, texto, false))
		}
	}

	params.Messages = append(params.Messages,
		anthropic.NewAssistantMessage(bloquesAsistente...),
		anthropic.NewUserMessage(resultados...),
	)
	return s.AnthropicClient.Messages.New(context.Background(), params)
}

func textoDeRespuesta(resp *anthropic.Message) (string, error) {
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
