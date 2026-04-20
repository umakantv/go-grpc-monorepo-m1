package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

type Config struct {
	Driver   string
	Host     string
	Port     int
	Name     string
	User     string
	Password string
	SSLMode  string
	Path     string
}

func New(cfg Config) (*sql.DB, error) {
	driverName, dsn := buildDSN(cfg)

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(10 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func buildDSN(cfg Config) (string, string) {
	if cfg.Driver == "sqlite" || cfg.Driver == "sqlite3" {
		return buildSQLiteDSN(cfg)
	}
	return "postgres", buildPostgresDSN(cfg)
}

func buildPostgresDSN(cfg Config) string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode,
	)
}

func buildSQLiteDSN(cfg Config) (string, string) {
	path := cfg.Path
	if path == "" || path == ":memory:" {
		return "sqlite", ":memory:?cache=shared"
	}
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}
	return "sqlite", fmt.Sprintf("file:%s?cache=shared&_fk=1", path)
}

// HealthCheck returns a function that checks database health
func HealthCheck(db *sql.DB) func(context.Context) error {
	return func(ctx context.Context) error {
		return db.PingContext(ctx)
	}
}
