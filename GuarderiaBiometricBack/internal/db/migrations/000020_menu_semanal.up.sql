-- "Quiero un apartado donde el administrador y staff con permisos puedan ir
-- cargando el menú de la semana." Una fila por día (no por semana) es más
-- simple de consultar por rango y no obliga a decidir de antemano qué
-- días tiene la semana de cada guardería.
CREATE TABLE IF NOT EXISTS menu_semanal (
    id SERIAL PRIMARY KEY,
    guarderia_id INTEGER REFERENCES guarderias(id) ON DELETE CASCADE,
    fecha DATE NOT NULL,
    desayuno TEXT,
    comida TEXT,
    merienda TEXT,
    actualizado_en TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (guarderia_id, fecha)
);
CREATE INDEX IF NOT EXISTS idx_menu_semanal_guarderia_fecha ON menu_semanal (guarderia_id, fecha);
