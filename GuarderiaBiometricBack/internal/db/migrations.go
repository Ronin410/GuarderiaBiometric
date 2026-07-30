package db

import (
	"database/sql"
	"fmt"
	"log"
)

// RunMigrations aplica el esquema en la conexión dada. Las sentencias son
// idempotentes (CREATE ... IF NOT EXISTS / ADD COLUMN IF NOT EXISTS), así que
// correrlas en cada arranque es seguro.
//
// TODO(Fase 3 de MEJORAS_PROPUESTAS.md): reemplazar este slice de SQL suelto
// por migraciones versionadas con golang-migrate, para tener control real de
// versión de esquema en vez de sentencias idempotentes sin historial.
func RunMigrations(conexion *sql.DB) {
	fmt.Println("Ejecutando migraciones...")

	// El orden es importante por las llaves foráneas (Foreign Keys)
	queries := []string{
		// 1. Tabla Guarderías
		`CREATE TABLE IF NOT EXISTS guarderias (
			id SERIAL PRIMARY KEY,
			nombre VARCHAR(100) NOT NULL,
			slug VARCHAR(50) UNIQUE NOT NULL,
			direccion TEXT,
			plan_suscripcion VARCHAR(20) DEFAULT 'basico',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,

		// 2. Tabla Usuarios
		`CREATE TABLE IF NOT EXISTS usuarios (
			id SERIAL PRIMARY KEY,
			guarderia_id INTEGER REFERENCES guarderias(id) ON DELETE CASCADE,
			username VARCHAR(50) UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			pin_admin VARCHAR(4) NOT NULL,
			rol VARCHAR(20) DEFAULT 'staff' CHECK (rol IN ('admin', 'staff', 'papa')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,

		// 3. Tabla Hijos
		`CREATE TABLE IF NOT EXISTS hijos (
			id SERIAL PRIMARY KEY,
			nombre_niño VARCHAR(100) NOT NULL,
			guarderia_id INTEGER REFERENCES guarderias(id) ON DELETE CASCADE
		);`,

		// 4. Tabla Padres
		`CREATE TABLE IF NOT EXISTS padres (
			id SERIAL PRIMARY KEY,
			nombre VARCHAR(100) NOT NULL,
			face_id VARCHAR(255) UNIQUE NOT NULL,
			guarderia_id INTEGER REFERENCES guarderias(id) ON DELETE CASCADE,
			creado_en TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,

		// 5. Tabla Intermedia Tutor_Hijos
		`CREATE TABLE IF NOT EXISTS tutor_hijos (
			padre_id INTEGER REFERENCES padres(id) ON DELETE CASCADE,
			hijo_id INTEGER REFERENCES hijos(id) ON DELETE CASCADE,
			guarderia_id INTEGER REFERENCES guarderias(id) ON DELETE CASCADE,
			PRIMARY KEY (padre_id, hijo_id)
		);`,

		// 6. Tabla Asistencia
		`CREATE TABLE IF NOT EXISTS asistencia (
			id SERIAL PRIMARY KEY,
			padre_id INTEGER REFERENCES padres(id) ON DELETE CASCADE,
			hijo_id INTEGER REFERENCES hijos(id) ON DELETE CASCADE,
			guarderia_id INTEGER REFERENCES guarderias(id) ON DELETE CASCADE,
			fecha_hora TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			aseado BOOLEAN DEFAULT true,
			reporte_golpe BOOLEAN DEFAULT false,
			observaciones TEXT,
			tipo_movimiento VARCHAR(20) CHECK (tipo_movimiento IN ('ENTRADA', 'SALIDA', 'REGISTRO'))
		);`,

		// Nueva tabla para el seguimiento diario
		`CREATE TABLE IF NOT EXISTS seguimiento_diario (
			id SERIAL PRIMARY KEY,
			hijo_id INTEGER REFERENCES hijos(id) ON DELETE CASCADE,
			guarderia_id INTEGER REFERENCES guarderias(id) ON DELETE CASCADE,
			fecha DATE DEFAULT CURRENT_DATE,
			desayuno VARCHAR(20) DEFAULT 'pendiente', -- 'no_comio', 'poco', 'todo'
			comida VARCHAR(20) DEFAULT 'pendiente',
			merienda VARCHAR(20) DEFAULT 'pendiente',
			esfinter VARCHAR(50),
			foto_url TEXT,
			observaciones TEXT,
			UNIQUE(hijo_id, fecha) -- Evita duplicados para el mismo niño el mismo día
		);`,

		`CREATE TABLE IF NOT EXISTS fotos_seguimiento (
			id SERIAL PRIMARY KEY,
			seguimiento_id INT REFERENCES seguimiento_diario(id),
			url TEXT NOT NULL,
			creado_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,

		// 7. Índices adicionales
		`CREATE INDEX IF NOT EXISTS idx_asistencia_fecha ON asistencia (fecha_hora DESC);`,

		// 8. Columnas usadas por el código pero que faltaban en las migraciones versionadas
		// (ya existían en la base de datos de producción de forma manual).
		`ALTER TABLE hijos ADD COLUMN IF NOT EXISTS activo BOOLEAN DEFAULT true;`,
		`ALTER TABLE hijos ADD COLUMN IF NOT EXISTS url_token UUID DEFAULT gen_random_uuid();`,
		`ALTER TABLE padres ADD COLUMN IF NOT EXISTS celular VARCHAR(20);`,
		`ALTER TABLE padres ADD COLUMN IF NOT EXISTS recibe_whatsapp BOOLEAN DEFAULT false;`,
		`ALTER TABLE seguimiento_diario ADD COLUMN IF NOT EXISTS durmio BOOLEAN DEFAULT false;`,
		// El CHECK original de "usuarios.rol" no incluía 'papa', a pesar de que todo
		// el portal del padre depende de ese rol. En una base ya existente (creada
		// antes de este fix) el CREATE TABLE de arriba es un no-op, así que se
		// corrige el constraint explícitamente aquí.
		`ALTER TABLE usuarios DROP CONSTRAINT IF EXISTS usuarios_rol_check;`,
		`ALTER TABLE usuarios ADD CONSTRAINT usuarios_rol_check CHECK (rol IN ('admin', 'staff', 'papa'));`,

		// 9. Perfil extendido del niño (módulo de Administración)
		`ALTER TABLE hijos ADD COLUMN IF NOT EXISTS fecha_nacimiento DATE;`,
		`ALTER TABLE hijos ADD COLUMN IF NOT EXISTS direccion TEXT;`,
		`ALTER TABLE hijos ADD COLUMN IF NOT EXISTS contacto_emergencia_nombre VARCHAR(150);`,
		`ALTER TABLE hijos ADD COLUMN IF NOT EXISTS contacto_emergencia_telefono VARCHAR(20);`,
		`ALTER TABLE hijos ADD COLUMN IF NOT EXISTS colegiatura_mensual NUMERIC(10,2) DEFAULT 0;`,

		// 10. Tabla de Pagos (módulo de Administración)
		// Nota: sin UNIQUE(hijo_id, periodo, concepto) a propósito — un mismo niño puede
		// tener varios abonos ("pagos parciales") con el mismo concepto en el mismo periodo.
		`CREATE TABLE IF NOT EXISTS pagos (
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
		);`,
		`CREATE INDEX IF NOT EXISTS idx_pagos_hijo_periodo ON pagos (hijo_id, periodo);`,

		// 11. Suscripciones de notificaciones push (Web Push, sin servicios de terceros).
		// padre_id guarda el mismo valor que ya usa /padre/:id/hijos (user_id del token
		// para cuentas rol "papa"), por eso no lleva FK a padres(id) — ver plan.
		`CREATE TABLE IF NOT EXISTS push_subscripciones (
			id SERIAL PRIMARY KEY,
			padre_id INTEGER NOT NULL,
			guarderia_id INTEGER REFERENCES guarderias(id) ON DELETE CASCADE,
			endpoint TEXT NOT NULL UNIQUE,
			p256dh TEXT NOT NULL,
			auth TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_push_padre ON push_subscripciones(padre_id);`,

		// 12. Índices compuestos para las consultas multi-tenant más frecuentes
		// (casi todo filtra por guarderia_id primero). tutor_hijos ya tiene PK
		// (padre_id, hijo_id), que no sirve para buscar solo por hijo_id (ej.
		// notificarEvento en push.go), por eso se agrega ese índice aparte.
		`CREATE INDEX IF NOT EXISTS idx_hijos_guarderia_activo ON hijos (guarderia_id, activo);`,
		`CREATE INDEX IF NOT EXISTS idx_padres_guarderia ON padres (guarderia_id);`,
		`CREATE INDEX IF NOT EXISTS idx_tutor_hijos_hijo_id ON tutor_hijos (hijo_id);`,
		`CREATE INDEX IF NOT EXISTS idx_pagos_guarderia ON pagos (guarderia_id);`,
	}

	for _, q := range queries {
		if _, err := conexion.Exec(q); err != nil {
			log.Printf("Error ejecutando migración: %v", err)
		}
	}

	fmt.Println("Migraciones finalizadas exitosamente.")
}
