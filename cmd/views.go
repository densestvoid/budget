package cmd

import (
	"github.com/densestvoid/budget/data"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// viewsCmd represents the views command
var viewsCmd = &cobra.Command{
	Use:   "views",
	Short: "Manage database views",
	Long: `Manage database views. Views are stored in data/views/ and use CREATE OR REPLACE,
making them safe to apply multiple times.`,
}

// viewsApplyCmd applies all views to the database
var viewsApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply all database views",
	Long: `Apply all views from the data/views/ directory to the database.
Views use CREATE OR REPLACE, so this is safe to run multiple times.`,
	Run: func(cmd *cobra.Command, args []string) {
		dsn := viper.GetString("DATABASE_URL")
		if dsn == "" {
			dsn = defaultDSN
		}

		// Get database connection
		db, err := data.GetDBConnection(dsn)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		defer db.Close()

		// Apply views
		if err := data.ApplyViews(db); err != nil {
			log.Fatalf("Failed to apply views: %v", err)
		}

		fmt.Println("Views applied successfully")
	},
}

// viewsListCmd lists all available views
var viewsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available views",
	Long:  `List all view files available in the data/views/ directory.`,
	Run: func(cmd *cobra.Command, args []string) {
		views, err := data.ListViews()
		if err != nil {
			log.Fatalf("Failed to list views: %v", err)
		}

		if len(views) == 0 {
			fmt.Println("No views found")
			return
		}

		fmt.Println("Available views:")
		for _, view := range views {
			fmt.Printf("  - %s\n", view)
		}
	},
}

func init() {
	viewsCmd.AddCommand(viewsApplyCmd)
	viewsCmd.AddCommand(viewsListCmd)
	rootCmd.AddCommand(viewsCmd)
}

