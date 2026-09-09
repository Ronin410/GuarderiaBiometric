-- "Quiero que los documentos que se subirán por niño sea modificable por
-- guardería, ellos decidirán qué documentos son los que les pedirán al
-- papá" -- hasta ahora el catálogo de tipos de documento era un CHECK fijo
-- en documentos_nino.tipo (6 valores iguales para todas las guarderías).
-- Esta migración lo vuelve configurable por guardería.
CREATE TABLE IF NOT EXISTS tipos_documento (
    id SERIAL PRIMARY KEY,
    guarderia_id INTEGER REFERENCES guarderias(id) ON DELETE CASCADE,
    clave VARCHAR(50) NOT NULL,
    nombre VARCHAR(100) NOT NULL,
    orden INTEGER NOT NULL DEFAULT 0,
    creado_en TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (guarderia_id, clave)
);
CREATE INDEX IF NOT EXISTS idx_tipos_documento_guarderia ON tipos_documento (guarderia_id);

-- Cada guardería que ya existe arranca con el mismo catálogo que tenía fijo
-- antes (mismo orden), para no perder ni renombrar nada de lo que ya se
-- subió -- documentos_nino.tipo sigue guardando estas mismas claves.
INSERT INTO tipos_documento (guarderia_id, clave, nombre, orden)
SELECT g.id, t.clave, t.nombre, t.orden
FROM guarderias g
CROSS JOIN (VALUES
    ('acta_nacimiento', 'Acta de Nacimiento', 0),
    ('curp', 'CURP', 1),
    ('comprobante_domicilio', 'Comprobante de Domicilio', 2),
    ('cartilla_vacunacion', 'Cartilla de Vacunación', 3),
    ('identificacion_tutor', 'Identificación del Tutor', 4),
    ('otro', 'Otro', 5)
) AS t(clave, nombre, orden)
ON CONFLICT (guarderia_id, clave) DO NOTHING;

-- El catálogo fijo de antes se reemplaza por una FK compuesta contra el
-- catálogo propio de cada guardería: un documento solo puede tener un tipo
-- que esa guardería en verdad tiene configurado.
ALTER TABLE documentos_nino DROP CONSTRAINT IF EXISTS documentos_nino_tipo_check;
ALTER TABLE documentos_nino
    ADD CONSTRAINT documentos_nino_tipo_fk
    FOREIGN KEY (guarderia_id, tipo) REFERENCES tipos_documento (guarderia_id, clave);
