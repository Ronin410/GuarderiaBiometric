ALTER TABLE pagos DROP COLUMN IF EXISTS eliminado_por;
ALTER TABLE pagos DROP COLUMN IF EXISTS eliminado_en;

DROP TABLE IF EXISTS seguimiento_diario_historial;
ALTER TABLE seguimiento_diario DROP COLUMN IF EXISTS actualizado_en;
ALTER TABLE seguimiento_diario DROP COLUMN IF EXISTS actualizado_por;

ALTER TABLE asistencia DROP COLUMN IF EXISTS usuario_id;
