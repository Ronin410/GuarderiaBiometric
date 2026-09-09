-- Base de conocimiento para el chat de soporte con IA (RAG): fragmentos de
-- los manuales de usuario (manual.html / manual-papa.html) con su embedding
-- ya calculado, para que el asistente busque por significado -- no por
-- texto exacto -- antes de contestar una pregunta de uso de la plataforma.
-- Se llena con cmd/ingest-conocimiento, nunca desde la aplicación en sí.
--
-- pgvector: si esta instancia de Postgres no trae ya la extensión
-- disponible, este CREATE EXTENSION falla y las migraciones siguientes NO
-- corren -- en Render hay que habilitarla desde el dashboard de la base de
-- datos (o pedírselo a soporte de Render) antes del primer deploy con esta
-- migración.
CREATE EXTENSION IF NOT EXISTS vector;

-- voyage-3.5-lite con output_dimension=1024 (ver internal/ia/voyage.go) --
-- si el modelo de embeddings cambia más adelante, esta columna necesita su
-- propia migración: el tamaño del vector es fijo por columna en pgvector.
CREATE TABLE IF NOT EXISTS fragmentos_conocimiento (
    id SERIAL PRIMARY KEY,
    contenido TEXT NOT NULL,
    embedding vector(1024) NOT NULL,
    -- De qué manual/sección viene ("manual.html › Recepción › ..."), para
    -- mostrarlo como referencia y para poder borrar/reemplazar solo los
    -- fragmentos de un archivo específico al reindexar (ver
    -- cmd/ingest-conocimiento).
    fuente TEXT NOT NULL,
    creado_en TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- HNSW en vez de IVFFlat: no necesita "entrenarse" con datos ya cargados
-- (un IVFFlat construido sobre una tabla vacía busca mal hasta que se
-- reconstruye) y, con el tamaño de esta base -- unos cuantos cientos de
-- fragmentos, la documentación de la plataforma -- su costo extra de
-- memoria no importa.
CREATE INDEX IF NOT EXISTS idx_fragmentos_conocimiento_embedding
    ON fragmentos_conocimiento USING hnsw (embedding vector_cosine_ops);

-- Distingue, dentro del chat de soporte, una respuesta que escribió el
-- asistente de IA (autor_rol sigue siendo 'plataforma' -- ver
-- chat_soporte.go / ia_soporte.go) de una que escribió el dueño de la
-- plataforma en persona.
ALTER TABLE mensajes_soporte ADD COLUMN IF NOT EXISTS generado_por_ia BOOLEAN NOT NULL DEFAULT false;
