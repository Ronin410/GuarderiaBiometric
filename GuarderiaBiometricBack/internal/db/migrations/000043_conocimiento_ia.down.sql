ALTER TABLE mensajes_soporte DROP COLUMN IF EXISTS generado_por_ia;
DROP INDEX IF EXISTS idx_fragmentos_conocimiento_embedding;
DROP TABLE IF EXISTS fragmentos_conocimiento;
-- No se hace DROP EXTENSION vector: si algo más llegara a depender de ella
-- más adelante, tumbarla aquí sería sorpresivo -- y Postgres la rechaza
-- sola de todos modos si otra cosa la sigue usando.
