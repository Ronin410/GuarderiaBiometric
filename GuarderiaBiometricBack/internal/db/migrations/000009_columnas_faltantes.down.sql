ALTER TABLE usuarios DROP CONSTRAINT IF EXISTS usuarios_rol_check;
ALTER TABLE usuarios ADD CONSTRAINT usuarios_rol_check CHECK (rol IN ('admin', 'staff'));

ALTER TABLE seguimiento_diario DROP COLUMN IF EXISTS durmio;
ALTER TABLE padres DROP COLUMN IF EXISTS recibe_whatsapp;
ALTER TABLE padres DROP COLUMN IF EXISTS celular;
ALTER TABLE hijos DROP COLUMN IF EXISTS url_token;
ALTER TABLE hijos DROP COLUMN IF EXISTS activo;
