-- encuestas -- "Encuestas para familias" del PDF de referencia. Cada
-- encuesta tiene una o más preguntas; una pregunta es de opción múltiple
-- (una sola respuesta entre varias opciones) o de texto libre. Las
-- opciones se guardan como JSON dentro de un TEXT (no TEXT[] de Postgres)
-- para no depender de un tipo específico del driver -- se marshalea /
-- desmarshalea en Go, igual que el resto de la app.
CREATE TABLE IF NOT EXISTS encuestas (
    id SERIAL PRIMARY KEY,
    guarderia_id INTEGER REFERENCES guarderias(id) ON DELETE CASCADE,
    titulo VARCHAR(255) NOT NULL,
    descripcion TEXT,
    activa BOOLEAN NOT NULL DEFAULT true,
    creado_por INTEGER NOT NULL,
    creado_en TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS encuesta_preguntas (
    id SERIAL PRIMARY KEY,
    encuesta_id INTEGER NOT NULL REFERENCES encuestas(id) ON DELETE CASCADE,
    texto TEXT NOT NULL,
    tipo VARCHAR(20) NOT NULL DEFAULT 'texto_libre' CHECK (tipo IN ('opcion_multiple', 'texto_libre')),
    opciones TEXT,
    orden INTEGER NOT NULL DEFAULT 0
);

-- padre_id NO lleva FK a padres(id) a propósito -- mismo criterio ya usado
-- en push_subscripciones/mensajes_chat: guarda el user_id del JWT
-- (usuarios.id), que por convención coincide con padres.id para cuentas
-- "papa".
CREATE TABLE IF NOT EXISTS encuesta_respuestas (
    id SERIAL PRIMARY KEY,
    pregunta_id INTEGER NOT NULL REFERENCES encuesta_preguntas(id) ON DELETE CASCADE,
    padre_id INTEGER NOT NULL,
    respuesta TEXT NOT NULL,
    creado_en TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (pregunta_id, padre_id)
);

CREATE INDEX IF NOT EXISTS idx_encuesta_preguntas_encuesta ON encuesta_preguntas (encuesta_id);
CREATE INDEX IF NOT EXISTS idx_encuesta_respuestas_pregunta ON encuesta_respuestas (pregunta_id);
