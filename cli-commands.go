package commands

import (
	"fmt"
	"log"
	"os"

	"github.com/taskmaster/core/internal/infrastructure/config"
	"github.com/taskmaster/core/internal/infrastructure/database"
	"github.com/taskmaster/core/internal/infrastructure/logger"
	"github.com/taskmaster/core/internal/infrastructure/server"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
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
		Long:  "Create and manage users in the system",
	}

	createUserCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new user",
		Run: func(cmd *cobra.Command, args []string) {
			email, _ := cmd.Flags().GetString("email")
			password, _ := cmd.Flags().GetString("password")
			role, _ := cmd.Flags().GetString("role")
			firstName, _ := cmd.Flags().GetString("first-name")
			lastName, _ := cmd.Flags().GetString("last-name")

			if email == "" || password == "" {
				log.Fatal("Email and password are required")
			}

			createUser(email, password, role, firstName, lastName)
		},
	}

	createUserCmd.Flags().String("email", "", "User email (required)")
	createUserCmd.Flags().String("password", "", "User password (required)")
	createUserCmd.Flags().String("role", "developer", "User role (admin, project_manager, team_lead, developer, viewer)")
	createUserCmd.Flags().String("first-name", "", "User first name")
	createUserCmd.Flags().String("last-name", "", "User last name")

	userCmd.AddCommand(createUserCmd)
	return userCmd
}

// NewVersionCommand creates the version command
func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print TaskMaster version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("TaskMaster Core v1.0.0")
			fmt.Println("Build Date: 2024-01-01")
			fmt.Println("Git Commit: development")
		},
	}
}

func runServer() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	appLogger, err := logger.New(cfg.Logger)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer appLogger.Sync()

	db, err := database.NewConnection(cfg.Database)
	if err != nil {
		appLogger.Fatal("Failed to connect to database", "error", err)
	}
	defer db.Close()

	srv, err := server.New(cfg, db, appLogger)
	if err != nil {
		appLogger.Fatal("Failed to initialize server", "error", err)
	}

	appLogger.Info("Starting TaskMaster API server", 
		"port", cfg.Server.Port,
		"environment", cfg.App.Environment,
	)

	if err := srv.Start(fmt.Sprintf(":%d", cfg.Server.Port)); err != nil {
		appLogger.Fatal("Server failed to start", "error", err)
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
		log.Fatalf("Failed to get migration version: %v", err)
	}

	fmt.Printf("Current migration version: %d\n", version)
	fmt.Printf("Dirty: %t\n", dirty)
}

func createUser(email, password, role, firstName, lastName string) {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	db, err := database.NewConnection(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	// Generate username from email
	username := email[:len(email)-len("@example.com")]
	if len(username) > 50 {
		username = username[:50]
	}

	query := `
		INSERT INTO users (email, username, password_hash, first_name, last_name, role, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, true)
		RETURNING id`

	var userID int
	err = db.QueryRow(query, email, username, string(hashedPassword), firstName, lastName, role).Scan(&userID)
	if err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}

	fmt.Printf("User created successfully:\n")
	fmt.Printf("  ID: %d\n", userID)
	fmt.Printf("  Email: %s\n", email)
	fmt.Printf("  Username: %s\n", username)
	fmt.Printf("  Role: %s\n", role)
	if firstName != "" {
		fmt.Printf("  Name: %s %s\n", firstName, lastName)
	}
}