-- eventos_calendario -- "Calendario escolar" del PDF de referencia: fechas
-- importantes del centro (suspensiones, actividades, juntas, vacaciones)
-- visibles para staff y padres. fecha_fin es opcional -- NULL significa
-- evento de un solo día.
CREATE TABLE IF NOT EXISTS eventos_calendario (
    id SERIAL PRIMARY KEY,
    guarderia_id INTEGER REFERENCES guarderias(id) ON DELETE CASCADE,
    titulo VARCHAR(255) NOT NULL,
    descripcion TEXT,
    fecha_inicio DATE NOT NULL,
    fecha_fin DATE,
    tipo VARCHAR(20) NOT NULL DEFAULT 'evento' CHECK (tipo IN ('evento', 'suspension', 'vacaciones', 'junta')),
    creado_por INTEGER NOT NULL,
    creado_en TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_calendario_guarderia_fecha ON eventos_calendario (guarderia_id, fecha_inicio);
