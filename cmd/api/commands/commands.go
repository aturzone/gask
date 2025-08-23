package commands

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"

	"github.com/taskmaster/core/internal/infrastructure/config"
	"github.com/taskmaster/core/internal/infrastructure/database"
	"github.com/taskmaster/core/internal/infrastructure/logger"
	"github.com/taskmaster/core/internal/infrastructure/server"
)

// NewServeCommand creates the serve command
func NewServeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the TaskMaster API server",
		Long:  "Start the TaskMaster API server with all configured routes and middleware",
		Run: func(cmd *cobra.Command, args []string) {
			runServer()
		},
	}
}

// NewMigrateCommand creates the migrate command with subcommands
func NewMigrateCommand() *cobra.Command {
	migrateCmd := &cobra.Command{
		Use:   "migrate",
		Short: "Database migration commands",
		Long:  "Manage database migrations (up, down, version)",
	}

	migrateCmd.AddCommand(&cobra.Command{
		Use:   "up",
		Short: "Run all up migrations",
		Run: func(cmd *cobra.Command, args []string) {
			runMigration("up", 0)
		},
	})

	migrateCmd.AddCommand(&cobra.Command{
		Use:   "down",
		Short: "Run all down migrations",
		Run: func(cmd *cobra.Command, args []string) {
			runMigration("down", 0)
		},
	})

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print current migration version",
		Run: func(cmd *cobra.Command, args []string) {
			showMigrationVersion()
		},
	}
	migrateCmd.AddCommand(versionCmd)

	return migrateCmd
}

// NewUserCommand creates the user management command
func NewUserCommand() *cobra.Command {
	userCmd := &cobra.Command{
		Use:   "user",
		Short: "User management commands",
		Long:  "Create and manage users",
	}

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new user",
		Run: func(cmd *cobra.Command, args []string) {
			createUser()
		},
	}

	userCmd.AddCommand(createCmd)

	return userCmd
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

	// Connect to database
	db, err := database.New(cfg.Database)
	if err != nil {
		appLogger.Fatal("Failed to connect to database", "error", err)
	}
	defer db.Close()

	// Create server
	srv, err := server.New(cfg, db, appLogger)
	if err != nil {
		appLogger.Fatal("Failed to initialize server", "error", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		appLogger.Info("Starting TaskMaster API server", 
			"port", cfg.Server.Port,
			"environment", cfg.App.Environment,
		)

		address := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
		if err := srv.Start(address); err != nil {
			appLogger.Fatal("Server failed to start", "error", err)
		}
	}()

	// Wait for interrupt signal
	<-sigChan
	appLogger.Info("Received interrupt signal, shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		appLogger.Error("Server shutdown error", "error", err)
	} else {
		appLogger.Info("Server shutdown completed")
	}
}

func runMigration(direction string, steps int) {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	db, err := database.NewConnection(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		log.Fatalf("Failed to create migration driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		log.Fatalf("Failed to create migration instance: %v", err)
	}

	switch direction {
	case "up":
		if steps > 0 {
			err = m.Steps(steps)
		} else {
			err = m.Up()
		}
	case "down":
		if steps > 0 {
			err = m.Steps(-steps)
		} else {
			err = m.Down()
		}
	}

	if err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Migration failed: %v", err)
	}

	if err == migrate.ErrNoChange {
		fmt.Println("No migrations to run")
	} else {
		fmt.Printf("Migration %s completed successfully\n", direction)
	}
}

func showMigrationVersion() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	db, err := database.NewConnection(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		log.Fatalf("Failed to create migration driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		log.Fatalf("Failed to create migration instance: %v", err)
	}

	version, dirty, err := m.Version()
	if err != nil {
		if err == migrate.ErrNilVersion {
			fmt.Println("Database version: No migrations applied")
		} else {
			log.Fatalf("Failed to get migration version: %v", err)
		}
		return
	}

	status := "clean"
	if dirty {
		status = "dirty"
	}

	fmt.Printf("Database version: %d (%s)\n", version, status)
}

func createUser() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	db, err := database.NewConnection(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// For now, create a simple admin user
	// In a real implementation, this would take input parameters
	email := "admin@taskmaster.dev"
	username := "admin"
	password := "admin123"
	role := "admin"

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	// Insert user
	query := `
		INSERT INTO users (email, username, password_hash, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (email) DO NOTHING
	`

	result, err := db.DB.Exec(query, email, username, string(hashedPassword), role, true)
	if err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Fatalf("Failed to get rows affected: %v", err)
	}

	if rowsAffected > 0 {
		fmt.Printf("Created admin user: %s (password: %s)\n", email, password)
	} else {
		fmt.Printf("User %s already exists\n", email)
	}
}
