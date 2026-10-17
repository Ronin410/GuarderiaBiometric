-- Soporte para "Gestión de personal": antes "usuarios" solo tenía un
-- username sin nombre para mostrar, y ninguna forma de revocar el acceso de
-- una cuenta sin borrarla (lo que rompería logs_acceso, pagos.registrado_por,
-- etc. si algún día llevan FK a usuarios).
ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS nombre VARCHAR(100);
ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS activo BOOLEAN NOT NULL DEFAULT true;
