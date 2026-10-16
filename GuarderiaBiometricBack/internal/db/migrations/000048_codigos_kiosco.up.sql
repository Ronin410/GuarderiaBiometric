-- Código QR del kiosco: lo genera la tablet (/admin/codigo-kiosco), no un
-- papá -- es ANÓNIMO a propósito, no identifica a ningún niño ni tutor en
-- particular. Solo prueba "alguien lo escaneó parado frente a esta tablet,
-- en este momento". La identidad real sale de la sesión del papá que lo
-- escanea desde SU PROPIA app (ya logueado) -- ver
-- handleCanjearCodigoKiosco en codigos_kiosco.go.
--
-- De un solo uso (usado_en/usado_por se llenan la primera vez que alguien
-- lo canjea, y nunca más) y de vida corta (expira_en, 1 minuto desde que
-- se generó) -- entre las dos cosas, ni se puede compartir entre dos papás
-- ni sirve fotografiarlo para usarlo después, lejos de la tablet.
CREATE TABLE IF NOT EXISTS codigos_kiosco (
    id SERIAL PRIMARY KEY,
    guarderia_id INTEGER NOT NULL REFERENCES guarderias(id) ON DELETE CASCADE,
    token VARCHAR(64) UNIQUE NOT NULL,
    expira_en TIMESTAMP WITH TIME ZONE NOT NULL,
    usado_en TIMESTAMP WITH TIME ZONE,
    usado_por INTEGER REFERENCES padres(id) ON DELETE SET NULL,
    creado_en TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_codigos_kiosco_token ON codigos_kiosco (token);
CREATE INDEX IF NOT EXISTS idx_codigos_kiosco_guarderia ON codigos_kiosco (guarderia_id);

-- Mismo criterio que codigo_acceso_id (migración 000047): nullable y con
-- SET NULL, se llena solo cuando el movimiento lo confirmó el propio papá
-- desde su app vía este mecanismo, para poder distinguirlo en el historial
-- sin depender de leer el texto libre de observaciones.
ALTER TABLE asistencia ADD COLUMN IF NOT EXISTS codigo_kiosco_id INTEGER REFERENCES codigos_kiosco(id) ON DELETE SET NULL;
