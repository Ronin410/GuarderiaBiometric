-- "Horarios/turnos y horas trabajadas del personal": dos tablas separadas
-- porque son dos cosas distintas -- horarios_personal es el turno FIJO
-- semanal (el plan: qué días y qué horas se espera que trabaje alguien),
-- registro_horas es el registro REAL día a día (para control de nómina,
-- puede diferir del turno por faltas, horas extra, etc.).
--
-- usuario_id NO lleva FK a usuarios(id) -- mismo criterio ya usado en
-- push_subscripciones y logs_acceso: "usuarios" vive conceptualmente en la
-- base de auth (DATABASE_URL_AUTH), separada de esta base operativa.
CREATE TABLE IF NOT EXISTS horarios_personal (
    id SERIAL PRIMARY KEY,
    usuario_id INTEGER NOT NULL,
    guarderia_id INTEGER REFERENCES guarderias(id) ON DELETE CASCADE,
    dia_semana SMALLINT NOT NULL CHECK (dia_semana BETWEEN 0 AND 6), -- 0=lunes ... 6=domingo
    hora_entrada TIME,
    hora_salida TIME,
    UNIQUE (usuario_id, dia_semana)
);
CREATE INDEX IF NOT EXISTS idx_horarios_personal_usuario ON horarios_personal (usuario_id);

CREATE TABLE IF NOT EXISTS registro_horas (
    id SERIAL PRIMARY KEY,
    usuario_id INTEGER NOT NULL,
    guarderia_id INTEGER REFERENCES guarderias(id) ON DELETE CASCADE,
    fecha DATE NOT NULL,
    horas_trabajadas NUMERIC(4,2) NOT NULL,
    observaciones TEXT,
    UNIQUE (usuario_id, fecha)
);
CREATE INDEX IF NOT EXISTS idx_registro_horas_usuario_fecha ON registro_horas (usuario_id, fecha);
