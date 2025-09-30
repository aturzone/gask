package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	// Application
	AppName     string
	Environment string
	LogLevel    string

	// API Server
	APIPort    int
	APIHost    string
	APITimeout time.Duration

	// Redis
	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDB       int

	// PostgreSQL
	PostgresHost     string
	PostgresPort     int
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
	PostgresSSLMode  string

	// Authentication
	OwnerPassword string
	OwnerEmail    string

	// Sync Service
	SyncInterval time.Duration

	// Timezone
	Timezone string
}

var AppConfig *Config

// Load configuration from environment variables
func Load() (*Config, error) {
	config := &Config{
		AppName:     getEnv("APP_NAME", "gask"),
		Environment: getEnv("ENVIRONMENT", "production"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),

		APIHost:    getEnv("API_HOST", "0.0.0.0"),
		APIPort:    getEnvAsInt("API_PORT", 7890),
		APITimeout: getEnvAsDuration("API_TIMEOUT", 15*time.Second),

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnvAsInt("REDIS_PORT", 6380),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvAsInt("REDIS_DB", 0),

		PostgresHost:     getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:     getEnvAsInt("POSTGRES_PORT", 5433),
		PostgresUser:     getEnv("POSTGRES_USER", "airflow"),
		PostgresPassword: getEnv("POSTGRES_PASSWORD", "EKQH9jQX7gAfV7pLwVmsbLbF3XfY6n4S"),
		PostgresDB:       getEnv("POSTGRES_DB", "airflow"),
		PostgresSSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),

		OwnerPassword: getEnv("OWNER_PASSWORD", "admin1234"),
		OwnerEmail:    getEnv("OWNER_EMAIL", "admin@gmail.com"),

		SyncInterval: getEnvAsDuration("SYNC_INTERVAL", 15*time.Minute),
		Timezone:     getEnv("TZ", "Asia/Tehran"),
	}

	// Find available API port if configured port is busy
	if getEnvAsBool("AUTO_PORT_FIND", true) {
		availablePort, err := findAvailablePort(config.APIPort, config.APIPort+100)
		if err != nil {
			return nil, fmt.Errorf("failed to find available port: %v", err)
		}
		config.APIPort = availablePort
	}

	AppConfig = config
	return config, nil
}

// GetRedisAddr returns Redis connection address
func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.RedisHost, c.RedisPort)
}

// GetPostgresDSN returns PostgreSQL connection string
func (c *Config) GetPostgresDSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
		c.PostgresHost,
		c.PostgresUser,
		c.PostgresPassword,
		c.PostgresDB,
		c.PostgresPort,
		c.PostgresSSLMode,
		c.Timezone,
	)
}

// GetAPIAddr returns API server address
func (c *Config) GetAPIAddr() string {
	return fmt.Sprintf("%s:%d", c.APIHost, c.APIPort)
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if value, err := time.ParseDuration(valueStr); err == nil {
		return value
	}
	return defaultValue
}

// findAvailablePort finds the first available port in the given range
func findAvailablePort(startPort, endPort int) (int, error) {
	for port := startPort; port <= endPort; port++ {
		if isPortAvailable(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port found in range %d-%d", startPort, endPort)
}

// isPortAvailable checks if a port is available
func isPortAvailable(port int) bool {
	address := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// Print configuration (safe - without passwords)
func (c *Config) Print() {
	fmt.Println("🔧 GASK Configuration")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  App Name:      %s\n", c.AppName)
	fmt.Printf("  Environment:   %s\n", c.Environment)
	fmt.Printf("  API Address:   http://%s\n", c.GetAPIAddr())
	fmt.Printf("  Redis:         %s\n", c.GetRedisAddr())
	fmt.Printf("  PostgreSQL:    %s:%d/%s\n", c.PostgresHost, c.PostgresPort, c.PostgresDB)
	fmt.Printf("  Sync Interval: %v\n", c.SyncInterval)
	fmt.Printf("  Timezone:      %s\n", c.Timezone)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
