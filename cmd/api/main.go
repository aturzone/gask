package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/taskmaster/core/internal/infrastructure/config"
	"github.com/taskmaster/core/internal/infrastructure/database"
	"github.com/taskmaster/core/internal/infrastructure/logger"
	"github.com/taskmaster/core/internal/infrastructure/server"
	"github.com/taskmaster/core/cmd/api/commands"
	"github.com/spf13/cobra"
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
		Short: "TaskMaster - Enterprise Task Management System",
		Long: `TaskMaster is a comprehensive task management system designed for 
enterprise environments with multi-project support, team collaboration,
and advanced resource management capabilities.`,
		Run: func(cmd *cobra.Command, args []string) {
			runServer()
		},
	}

	// Add subcommands
	rootCmd.AddCommand(commands.NewServeCommand())
	rootCmd.AddCommand(commands.NewMigrateCommand())
	rootCmd.AddCommand(commands.NewUserCommand())
	rootCmd.AddCommand(commands.NewVersionCommand())

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runServer() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	appLogger, err := logger.New(cfg.Logger)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer appLogger.Sync()

	// Initialize database
	db, err := database.NewConnection(cfg.Database)
	if err != nil {
		appLogger.Fatal("Failed to connect to database", "error", err)
	}
	defer db.Close()

	// Ping database to ensure connection
	if err := db.Ping(); err != nil {
		appLogger.Fatal("Failed to ping database", "error", err)
	}

	appLogger.Info("Successfully connected to database")

	// Initialize server
	srv, err := server.New(cfg, db, appLogger)
	if err != nil {
		appLogger.Fatal("Failed to initialize server", "error", err)
	}

	// Start server in a goroutine
	go func() {
		appLogger.Info("Starting TaskMaster API server", 
			"port", cfg.Server.Port,
			"environment", cfg.App.Environment,
		)
		
		if err := srv.Start(fmt.Sprintf(":%d", cfg.Server.Port)); err != nil {
			appLogger.Fatal("Server failed to start", "error", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down server...")

	// Create a deadline for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		appLogger.Fatal("Server forced to shutdown", "error", err)
	}

	appLogger.Info("Server exited gracefully")
}