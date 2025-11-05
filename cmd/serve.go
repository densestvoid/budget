package cmd

import (
	"github.com/densestvoid/budget/app"
	"github.com/densestvoid/budget/server"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web server",
	Long: `Start the Budget App web server with the specified configuration.
The server will listen on the configured port and serve the web application.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Create application instance
		app := app.NewApp()

		// Connect to database
		dsn := viper.GetString("DATABASE_URL")
		if dsn == "" {
			dsn = "postgres://postgres:password@localhost:5432/budget?sslmode=disable"
		}

		if err := app.ConnectDB(dsn); err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		defer func() {
			if err := app.CloseDB(); err != nil {
				log.Printf("Error closing database connection: %v", err)
			}
		}()

		// Create and configure server
		port := viper.GetString("PORT")
		if port == "" {
			port = "8080"
		}

		srv := server.NewServer(port)
		srv.SetDB(app.GetDB())
		srv.SetupMiddleware()
		srv.SetupRoutes()

		// Run the server
		if err := srv.Run(); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
