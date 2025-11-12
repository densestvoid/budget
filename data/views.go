package data

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
)

//go:embed views/*.sql
var viewsFS embed.FS

// extractViewName extracts the view name from a CREATE OR REPLACE VIEW statement
func extractViewName(sql string) (string, error) {
	// Match CREATE OR REPLACE VIEW view_name AS
	re := regexp.MustCompile(`(?i)CREATE\s+OR\s+REPLACE\s+VIEW\s+([a-zA-Z_][a-zA-Z0-9_]*)`)
	matches := re.FindStringSubmatch(sql)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not extract view name from SQL")
	}
	return matches[1], nil
}

// ApplyViews applies all views from the views directory to the database
// Views are dropped first if they exist, then recreated, so this is safe to run multiple times
func ApplyViews(db *sql.DB) error {
	entries, err := viewsFS.ReadDir("views")
	if err != nil {
		return fmt.Errorf("failed to read views directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process .sql files
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		viewPath := filepath.Join("views", entry.Name())
		viewSQL, err := viewsFS.ReadFile(viewPath)
		if err != nil {
			return fmt.Errorf("failed to read view file %s: %w", viewPath, err)
		}

		// Extract view name and drop it first (PostgreSQL doesn't allow changing column names with CREATE OR REPLACE)
		viewName, err := extractViewName(string(viewSQL))
		if err != nil {
			return fmt.Errorf("failed to extract view name from %s: %w", entry.Name(), err)
		}

		// Drop the view if it exists
		dropSQL := fmt.Sprintf("DROP VIEW IF EXISTS %s CASCADE;", viewName)
		if _, err := db.Exec(dropSQL); err != nil {
			return fmt.Errorf("failed to drop view %s: %w", viewName, err)
		}

		// Execute the view definition
		if _, err := db.Exec(string(viewSQL)); err != nil {
			return fmt.Errorf("failed to apply view %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// ApplyView applies a specific view by name (without .sql extension)
func ApplyView(db *sql.DB, viewName string) error {
	viewPath := filepath.Join("views", viewName+".sql")
	viewSQL, err := viewsFS.ReadFile(viewPath)
	if err != nil {
		return fmt.Errorf("failed to read view file %s: %w", viewPath, err)
	}

	// Extract view name from SQL and drop it first
	sqlViewName, err := extractViewName(string(viewSQL))
	if err != nil {
		return fmt.Errorf("failed to extract view name from %s: %w", viewName, err)
	}

	// Drop the view if it exists
	dropSQL := fmt.Sprintf("DROP VIEW IF EXISTS %s CASCADE;", sqlViewName)
	if _, err := db.Exec(dropSQL); err != nil {
		return fmt.Errorf("failed to drop view %s: %w", sqlViewName, err)
	}

	// Execute the view definition
	if _, err := db.Exec(string(viewSQL)); err != nil {
		return fmt.Errorf("failed to apply view %s: %w", viewName, err)
	}

	return nil
}

// ListViews returns a list of all available view names
func ListViews() ([]string, error) {
	entries, err := viewsFS.ReadDir("views")
	if err != nil {
		return nil, fmt.Errorf("failed to read views directory: %w", err)
	}

	var views []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if strings.HasSuffix(entry.Name(), ".sql") {
			// Remove .sql extension
			viewName := strings.TrimSuffix(entry.Name(), ".sql")
			views = append(views, viewName)
		}
	}

	return views, nil
}

// WalkViews walks through all view files and calls the provided function for each
func WalkViews(fn func(name string, sql string) error) error {
	return fs.WalkDir(viewsFS, "views", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".sql") {
			return nil
		}

		viewSQL, err := viewsFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read view file %s: %w", path, err)
		}

		viewName := strings.TrimSuffix(filepath.Base(path), ".sql")
		return fn(viewName, string(viewSQL))
	})
}

