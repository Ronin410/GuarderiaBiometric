ALTER TABLE logs_acceso DROP CONSTRAINT IF EXISTS logs_acceso_evento_check;
ALTER TABLE logs_acceso ADD CONSTRAINT logs_acceso_evento_check CHECK (evento IN (
    'login_exitoso', 'login_fallido', 'arco_exportar',
    'arco_eliminar', 'bitacora_publica',
    'personal_creado', 'personal_actualizado',
    'personal_pin_cambiado', 'personal_password_reset',
    'documento_subido', 'documento_eliminado'
));
