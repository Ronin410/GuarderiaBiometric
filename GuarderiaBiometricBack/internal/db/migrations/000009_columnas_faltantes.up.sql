-- Columnas usadas por el código pero que faltaban en las migraciones versionadas
-- (ya existían en la base de datos de producción de forma manual).
ALTER TABLE hijos ADD COLUMN IF NOT EXISTS activo BOOLEAN DEFAULT true;
ALTER TABLE hijos ADD COLUMN IF NOT EXISTS url_token UUID DEFAULT gen_random_uuid();
ALTER TABLE padres ADD COLUMN IF NOT EXISTS celular VARCHAR(20);
ALTER TABLE padres ADD COLUMN IF NOT EXISTS recibe_whatsapp BOOLEAN DEFAULT false;
ALTER TABLE seguimiento_diario ADD COLUMN IF NOT EXISTS durmio BOOLEAN DEFAULT false;

-- El CHECK original de "usuarios.rol" no incluía 'papa', a pesar de que todo el
-- portal del padre depende de ese rol.
ALTER TABLE usuarios DROP CONSTRAINT IF EXISTS usuarios_rol_check;
ALTER TABLE usuarios ADD CONSTRAINT usuarios_rol_check CHECK (rol IN ('admin', 'staff', 'papa'));
