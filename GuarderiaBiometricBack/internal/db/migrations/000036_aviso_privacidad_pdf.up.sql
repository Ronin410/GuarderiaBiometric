-- "En el aviso de privacidad también podrán, en lugar de escribir el
-- aviso, poner un archivo PDF" -- alternativa al texto pegado a mano.
-- Mismo bucket privado y mismo criterio que documentos/circulares: se
-- guarda la key de S3, no una URL pública, y se firma al leer.
ALTER TABLE guarderias ADD COLUMN IF NOT EXISTS aviso_privacidad_pdf_s3_key TEXT;
