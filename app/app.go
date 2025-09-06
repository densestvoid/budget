package app

import (
	"database/sql"
	"fmt"
	"log"
)

// App represents the main application with all its dependencies
type App struct {
	db *sql.DB
}

// NewApp creates a new application instance
func NewApp() *App {
	return &App{}
}

// SetDB sets the database connection for the application
func (a *App) SetDB(db *sql.DB) {
	a.db = db
}

// GetDB returns the database connection
func (a *App) GetDB() *sql.DB {
	return a.db
}

// ConnectDB establishes a database connection
func (a *App) ConnectDB(dsn string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	a.db = db
	log.Println("Connected to database successfully")
	return nil
}

// CloseDB closes the database connection
func (a *App) CloseDB() error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}
