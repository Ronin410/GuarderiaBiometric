-- Índices compuestos para las consultas multi-tenant más frecuentes (casi todo
-- filtra por guarderia_id primero). tutor_hijos ya tiene PK (padre_id, hijo_id),
-- que no sirve para buscar solo por hijo_id (ej. notificarEvento en push.go), por
-- eso se agrega ese índice aparte.
CREATE INDEX IF NOT EXISTS idx_hijos_guarderia_activo ON hijos (guarderia_id, activo);
CREATE INDEX IF NOT EXISTS idx_padres_guarderia ON padres (guarderia_id);
CREATE INDEX IF NOT EXISTS idx_tutor_hijos_hijo_id ON tutor_hijos (hijo_id);
CREATE INDEX IF NOT EXISTS idx_pagos_guarderia ON pagos (guarderia_id);
