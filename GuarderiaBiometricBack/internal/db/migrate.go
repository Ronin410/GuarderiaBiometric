package db

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// RunMigrations aplica el esquema versionado en la conexión dada, usando
// golang-migrate en vez del slice de SQL suelto que usaba el proyecto antes.
// El binario sigue corriendo las migraciones solo al arrancar (mismo
// comportamiento de despliegue en Render que ya existía) pero ahora con una
// tabla schema_migrations que registra qué versión quedó aplicada, en vez de
// sentencias idempotentes sin historial.
func RunMigrations(conexion *sql.DB) error {
	fmt.Println("Ejecutando migraciones...")

	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("no se pudo leer las migraciones embebidas: %w", err)
	}

	driver, err := postgres.WithInstance(conexion, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("no se pudo preparar el driver de migraciones: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return fmt.Errorf("no se pudo inicializar golang-migrate: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("error aplicando migraciones: %w", err)
	}

	fmt.Println("Migraciones finalizadas exitosamente.")
	return nil
}
