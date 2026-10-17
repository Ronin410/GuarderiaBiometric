-- "Los niños estarán asignados a un grupo en específico" (ej. salón, rango
-- de edad). grupo_id es NULLABLE a propósito: un niño puede existir sin
-- grupo asignado todavía (recién inscrito) sin que eso bloquee nada.
CREATE TABLE IF NOT EXISTS grupos (
    id SERIAL PRIMARY KEY,
    guarderia_id INTEGER REFERENCES guarderias(id) ON DELETE CASCADE,
    nombre VARCHAR(80) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_grupos_guarderia ON grupos (guarderia_id);

ALTER TABLE hijos ADD COLUMN IF NOT EXISTS grupo_id INTEGER REFERENCES grupos(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_hijos_grupo ON hijos (grupo_id);
