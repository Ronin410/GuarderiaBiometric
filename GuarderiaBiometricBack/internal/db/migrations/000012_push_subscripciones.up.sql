-- padre_id guarda el mismo valor que ya usa /padre/:id/hijos (user_id del token
-- para cuentas rol "papa"), por eso no lleva FK a padres(id).
CREATE TABLE IF NOT EXISTS push_subscripciones (
    id SERIAL PRIMARY KEY,
    padre_id INTEGER NOT NULL,
    guarderia_id INTEGER REFERENCES guarderias(id) ON DELETE CASCADE,
    endpoint TEXT NOT NULL UNIQUE,
    p256dh TEXT NOT NULL,
    auth TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_push_padre ON push_subscripciones(padre_id);
