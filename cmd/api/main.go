package main

import (
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/taskmaster/core/cmd/api/commands"
)

// @title TaskMaster API
// @version 1.0
// @description Enterprise multi-project task management system
// @termsOfService https://github.com/taskmaster/core/blob/main/LICENSE

// @contact.name TaskMaster Support
// @contact.url https://github.com/taskmaster/core
// @contact.email support@taskmaster.dev

// @license.name MIT
// @license.url https://github.com/taskmaster/core/blob/main/LICENSE

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	rootCmd := &cobra.Command{
		Use:   "taskmaster",
		Short: "TaskMaster API Server",
		Long:  `TaskMaster is an enterprise multi-project task management system with advanced features for team collaboration and project tracking.`,
	}

	// Add commands
	rootCmd.AddCommand(commands.NewServeCommand())
	rootCmd.AddCommand(commands.NewMigrateCommand())
	rootCmd.AddCommand(commands.NewUserCommand())

	// Execute root command
	if err := rootCmd.Execute(); err != nil {
		log.Printf("Command execution failed: %v", err)
		os.Exit(1)
	}
}
