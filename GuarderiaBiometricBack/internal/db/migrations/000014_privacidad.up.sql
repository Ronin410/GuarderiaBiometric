-- Aviso de Privacidad configurable por guardería, y registro de
-- consentimientos de tutores para el tratamiento de datos biométricos
-- propios y de sus hijos (LFPDPPP: datos sensibles / de menores).
ALTER TABLE guarderias ADD COLUMN IF NOT EXISTS aviso_privacidad_texto TEXT;
ALTER TABLE guarderias ADD COLUMN IF NOT EXISTS aviso_privacidad_version VARCHAR(20) NOT NULL DEFAULT 'v1';

-- padre_id es ON DELETE SET NULL (no CASCADE) a propósito: si el padre
-- ejerce después su derecho de cancelación (ARCO) y se borra su fila en
-- "padres", la evidencia de que sí hubo consentimiento en su momento debe
-- sobrevivir — por eso también se guarda padre_nombre_historico como copia.
CREATE TABLE IF NOT EXISTS consentimientos (
    id SERIAL PRIMARY KEY,
    padre_id INTEGER REFERENCES padres(id) ON DELETE SET NULL,
    padre_nombre_historico VARCHAR(100) NOT NULL,
    guarderia_id INTEGER REFERENCES guarderias(id) ON DELETE CASCADE,
    version_aviso VARCHAR(20) NOT NULL,
    aceptado_en TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ip VARCHAR(45)
);
CREATE INDEX IF NOT EXISTS idx_consentimientos_padre ON consentimientos(padre_id);
