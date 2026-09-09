-- pedidos_comedor guarda las EXCEPCIONES al comedor del día -- "Pedidos
-- para el comedor o catering" del PDF de referencia. Por defecto, todo
-- niño activo come las tres comidas del día; un padre solo necesita
-- reportar aquí cuando algo cambia (no desayuna porque ya comió en casa,
-- trae notas de alergias, etc.) -- así el conteo para cocina/catering sale
-- de "total de niños activos" menos las excepciones de cada comida, sin
-- que cada familia tenga que confirmar todos los días que "sí come".
CREATE TABLE IF NOT EXISTS pedidos_comedor (
    id SERIAL PRIMARY KEY,
    hijo_id INTEGER NOT NULL REFERENCES hijos(id) ON DELETE CASCADE,
    guarderia_id INTEGER REFERENCES guarderias(id) ON DELETE CASCADE,
    fecha DATE NOT NULL,
    desayuno BOOLEAN NOT NULL DEFAULT true,
    comida BOOLEAN NOT NULL DEFAULT true,
    merienda BOOLEAN NOT NULL DEFAULT true,
    notas TEXT,
    creado_por INTEGER NOT NULL,
    actualizado_en TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (hijo_id, fecha)
);

CREATE INDEX IF NOT EXISTS idx_pedidos_comedor_fecha ON pedidos_comedor (guarderia_id, fecha);
