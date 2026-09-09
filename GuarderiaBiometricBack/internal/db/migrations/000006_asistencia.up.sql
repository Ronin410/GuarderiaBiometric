CREATE TABLE IF NOT EXISTS asistencia (
    id SERIAL PRIMARY KEY,
    padre_id INTEGER REFERENCES padres(id) ON DELETE CASCADE,
    hijo_id INTEGER REFERENCES hijos(id) ON DELETE CASCADE,
    guarderia_id INTEGER REFERENCES guarderias(id) ON DELETE CASCADE,
    fecha_hora TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    aseado BOOLEAN DEFAULT true,
    reporte_golpe BOOLEAN DEFAULT false,
    observaciones TEXT,
    tipo_movimiento VARCHAR(20) CHECK (tipo_movimiento IN ('ENTRADA', 'SALIDA', 'REGISTRO'))
);
