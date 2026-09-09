-- Circulares: avisos que el admin o staff mandan a TODOS los padres de la
-- guardería (inscripciones, eventos, cierres, etc.) -- no van ligadas a un
-- niño en particular, a diferencia de la bitácora.
CREATE TABLE IF NOT EXISTS circulares (
    id SERIAL PRIMARY KEY,
    guarderia_id INTEGER REFERENCES guarderias(id) ON DELETE CASCADE,
    titulo VARCHAR(150) NOT NULL,
    contenido TEXT NOT NULL,
    creado_por INTEGER,
    creado_en TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_circulares_guarderia_fecha ON circulares (guarderia_id, creado_en DESC);
