-- Códigos de acceso (QR) para que un papá autorice a alguien que no está
-- enrolado con reconocimiento facial -- un familiar que va por el niño
-- ocasionalmente, o un papá al que no le gusta poner su rostro -- a pasar
-- por el kiosco de Recepción. Se genera desde el portal del papá
-- (codigos_acceso.go) y se valida en el kiosco igual que hoy se valida un
-- rostro: identifica primero (handleIdentificarPorCodigo), confirma
-- después (handleConfirmarAsistencia).
--
-- A diferencia de la biometría, un código NO prueba quién lo trae -- solo
-- que quien lo trae lo tiene. persona_autorizada es el texto libre que
-- escribe el papá ("Tía Lupita") precisamente para que quede una pista de
-- a quién se esperaba, no solo que "alguien con el código" pasó -- ver el
-- comentario largo en asistencia.go sobre cómo se usa al confirmar.
CREATE TABLE IF NOT EXISTS codigos_acceso (
    id SERIAL PRIMARY KEY,
    padre_id INTEGER NOT NULL REFERENCES padres(id) ON DELETE CASCADE,
    guarderia_id INTEGER NOT NULL REFERENCES guarderias(id) ON DELETE CASCADE,
    -- token: lo que lleva el QR (largo, para que no sea adivinable).
    -- codigo_corto: respaldo para que recepción lo teclee a mano si la
    -- cámara no logra leer el QR (mala luz, brillo de la pantalla del
    -- celular) -- alfabeto sin 0/O/1/I/L para no confundirlos al dictarlos.
    token VARCHAR(64) UNIQUE NOT NULL,
    codigo_corto VARCHAR(8) UNIQUE NOT NULL,
    persona_autorizada VARCHAR(100) NOT NULL,
    expira_en TIMESTAMP WITH TIME ZONE NOT NULL,
    revocado_en TIMESTAMP WITH TIME ZONE,
    creado_en TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_codigos_acceso_token ON codigos_acceso (token);
CREATE INDEX IF NOT EXISTS idx_codigos_acceso_codigo_corto ON codigos_acceso (codigo_corto);
CREATE INDEX IF NOT EXISTS idx_codigos_acceso_padre ON codigos_acceso (padre_id);

-- Un código puede autorizar a varios hijos del mismo papá (o a uno solo,
-- si tiene más de uno y solo quiere delegar la recogida de ese niño).
CREATE TABLE IF NOT EXISTS codigos_acceso_hijos (
    codigo_id INTEGER NOT NULL REFERENCES codigos_acceso(id) ON DELETE CASCADE,
    hijo_id INTEGER NOT NULL REFERENCES hijos(id) ON DELETE CASCADE,
    PRIMARY KEY (codigo_id, hijo_id)
);

-- Nullable y con SET NULL -- la enorme mayoría de los movimientos siguen
-- siendo por rostro y nunca tocan esta columna; se llena solo cuando el
-- movimiento vino de un código (ver handleConfirmarAsistencia), para poder
-- distinguir "cuántas recogidas fueron por QR" sin depender de leer el
-- texto libre de observaciones.
ALTER TABLE asistencia ADD COLUMN IF NOT EXISTS codigo_acceso_id INTEGER REFERENCES codigos_acceso(id) ON DELETE SET NULL;
