CREATE TABLE IF NOT EXISTS tutor_hijos (
    padre_id INTEGER REFERENCES padres(id) ON DELETE CASCADE,
    hijo_id INTEGER REFERENCES hijos(id) ON DELETE CASCADE,
    guarderia_id INTEGER REFERENCES guarderias(id) ON DELETE CASCADE,
    PRIMARY KEY (padre_id, hijo_id)
);
