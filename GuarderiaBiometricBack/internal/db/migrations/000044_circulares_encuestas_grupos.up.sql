-- "Que si alguien quiere hacer una encuesta pueda escoger a quién va
-- dirigida, y de la misma forma las circulares": hasta ahora toda circular y
-- toda encuesta iban forzosamente a TODAS las familias de la guardería. Con
-- salones distintos (maternal, preescolar) eso obliga a redactar avisos
-- genéricos o a mandarle a todos algo que solo le toca a un grupo.
--
-- para_todos es una columna aparte de las tablas de grupos, en vez de
-- deducir "sin grupos = para todos" de la tabla puente: si un grupo se
-- borra, sus filas se van en cascada, y sin esta bandera una circular que
-- iba dirigida a UN salón se convertiría de golpe en una circular para toda
-- la guardería. Con para_todos = false explícito, en ese caso deja de verla
-- nadie -- que es el lado seguro por el que hay que equivocarse cuando se
-- trata de a quién se le muestra un aviso.
--
-- DEFAULT TRUE deja todo lo ya publicado exactamente como estaba.
ALTER TABLE circulares ADD COLUMN IF NOT EXISTS para_todos BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE encuestas  ADD COLUMN IF NOT EXISTS para_todos BOOLEAN NOT NULL DEFAULT TRUE;

CREATE TABLE IF NOT EXISTS circulares_grupos (
    circular_id INTEGER REFERENCES circulares(id) ON DELETE CASCADE,
    grupo_id    INTEGER REFERENCES grupos(id) ON DELETE CASCADE,
    PRIMARY KEY (circular_id, grupo_id)
);
CREATE INDEX IF NOT EXISTS idx_circulares_grupos_grupo ON circulares_grupos (grupo_id);

CREATE TABLE IF NOT EXISTS encuestas_grupos (
    encuesta_id INTEGER REFERENCES encuestas(id) ON DELETE CASCADE,
    grupo_id    INTEGER REFERENCES grupos(id) ON DELETE CASCADE,
    PRIMARY KEY (encuesta_id, grupo_id)
);
CREATE INDEX IF NOT EXISTS idx_encuestas_grupos_grupo ON encuestas_grupos (grupo_id);
