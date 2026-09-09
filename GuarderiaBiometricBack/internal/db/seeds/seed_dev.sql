-- Datos de prueba para desarrollo local (Podman/Docker). NO se aplica
-- automáticamente con las migraciones (internal/db/migrate.go) — es un
-- script aparte que se corre a mano una vez que el backend ya levantó y
-- aplicó las migraciones (ver podman/run.sh).
--
-- Pensado para probar todo lo de administración (paneles de Familia,
-- Bitácora, Perfiles, Pagos, Reportes, Estadísticas, Configuración, ARCO)
-- SIN depender de credenciales reales de AWS: los "padres" ya vienen con un
-- face_id de mentira (nunca se llamó a Rekognition para generarlo), así que
-- el login/reconocimiento facial del kiosco (POST /registrar, /identificar)
-- NO va a funcionar con estos datos — para eso hace falta AWS real (ver el
-- README de podman/, sección "Probar con Rekognition real").
--
-- Usuarios de prueba (contraseña para los tres: Demo1234!):
--   admin_demo  (rol admin, PIN 1234)
--   staff_demo  (rol staff, PIN 1234)
--   papa_demo   (rol papa — portal del padre, hijo vinculado: Emiliano Demo)
--
-- Es seguro volver a correrlo: empieza truncando todo lo que cuelga de
-- guarderias/usuarios y vuelve a sembrar desde cero.

TRUNCATE TABLE guarderias, usuarios RESTART IDENTITY CASCADE;

-- 1. Guardería -------------------------------------------------------------
INSERT INTO guarderias (id, nombre, slug, direccion, plan_suscripcion, aviso_privacidad_texto, aviso_privacidad_version) VALUES
(1, 'Guardería Demo', 'demo', 'Av. Insurgentes 123, Culiacán, Sinaloa',
 'basico',
 'TEXTO DE EJEMPLO — reemplaza esto con el Aviso de Privacidad real que te ' ||
 'entregue tu asesor legal antes de usar el sistema con datos reales. Debe ' ||
 'cubrir el tratamiento de datos biométricos (reconocimiento facial de ' ||
 'tutores) y datos de menores de edad conforme a la LFPDPPP.',
 'v1');

-- 2. Usuarios (login) -------------------------------------------------------
-- Hash bcrypt de "Demo1234!" — igual para los tres, generado una sola vez
-- (bcrypt.GenerateFromPassword), no es un secreto real de producción.
INSERT INTO usuarios (id, guarderia_id, username, password_hash, pin_admin, rol) VALUES
(1, 1, 'admin_demo', '$2a$10$avNStHUd9bN7ebOkCYVx1OpcPG1n9HC9bvrBNUMNEZ1cRXsgdcBaq', '1234', 'admin'),
(2, 1, 'staff_demo', '$2a$10$avNStHUd9bN7ebOkCYVx1OpcPG1n9HC9bvrBNUMNEZ1cRXsgdcBaq', '1234', 'staff'),
(3, 1, 'papa_demo',  '$2a$10$avNStHUd9bN7ebOkCYVx1OpcPG1n9HC9bvrBNUMNEZ1cRXsgdcBaq', '0000', 'papa');

-- 3. Padres (tutores) --------------------------------------------------------
-- face_id es "de mentira": nunca pasó por Rekognition, solo existe para que
-- la fila cumpla NOT NULL/UNIQUE. El id=3 coincide a propósito con
-- usuarios.id=3 (papa_demo): así es como el backend resuelve el comodín
-- "0" de /padre/0/hijos para una cuenta rol "papa" (ver handleHijosDePadre
-- en internal/server/hijos.go) — el user_id del token SE USA como padre_id.
INSERT INTO padres (id, nombre, face_id, guarderia_id, celular, recibe_whatsapp) VALUES
(1, 'Carlos Sánchez',        'seed-fake-face-001', 1, '6671234567', true),
(2, 'María Fernanda López',  'seed-fake-face-002', 1, '6679876543', false),
(3, 'Papá Demo',             'seed-fake-face-003', 1, '6675551234', true);

-- 4. Hijos --------------------------------------------------------------------
INSERT INTO hijos (id, nombre_niño, guarderia_id, activo, fecha_nacimiento, direccion, contacto_emergencia_nombre, contacto_emergencia_telefono, colegiatura_mensual) VALUES
(1, 'Valentina Cruz',   1, true, '2021-03-14', 'Calle Framboyanes 45', 'Ana Cruz',              '6671112233', 1800.00),
(2, 'Mateo Sánchez',    1, true, '2020-07-22', 'Calle Framboyanes 45', 'Carlos Sánchez',        '6671234567', 1800.00),
(3, 'Regina López',     1, true, '2022-01-05', 'Blvd. Elbert 900',     'María Fernanda López',  '6679876543', 2000.00),
(4, 'Emiliano Demo',    1, true, '2021-11-30', 'Calle Demo 1',         'Papá Demo',             '6675551234', 1800.00);

-- 5. Vínculos tutor-hijo ------------------------------------------------------
INSERT INTO tutor_hijos (padre_id, hijo_id, guarderia_id) VALUES
(1, 1, 1),  -- Carlos -> Valentina
(1, 2, 1),  -- Carlos -> Mateo
(2, 3, 1),  -- María Fernanda -> Regina
(3, 4, 1);  -- Papá Demo -> Emiliano

-- 6. Asistencia (para Bitácora, Reportes, Estadísticas) ------------------------
INSERT INTO asistencia (padre_id, hijo_id, guarderia_id, fecha_hora, aseado, reporte_golpe, observaciones, tipo_movimiento) VALUES
(1, 1, 1, CURRENT_DATE + TIME '08:15', true,  false, '',                          'ENTRADA'),
(1, 2, 1, CURRENT_DATE + TIME '08:20', true,  false, '',                          'ENTRADA'),
(2, 3, 1, CURRENT_DATE + TIME '08:40', true,  false, 'Llegó con gripa leve',      'ENTRADA'),
(3, 4, 1, CURRENT_DATE + TIME '09:00', true,  false, '',                          'ENTRADA'),
(1, 1, 1, CURRENT_DATE - INTERVAL '1 day' + TIME '08:10', true, false, '',        'ENTRADA'),
(1, 1, 1, CURRENT_DATE - INTERVAL '1 day' + TIME '13:30', true, false, '',        'SALIDA');

-- 7. Bitácora diaria (para VistaBitacora, ReporteDiario, portal del padre) -----
INSERT INTO seguimiento_diario (hijo_id, guarderia_id, fecha, desayuno, comida, merienda, esfinter, observaciones, durmio) VALUES
(1, 1, CURRENT_DATE, 'todo',     'poco',      'todo',      'Normal',                                 'Buen día, jugó mucho en el patio', true),
(2, 1, CURRENT_DATE, 'poco',     'todo',      'todo',      'Normal',                                 '',                                    true),
(3, 1, CURRENT_DATE, 'no_comio', 'poco',      'pendiente', 'Con gripa, se le monitoreó temperatura', 'Se le llamó a mamá para avisar',      false),
(4, 1, CURRENT_DATE, 'todo',     'todo',      'todo',      'Normal',                                 '',                                    true);
-- Nota: no se siembran fotos (fotos_seguimiento) porque requieren S3 real
-- (firmarURLFoto necesita credenciales de AWS válidas para generar la URL).

-- 8. Pagos (para PanelPagos / estado de colegiatura) ---------------------------
INSERT INTO pagos (hijo_id, guarderia_id, monto, concepto, periodo, fecha_pago, metodo_pago, observaciones) VALUES
(1, 1, 1800.00, 'Colegiatura', TO_CHAR(CURRENT_DATE, 'YYYY-MM'), CURRENT_DATE, 'transferencia', 'Pago completo del mes — hijo 1, debería verse "pagado"'),
(3, 1, 1000.00, 'Colegiatura', TO_CHAR(CURRENT_DATE, 'YYYY-MM'), CURRENT_DATE, 'efectivo',       'Abono parcial — hijo 3, debería verse "parcial"'),
(4, 1, 1800.00, 'Colegiatura', TO_CHAR(CURRENT_DATE - INTERVAL '1 month', 'YYYY-MM'), CURRENT_DATE - INTERVAL '1 month', 'efectivo', 'Pago del mes pasado únicamente — hijo 4, el mes actual debería verse pendiente/vencido');
-- Hijo 2 (Mateo) se deja sin ningún pago a propósito: debería verse "pendiente".

-- 9. Consentimientos (para el contador en el panel Configuración) -------------
INSERT INTO consentimientos (padre_id, padre_nombre_historico, guarderia_id, version_aviso, aceptado_en, ip) VALUES
(1, 'Carlos Sánchez',       1, 'v1', CURRENT_TIMESTAMP - INTERVAL '3 days', '127.0.0.1'),
(2, 'María Fernanda López', 1, 'v1', CURRENT_TIMESTAMP - INTERVAL '2 days', '127.0.0.1'),
(3, 'Papá Demo',            1, 'v1', CURRENT_TIMESTAMP - INTERVAL '1 day',  '127.0.0.1');

-- Reacomoda las secuencias para que la próxima fila que inserte la propia
-- app (ej. un padre nuevo desde el kiosco) no choque con los IDs sembrados
-- a mano arriba.
SELECT setval('guarderias_id_seq', (SELECT MAX(id) FROM guarderias));
SELECT setval('usuarios_id_seq', (SELECT MAX(id) FROM usuarios));
SELECT setval('padres_id_seq', (SELECT MAX(id) FROM padres));
SELECT setval('hijos_id_seq', (SELECT MAX(id) FROM hijos));
