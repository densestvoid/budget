package data

import (
	"database/sql"
	"fmt"
)

// Storage represents the data access layer
type Storage struct {
	db *sql.DB
}

// NewStorage creates a new storage instance
func NewStorage(db *sql.DB) *Storage {
	return &Storage{
		db: db,
	}
}

// GetDB returns the database connection
func (s *Storage) GetDB() *sql.DB {
	return s.db
}

// Close closes the database connection
func (s *Storage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Ping tests the database connection
func (s *Storage) Ping() error {
	if s.db == nil {
		return fmt.Errorf("database connection not established")
	}
	return s.db.Ping()
}
