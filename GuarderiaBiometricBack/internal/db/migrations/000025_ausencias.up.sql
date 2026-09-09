-- ausencias_planificadas guarda los días en que un padre avisó que su hijo
-- no asistirá -- "Planificación de ausencias por parte de los padres" del
-- PDF de referencia. Un registro por día (no por rango) para que consultar
-- "quién falta hoy" sea un simple WHERE fecha = $1; un rango de varios días
-- se reporta como varias filas (ver handleCrearAusencia).
CREATE TABLE IF NOT EXISTS ausencias_planificadas (
    id SERIAL PRIMARY KEY,
    hijo_id INTEGER NOT NULL REFERENCES hijos(id) ON DELETE CASCADE,
    guarderia_id INTEGER REFERENCES guarderias(id) ON DELETE CASCADE,
    fecha DATE NOT NULL,
    motivo TEXT,
    creado_por INTEGER NOT NULL,
    creado_en TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (hijo_id, fecha)
);

CREATE INDEX IF NOT EXISTS idx_ausencias_fecha ON ausencias_planificadas (guarderia_id, fecha);
CREATE INDEX IF NOT EXISTS idx_ausencias_hijo ON ausencias_planificadas (hijo_id);
