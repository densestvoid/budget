package data

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const budgetSchemaName = "budget"

// BootstrapSchema ensures the budget schema exists and the application user owns it.
// When the schema is missing, adminDSN must point at the cluster admin user.
func BootstrapSchema(appDSN, adminDSN string) error {
	appUser, err := postgresUsername(appDSN)
	if err != nil {
		return fmt.Errorf("parse application database URL: %w", err)
	}

	db, err := openPostgres(appDSN)
	if err != nil {
		return err
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	exists, err := schemaExists(ctx, db)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	if strings.TrimSpace(adminDSN) == "" {
		adminDSN = appDSN
	}

	adminDB, err := openPostgres(adminDSN)
	if err != nil {
		return fmt.Errorf("connect with bootstrap database URL: %w", err)
	}
	defer adminDB.Close()

	adminCtx, adminCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer adminCancel()

	if err := createBudgetSchema(adminCtx, adminDB, appUser); err != nil {
		return err
	}

	return nil
}

func openPostgres(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database connection: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, fmt.Errorf("ping database: %w (close failed: %v)", err, closeErr)
		}
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}

func schemaExists(ctx context.Context, db *sql.DB) (bool, error) {
	var exists bool
	err := db.QueryRowContext(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = $1)`,
		budgetSchemaName,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check for budget schema: %w", err)
	}

	return exists, nil
}

func createBudgetSchema(ctx context.Context, db *sql.DB, appUser string) error {
	quotedUser := quoteIdentifier(appUser)

	statements := []string{
		fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", budgetSchemaName),
		fmt.Sprintf("GRANT ALL PRIVILEGES ON SCHEMA %s TO %s", budgetSchemaName, quotedUser),
		fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT ALL ON TABLES TO %s", budgetSchemaName, quotedUser),
		fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT ALL ON SEQUENCES TO %s", budgetSchemaName, quotedUser),
		fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT ALL ON FUNCTIONS TO %s", budgetSchemaName, quotedUser),
		fmt.Sprintf("ALTER SCHEMA %s OWNER TO %s", budgetSchemaName, quotedUser),
	}

	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("bootstrap budget schema: %w", err)
		}
	}

	return nil
}

func postgresUsername(dsn string) (string, error) {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}

	user := parsed.User.Username()
	if user == "" {
		return "", fmt.Errorf("database URL is missing a username")
	}

	return user, nil
}

func quoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}
