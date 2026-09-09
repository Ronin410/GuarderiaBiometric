-- Adjuntos en el chat privado padres<->guardería: una imagen o archivo por
-- mensaje, aparte del texto -- contenido puede quedar vacío si el mensaje
-- es solo el adjunto (la validación de "algo tiene que traer" vive en el
-- backend, no en una restricción de la tabla). adjunto_tipo distingue
-- 'imagen' (el frontend la muestra en línea, como WhatsApp) de 'archivo'
-- (enlace de descarga) sin tener que adivinar por la extensión del nombre.
ALTER TABLE mensajes_chat ADD COLUMN IF NOT EXISTS adjunto_s3_key TEXT;
ALTER TABLE mensajes_chat ADD COLUMN IF NOT EXISTS adjunto_nombre VARCHAR(255);
ALTER TABLE mensajes_chat ADD COLUMN IF NOT EXISTS adjunto_tipo VARCHAR(20) CHECK (adjunto_tipo IN ('imagen', 'archivo'));
