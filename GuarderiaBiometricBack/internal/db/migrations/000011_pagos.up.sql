-- Sin UNIQUE(hijo_id, periodo, concepto) a propósito — un mismo niño puede tener
-- varios abonos ("pagos parciales") con el mismo concepto en el mismo periodo.
CREATE TABLE IF NOT EXISTS pagos (
    id SERIAL PRIMARY KEY,
    hijo_id INTEGER REFERENCES hijos(id) ON DELETE CASCADE,
    guarderia_id INTEGER REFERENCES guarderias(id) ON DELETE CASCADE,
    monto NUMERIC(10,2) NOT NULL,
    concepto VARCHAR(100) NOT NULL DEFAULT 'Colegiatura',
    periodo VARCHAR(7) NOT NULL,
    fecha_pago DATE DEFAULT CURRENT_DATE,
    metodo_pago VARCHAR(20) DEFAULT 'efectivo',
    observaciones TEXT,
    registrado_por INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_pagos_hijo_periodo ON pagos (hijo_id, periodo);
