-- "Al momento de crear el perfil de un niño, se guardan algunos datos de
-- inscripción como acta de nacimiento, comprobante de domicilio, etc, quiero
-- que estos se puedan guardar y actualizar." UNIQUE(hijo_id, tipo): subir un
-- documento del mismo tipo reemplaza al anterior (vía ON CONFLICT en el
-- handler), no acumula versiones viejas.
CREATE TABLE IF NOT EXISTS documentos_nino (
    id SERIAL PRIMARY KEY,
    hijo_id INTEGER REFERENCES hijos(id) ON DELETE CASCADE,
    guarderia_id INTEGER REFERENCES guarderias(id) ON DELETE CASCADE,
    tipo VARCHAR(40) NOT NULL CHECK (tipo IN (
        'acta_nacimiento', 'curp', 'comprobante_domicilio',
        'cartilla_vacunacion', 'identificacion_tutor', 'otro'
    )),
    nombre_archivo VARCHAR(255) NOT NULL,
    s3_key TEXT NOT NULL,
    subido_en TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (hijo_id, tipo)
);
CREATE INDEX IF NOT EXISTS idx_documentos_nino_hijo ON documentos_nino (hijo_id);

-- Nuevos eventos auditables: subir/eliminar un documento de inscripción
-- toca datos personales sensibles (domicilio, CURP), igual criterio que el
-- resto de logs_acceso.
ALTER TABLE logs_acceso DROP CONSTRAINT IF EXISTS logs_acceso_evento_check;
ALTER TABLE logs_acceso ADD CONSTRAINT logs_acceso_evento_check CHECK (evento IN (
    'login_exitoso', 'login_fallido', 'arco_exportar',
    'arco_eliminar', 'bitacora_publica',
    'personal_creado', 'personal_actualizado',
    'personal_pin_cambiado', 'personal_password_reset',
    'documento_subido', 'documento_eliminado'
));
