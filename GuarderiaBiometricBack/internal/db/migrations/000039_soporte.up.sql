-- Chat de soporte: distinto del chat de mensajes_chat (que es
-- papás<->staff DENTRO de una guardería). Este es papás, staff/admin de
-- CUALQUIER guardería, y prospectos (visitantes sin cuenta todavía) <->
-- el dueño de la plataforma, quien responde desde /plataforma (protegido
-- por PLATFORM_ADMIN_KEY, ver middleware.RequirePlatformKey).
--
-- Se denormaliza nombre/email en la conversación en vez de resolverlos por
-- join en cada lectura: un prospecto no tiene fila en padres/usuarios que
-- referenciar, y así el listado de /plataforma/soporte/conversaciones no
-- depende de cruzar DB con DBAuth por cada fila (mismo problema ya descrito
-- en el comentario de handleListarGuarderiasPlataforma).
--
-- guarderia_id/usuario_id NO llevan FK a padres/usuarios -- mismo criterio
-- que padre_id/personal_id en mensajes_chat: "padres"/"usuarios" viven
-- conceptualmente aparte, y de un prospecto no hay id que guardar.
CREATE TABLE IF NOT EXISTS conversaciones_soporte (
    id SERIAL PRIMARY KEY,
    tipo VARCHAR(10) NOT NULL CHECK (tipo IN ('papa', 'staff', 'prospecto')),
    guarderia_id INTEGER REFERENCES guarderias(id) ON DELETE CASCADE,
    usuario_id INTEGER,
    nombre VARCHAR(150) NOT NULL,
    email VARCHAR(150),
    -- Identifica la conversación de un prospecto sin que tenga que crear
    -- una cuenta: el frontend lo guarda en localStorage y lo manda de
    -- vuelta en cada request siguiente. NULL para papa/staff (esos ya se
    -- identifican con su JWT normal).
    token VARCHAR(64) UNIQUE,
    cerrada BOOLEAN NOT NULL DEFAULT false,
    creado_en TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    actualizado_en TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Una sola conversación continua por (guardería, usuario) autenticado --
-- igual que un inbox de soporte de toda la vida, no un hilo nuevo cada vez
-- que alguien vuelve a escribir. Los prospectos quedan fuera de este
-- índice (se identifican por token, no por usuario_id, y de hecho lo
-- tienen NULL).
CREATE UNIQUE INDEX IF NOT EXISTS idx_conv_soporte_usuario
    ON conversaciones_soporte (tipo, guarderia_id, usuario_id)
    WHERE tipo != 'prospecto';

CREATE INDEX IF NOT EXISTS idx_conv_soporte_actividad ON conversaciones_soporte (actualizado_en DESC);

CREATE TABLE IF NOT EXISTS mensajes_soporte (
    id SERIAL PRIMARY KEY,
    conversacion_id INTEGER NOT NULL REFERENCES conversaciones_soporte(id) ON DELETE CASCADE,
    autor_rol VARCHAR(12) NOT NULL CHECK (autor_rol IN ('papa', 'staff', 'admin', 'prospecto', 'plataforma')),
    contenido TEXT NOT NULL,
    -- "leído por el otro lado", mismo criterio que mensajes_chat.leido: si
    -- autor_rol = 'plataforma', se refiere a si quien escribió el hilo ya
    -- lo vio; si no, a si el dueño de la plataforma ya lo vio.
    leido BOOLEAN NOT NULL DEFAULT false,
    creado_en TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_mensajes_soporte_conversacion ON mensajes_soporte (conversacion_id, creado_en);
