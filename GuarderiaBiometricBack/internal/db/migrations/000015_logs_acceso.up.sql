-- Bitácora de accesos a datos sensibles (LFPDPPP): intentos de login,
-- acciones ARCO (exportar/eliminar datos de un tutor), y accesos al link
-- público de bitácora (el único punto del sistema sin autenticación).
CREATE TABLE IF NOT EXISTS logs_acceso (
    id SERIAL PRIMARY KEY,
    evento VARCHAR(30) NOT NULL CHECK (evento IN (
        'login_exitoso', 'login_fallido', 'arco_exportar',
        'arco_eliminar', 'bitacora_publica'
    )),
    -- usuario_id no lleva FK: "usuarios" vive en la base de auth
    -- (DATABASE_URL_AUTH), conceptualmente separada de esta base operativa
    -- y potencialmente una base física distinta en producción.
    guarderia_id INTEGER REFERENCES guarderias(id) ON DELETE CASCADE,
    usuario_id INTEGER,
    detalle VARCHAR(200),
    ip VARCHAR(45),
    creado_en TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_logs_acceso_guarderia ON logs_acceso(guarderia_id);
CREATE INDEX IF NOT EXISTS idx_logs_acceso_fecha ON logs_acceso(creado_en DESC);
