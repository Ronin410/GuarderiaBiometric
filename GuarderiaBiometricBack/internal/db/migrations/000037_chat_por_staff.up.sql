-- "Quiero que al papá le aparezcan los staff o administradores de la
-- guardería para los chats de los papás para que puedan escoger con quien
-- hablar" -- hasta ahora había una sola conversación por familia con "la
-- guardería" en general (cualquier staff/admin podía leer y responder
-- cualquier cosa). Ahora cada conversación queda dirigida a un miembro del
-- staff/admin en particular: cada quien ve y responde solo las suyas (el
-- admin, además, puede ver todas -- supervisión).
--
-- personal_id NO lleva FK a usuarios(id) -- mismo criterio que padre_id/
-- autor_id ya usan en esta tabla (ver el comentario de la migración
-- 000023_chat): "usuarios" vive conceptualmente en la base de auth
-- (DATABASE_URL_AUTH).
--
-- Los mensajes que ya existían antes de esta migración se quedan con
-- personal_id NULL -- son las conversaciones "generales" de antes, que ya
-- no se pueden seguir desde la app (no hay un contacto "general" en el
-- selector nuevo) pero no se borran, por si hace falta consultarlas
-- directo en la base de datos.
ALTER TABLE mensajes_chat ADD COLUMN IF NOT EXISTS personal_id INTEGER;
CREATE INDEX IF NOT EXISTS idx_mensajes_chat_personal ON mensajes_chat (guarderia_id, personal_id, padre_id, creado_en);
