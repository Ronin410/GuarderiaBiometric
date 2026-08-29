-- "A la guardería también le llegarán notificaciones tanto de los mensajes
-- como de pedidos de comedor y otras cosas" -- hasta ahora
-- push_subscripciones solo servía para papás (padre_id NOT NULL). Se
-- agrega personal_id para que el staff/admin también pueda suscribirse,
-- y padre_id se vuelve opcional: cada fila trae UNO de los dos, nunca
-- ambos ni ninguno (ver el INSERT de POST /push/suscribir en push.go, que
-- decide cuál llenar según el rol de quien se suscribe).
--
-- personal_id NO lleva FK a usuarios(id) -- mismo criterio que padre_id/
-- autor_id ya usan en mensajes_chat (ver 000023_chat): "usuarios" vive
-- conceptualmente en la base de auth (DATABASE_URL_AUTH).
ALTER TABLE push_subscripciones ALTER COLUMN padre_id DROP NOT NULL;
ALTER TABLE push_subscripciones ADD COLUMN IF NOT EXISTS personal_id INTEGER;
CREATE INDEX IF NOT EXISTS idx_push_personal ON push_subscripciones(personal_id);
