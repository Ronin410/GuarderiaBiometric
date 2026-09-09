DROP INDEX IF EXISTS idx_mensajes_chat_personal;
ALTER TABLE mensajes_chat DROP COLUMN IF EXISTS personal_id;
