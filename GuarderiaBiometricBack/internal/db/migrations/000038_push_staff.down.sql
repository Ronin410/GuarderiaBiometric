DROP INDEX IF EXISTS idx_push_personal;
ALTER TABLE push_subscripciones DROP COLUMN IF EXISTS personal_id;
-- padre_id se deja nullable incluso al revertir: forzar NOT NULL de vuelta
-- fallaría si para entonces ya existen filas de staff con padre_id NULL, y
-- no hay forma de saber en qué estado quedó la tabla al momento del rollback.
