-- circulares_lecturas registra qué padre ya leyó cada circular, para el
-- contador ("X de Y familias ya la leyeron") y el detalle de quién la leyó
-- que ve el staff -- justo el diferenciador que marca el PDF de referencia
-- ("verifica quiénes han leído tus mensajes").
--
-- padre_id NO lleva FK a padres(id) a propósito -- mismo criterio ya usado
-- en push_subscripciones/mensajes_chat: guarda el user_id del JWT
-- (usuarios.id), que por convención coincide con padres.id para cuentas
-- "papa" (ver handleHijosDePadre en hijos.go).
CREATE TABLE IF NOT EXISTS circulares_lecturas (
    id SERIAL PRIMARY KEY,
    circular_id INTEGER NOT NULL REFERENCES circulares(id) ON DELETE CASCADE,
    padre_id INTEGER NOT NULL,
    leido_en TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (circular_id, padre_id)
);

CREATE INDEX IF NOT EXISTS idx_circulares_lecturas_circular ON circulares_lecturas (circular_id);
