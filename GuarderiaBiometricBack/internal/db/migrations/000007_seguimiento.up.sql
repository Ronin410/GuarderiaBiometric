CREATE TABLE IF NOT EXISTS seguimiento_diario (
    id SERIAL PRIMARY KEY,
    hijo_id INTEGER REFERENCES hijos(id) ON DELETE CASCADE,
    guarderia_id INTEGER REFERENCES guarderias(id) ON DELETE CASCADE,
    fecha DATE DEFAULT CURRENT_DATE,
    desayuno VARCHAR(20) DEFAULT 'pendiente', -- 'no_comio', 'poco', 'todo'
    comida VARCHAR(20) DEFAULT 'pendiente',
    merienda VARCHAR(20) DEFAULT 'pendiente',
    esfinter VARCHAR(50),
    foto_url TEXT,
    observaciones TEXT,
    UNIQUE(hijo_id, fecha) -- Evita duplicados para el mismo niño el mismo día
);

CREATE TABLE IF NOT EXISTS fotos_seguimiento (
    id SERIAL PRIMARY KEY,
    seguimiento_id INT REFERENCES seguimiento_diario(id),
    url TEXT NOT NULL,
    creado_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
