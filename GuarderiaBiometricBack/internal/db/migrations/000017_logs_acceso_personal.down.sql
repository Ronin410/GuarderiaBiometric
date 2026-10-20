ALTER TABLE logs_acceso DROP CONSTRAINT IF EXISTS logs_acceso_evento_check;
ALTER TABLE logs_acceso ADD CONSTRAINT logs_acceso_evento_check CHECK (evento IN (
    'login_exitoso', 'login_fallido', 'arco_exportar',
    'arco_eliminar', 'bitacora_publica'
));
