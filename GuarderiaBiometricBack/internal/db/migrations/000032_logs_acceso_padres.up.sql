-- Nuevos eventos auditables sobre cuentas del portal de padres:
-- restablecer la contraseña de una cuenta ya existente
-- (handleRestablecerPasswordPadre), y mover a un padre a un id nuevo
-- cuando su id chocaba con la cuenta de otra persona
-- (handleResolverChoqueID) -- mismo criterio que el resto de logs_acceso:
-- ambas acciones tocan credenciales/datos sensibles y deben quedar
-- registradas, no fallar en silencio como pasaba antes de esta migración
-- (el CHECK las rechazaba por no estar en la lista).
ALTER TABLE logs_acceso DROP CONSTRAINT IF EXISTS logs_acceso_evento_check;
ALTER TABLE logs_acceso ADD CONSTRAINT logs_acceso_evento_check CHECK (evento IN (
    'login_exitoso', 'login_fallido', 'arco_exportar',
    'arco_eliminar', 'bitacora_publica',
    'personal_creado', 'personal_actualizado',
    'personal_pin_cambiado', 'personal_password_reset',
    'documento_subido', 'documento_eliminado',
    'padre_password_reset', 'padre_id_reasignado'
));
