-- Chat privado padres<->maestros, para no depender de WhatsApp (Meta) para
-- la mensajería. Una conversación por familia (padre_id) con "la guardería"
-- en general -- cualquier staff/admin puede leer y responder, no hay
-- asignación de un maestro específico por niño en el modelo actual.
--
-- padre_id/autor_id NO llevan FK a usuarios(id) -- mismo criterio ya usado
-- en push_subscripciones/logs_acceso/horarios_personal: "usuarios" vive
-- conceptualmente en la base de auth (DATABASE_URL_AUTH).
CREATE TABLE IF NOT EXISTS mensajes_chat (
    id SERIAL PRIMARY KEY,
    guarderia_id INTEGER REFERENCES guarderias(id) ON DELETE CASCADE,
    padre_id INTEGER NOT NULL,
    autor_id INTEGER NOT NULL,
    autor_rol VARCHAR(10) NOT NULL CHECK (autor_rol IN ('papa', 'staff', 'admin')),
    contenido TEXT NOT NULL,
    -- "leído por el otro lado" de la conversación: si autor_rol='papa', leido
    -- se refiere a si el staff ya lo vio; si autor_rol IN ('staff','admin'),
    -- a si el papá ya lo vio.
    leido BOOLEAN NOT NULL DEFAULT false,
    creado_en TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_mensajes_chat_conversacion ON mensajes_chat (guarderia_id, padre_id, creado_en);
