-- Solicitudes de alta de una guardería nueva (autoservicio con aprobación
-- manual): el formulario público SOLO inserta aquí -- crear la guardería y
-- su primera cuenta admin de verdad (tablas guarderias/usuarios) pasa a
-- ocurrir hasta que /plataforma/solicitudes/:id/aprobar la apruebe. Así una
-- solicitud sin aprobar nunca consume el username deseado ni deja un tenant
-- a medias dando vueltas.
CREATE TABLE IF NOT EXISTS solicitudes_guarderia (
    id SERIAL PRIMARY KEY,
    nombre_guarderia VARCHAR(100) NOT NULL,
    direccion TEXT,
    nombre_contacto VARCHAR(100) NOT NULL,
    email_contacto VARCHAR(150) NOT NULL,
    telefono_contacto VARCHAR(20),
    username_deseado VARCHAR(50) NOT NULL,
    password_hash TEXT NOT NULL,
    estado VARCHAR(20) NOT NULL DEFAULT 'pendiente' CHECK (estado IN ('pendiente', 'aprobada', 'rechazada')),
    nota_revision TEXT,
    guarderia_id INTEGER REFERENCES guarderias(id),
    creado_en TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    revisado_en TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_solicitudes_guarderia_estado ON solicitudes_guarderia (estado);
