-- "Quiero que en las circulares también se puedan poner imágenes" --
-- misma key S3 privada que ya usan fotos de bitácora, documentos y
-- adjuntos de chat. Nullable: la mayoría de las circulares van a seguir
-- siendo solo texto.
ALTER TABLE circulares ADD COLUMN IF NOT EXISTS imagen_s3_key TEXT;
