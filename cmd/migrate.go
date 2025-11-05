package cmd

import (
	"github.com/densestvoid/budget/data"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const defaultDSN = "postgres://postgres:password@localhost:5432/budget?sslmode=disable"

// runMigration executes a migration operation with proper setup
func runMigration(operation func(*data.Storage) error, operationName string) error {
	dsn := viper.GetString("DATABASE_URL")
	if dsn == "" {
		dsn = defaultDSN
	}

	// Initialize migrations
	data.InitMigrations()

	// Get database connection
	db, err := data.GetDBConnection(dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Create storage instance
	store := data.NewStorage(db)

	// Run the migration operation
	if err := operation(store); err != nil {
		return fmt.Errorf("%s failed: %w", operationName, err)
	}

	return nil
}

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
	Long: `Run database migrations using embedded migration files.
This command will apply pending migrations to the database.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runMigration(data.Up, "Migration"); err != nil {
			log.Fatalf("%v", err)
		}
		fmt.Println("Migrations completed successfully")
	},
}

// migrateDownCmd represents the migrate down command
var migrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Rollback database migrations",
	Long: `Rollback the last database migration.
This command will undo the most recent migration.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runMigration(data.Down, "Migration rollback"); err != nil {
			log.Fatalf("%v", err)
		}
		fmt.Println("Migration rollback completed successfully")
	},
}

// migrateStatusCmd represents the migrate status command
var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration status",
	Long: `Show the current status of database migrations.
This command will display which migrations have been applied and which are pending.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runMigration(data.Status, "Getting migration status"); err != nil {
			log.Fatalf("%v", err)
		}
	},
}

func init() {
	migrateCmd.AddCommand(migrateDownCmd)
	migrateCmd.AddCommand(migrateStatusCmd)
	rootCmd.AddCommand(migrateCmd)
}
