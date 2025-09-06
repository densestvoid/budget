package data

import (
	"database/sql"
	"embed"
	"fmt"
	"log"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrations embed.FS

// InitMigrations initializes the migrations
func InitMigrations() {
	// Debug: List all embedded files
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		log.Printf("Error reading migrations directory: %v", err)
	} else {
		log.Printf("Found %d migration files:", len(entries))
		for _, entry := range entries {
			log.Printf("  - %s", entry.Name())
		}
	}

	goose.SetBaseFS(migrations)
}

// Up runs all pending migrations
func Up(store *Storage) error {
	if err := goose.Up(store.db, "migrations"); err != nil {
		return fmt.Errorf("failed to run migrations up: %w", err)
	}
	return nil
}

// Down rolls back the last migration
func Down(store *Storage) error {
	if err := goose.Down(store.db, "migrations"); err != nil {
		return fmt.Errorf("failed to run migration down: %w", err)
	}
	return nil
}

// Status shows the current migration status
func Status(store *Storage) error {
	if err := goose.Status(store.db, "migrations"); err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}
	return nil
}

// GetDBConnection creates a database connection for migrations
func GetDBConnection(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
