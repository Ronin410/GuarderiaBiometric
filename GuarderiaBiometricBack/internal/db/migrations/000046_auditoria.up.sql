-- Trazabilidad para tres puntos que hoy no dejan rastro de quién hizo qué --
-- lo que un inspector (o un abogado, si algo sale mal de verdad) pediría
-- ver primero. Los tres son aditivos: no cambian ni un dato existente, solo
-- agregan de dónde puede salir la evidencia de aquí en adelante.

-- 1) Atribución de staff en asistencia. Hasta ahora una fila de asistencia
-- solo prueba QUÉ tutor recogió al niño (vía padre_id, identificado por
-- Rekognition) pero no QUÉ CUENTA de staff operaba el kiosco -- y para un
-- movimiento forzado a mano (ver handleForzarEstatus en asistencia.go) ni
-- siquiera eso: el texto que quedaba grabado era el string fijo "Actualizado
-- por Admin", igual sin importar cuál admin. Nullable porque las filas ya
-- existentes no tienen este dato y no hay forma de reconstruirlo; a partir
-- de esta migración todo INSERT nuevo sí lo trae.
ALTER TABLE asistencia ADD COLUMN IF NOT EXISTS usuario_id INTEGER REFERENCES usuarios(id) ON DELETE SET NULL;

-- 2) Historial de bitácora (seguimiento_diario). El guardado de hoy hace un
-- UPSERT (ON CONFLICT ... DO UPDATE) que pisa el valor anterior sin dejar
-- rastro -- si alguien corrige "no comió" por "comió bien" un día después,
-- no queda ni quién lo cambió, ni cuándo, ni cuál era el valor de antes.
-- actualizado_por/actualizado_en en la tabla misma responden "¿quién tocó
-- esto por última vez?" sin tener que ir a buscarlo en otro lado;
-- seguimiento_diario_historial (solo se le hace INSERT, nunca UPDATE ni
-- DELETE, desde ningún código de este proyecto) guarda cada versión
-- completa que existió, en el orden en que se guardó.
ALTER TABLE seguimiento_diario ADD COLUMN IF NOT EXISTS actualizado_por INTEGER REFERENCES usuarios(id) ON DELETE SET NULL;
ALTER TABLE seguimiento_diario ADD COLUMN IF NOT EXISTS actualizado_en TIMESTAMP WITH TIME ZONE;

CREATE TABLE IF NOT EXISTS seguimiento_diario_historial (
    id SERIAL PRIMARY KEY,
    seguimiento_id INTEGER NOT NULL REFERENCES seguimiento_diario(id) ON DELETE CASCADE,
    hijo_id INTEGER NOT NULL,
    guarderia_id INTEGER NOT NULL,
    fecha DATE NOT NULL,
    desayuno TEXT,
    comida TEXT,
    merienda TEXT,
    esfinter TEXT,
    observaciones TEXT,
    durmio BOOLEAN,
    usuario_id INTEGER REFERENCES usuarios(id) ON DELETE SET NULL,
    guardado_en TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_seguimiento_historial_seguimiento ON seguimiento_diario_historial (seguimiento_id, guardado_en);

-- 3) Pagos: borrado suave. Un DELETE de verdad hace que un registro
-- financiero desaparezca sin dejar ninguna huella de que existió ni quién
-- lo quitó -- justo lo contrario de lo que se le puede pedir a una
-- guardería que demuestre. De aquí en adelante "eliminar un pago" (sigue
-- existiendo para corregir una captura equivocada, ver pagos.go) marca
-- estas dos columnas en vez de correr un DELETE; toda consulta que arma
-- saldos, historiales o recibos filtra eliminado_en IS NULL, así que para
-- cualquier pantalla de la app un pago "eliminado" se comporta exactamente
-- como si nunca hubiera existido -- pero la fila sigue en la base por si
-- algún día hay que revisar qué pasó.
ALTER TABLE pagos ADD COLUMN IF NOT EXISTS eliminado_en TIMESTAMP WITH TIME ZONE;
ALTER TABLE pagos ADD COLUMN IF NOT EXISTS eliminado_por INTEGER REFERENCES usuarios(id) ON DELETE SET NULL;
