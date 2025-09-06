package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	port    string
	env     string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "budget",
	Short: "A modern budget management web application",
	Long: `Budget App is a modern web application built with Go, featuring:
- Chi router for HTTP routing
- PostgreSQL database with Goose migrations
- HTMX for dynamic interactions
- Alpine.js for reactive UI
- Bootstrap 5 for styling
- Gomponents for HTML generation`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.budget.yaml)")
	rootCmd.PersistentFlags().StringVarP(&port, "port", "p", "8080", "port to run the server on")
	rootCmd.PersistentFlags().StringVarP(&env, "env", "e", "development", "environment (development, production, test)")

	// Bind flags to viper
	if err := viper.BindPFlag("port", rootCmd.PersistentFlags().Lookup("port")); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding port flag: %v\n", err)
	}
	if err := viper.BindPFlag("env", rootCmd.PersistentFlags().Lookup("env")); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding env flag: %v\n", err)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".budget" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".budget")
	}

	// Search config in the working directory
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")

	// Environment variables
	viper.SetEnvPrefix("BUDGET")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}

	// Set defaults
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("ENV", "development")
	viper.SetDefault("DATABASE_URL", "postgres://postgres:password@localhost:5432/budget?sslmode=disable")
	viper.SetDefault("LOG_LEVEL", "info")
}
