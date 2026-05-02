package migrate

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Up runs all pending migrations against the given database.
// migrationsPath is relative to the working directory (e.g. "migrations").
// Returns nil when the database is already up-to-date.
func Up(db *sql.DB, driverName string, migrationsPath string) error {
	m, err := newMigrate(db, driverName, migrationsPath)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

// Down rolls back the last migration.
func Down(db *sql.DB, driverName string, migrationsPath string) error {
	m, err := newMigrate(db, driverName, migrationsPath)
	if err != nil {
		return err
	}

	if err := m.Steps(-1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate down: %w", err)
	}
	return nil
}

// Version returns the current migration version and dirty flag.
func Version(db *sql.DB, driverName string, migrationsPath string) (uint, bool, error) {
	m, err := newMigrate(db, driverName, migrationsPath)
	if err != nil {
		return 0, false, err
	}
	return m.Version()
}

func newMigrate(db *sql.DB, driverName string, migrationsPath string) (*migrate.Migrate, error) {
	driver, err := databaseDriver(db, driverName)
	if err != nil {
		return nil, err
	}

	sourceURL := fmt.Sprintf("file://%s", migrationsPath)

	m, err := migrate.NewWithDatabaseInstance(sourceURL, driverName, driver)
	if err != nil {
		return nil, fmt.Errorf("migrate: failed to initialise: %w", err)
	}
	return m, nil
}

func databaseDriver(db *sql.DB, driverName string) (database.Driver, error) {
	switch driverName {
	case "postgres", "":
		d, err := postgres.WithInstance(db, &postgres.Config{})
		if err != nil {
			return nil, fmt.Errorf("migrate: postgres driver: %w", err)
		}
		return d, nil
	case "sqlite", "sqlite3":
		d, err := sqlite.WithInstance(db, &sqlite.Config{})
		if err != nil {
			return nil, fmt.Errorf("migrate: sqlite driver: %w", err)
		}
		return d, nil
	default:
		return nil, fmt.Errorf("migrate: unsupported driver %q", driverName)
	}
}

// DatabaseDriver is re-exported so callers don't need to import the migrate package.
type DatabaseDriver = database.Driver