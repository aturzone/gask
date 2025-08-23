package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Logger   LoggerConfig   `mapstructure:"logger"`
	Security SecurityConfig `mapstructure:"security"`
	Metrics  MetricsConfig  `mapstructure:"metrics"`
}

// AppConfig holds application-specific configuration
type AppConfig struct {
	Name        string `mapstructure:"name"`
	Version     string `mapstructure:"version"`
	Environment string `mapstructure:"environment"`
	Debug       bool   `mapstructure:"debug"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	Host         string        `mapstructure:"host"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Name            string        `mapstructure:"name"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret           string        `mapstructure:"secret"`
	ExpiresIn        time.Duration `mapstructure:"expires_in"`
	RefreshExpiresIn time.Duration `mapstructure:"refresh_expires_in"`
	Issuer           string        `mapstructure:"issuer"`
}

// LoggerConfig holds logging configuration
type LoggerConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	CORSAllowedOrigins string        `mapstructure:"cors_allowed_origins"`
	RateLimitRequests  int           `mapstructure:"rate_limit_requests"`
	RateLimitWindow    time.Duration `mapstructure:"rate_limit_window"`
}

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
	Enabled bool `mapstructure:"enabled"`
	Port    int  `mapstructure:"port"`
}

// Load loads configuration from various sources
func Load() (*Config, error) {
	// Load .env file if it exists (ignore errors)
	_ = godotenv.Load()

	// Configure viper
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set defaults
	setDefaults()

	// Bind environment variables
	bindEnvVars()

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

func setDefaults() {
	// App defaults
	viper.SetDefault("app.name", "TaskMaster")
	viper.SetDefault("app.version", "1.0.0")
	viper.SetDefault("app.environment", "development")
	viper.SetDefault("app.debug", false)

	// Server defaults
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")
	viper.SetDefault("server.idle_timeout", "120s")

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.name", "taskmaster")
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "")
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 10)
	viper.SetDefault("database.conn_max_lifetime", "5m")
	viper.SetDefault("database.conn_max_idle_time", "30s")

	// Redis defaults
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// JWT defaults
	viper.SetDefault("jwt.secret", "your-super-secret-jwt-key")
	viper.SetDefault("jwt.expires_in", "24h")
	viper.SetDefault("jwt.refresh_expires_in", "168h") // 7 days
	viper.SetDefault("jwt.issuer", "taskmaster-api")

	// Logger defaults
	viper.SetDefault("logger.level", "info")
	viper.SetDefault("logger.format", "json")
	viper.SetDefault("logger.output", "stdout")

	// Security defaults
	viper.SetDefault("security.cors_allowed_origins", "*")
	viper.SetDefault("security.rate_limit_requests", 100)
	viper.SetDefault("security.rate_limit_window", "1m")

	// Metrics defaults
	viper.SetDefault("metrics.enabled", true)
	viper.SetDefault("metrics.port", 9090)
}

func bindEnvVars() {
	// App
	viper.BindEnv("app.name", "APP_NAME")
	viper.BindEnv("app.version", "APP_VERSION")
	viper.BindEnv("app.environment", "APP_ENVIRONMENT")
	viper.BindEnv("app.debug", "APP_DEBUG")

	// Server
	viper.BindEnv("server.port", "SERVER_PORT")
	viper.BindEnv("server.host", "SERVER_HOST")
	viper.BindEnv("server.read_timeout", "SERVER_READ_TIMEOUT")
	viper.BindEnv("server.write_timeout", "SERVER_WRITE_TIMEOUT")
	viper.BindEnv("server.idle_timeout", "SERVER_IDLE_TIMEOUT")

	// Database
	viper.BindEnv("database.host", "DB_HOST")
	viper.BindEnv("database.port", "DB_PORT")
	viper.BindEnv("database.name", "DB_NAME")
	viper.BindEnv("database.user", "DB_USER")
	viper.BindEnv("database.password", "DB_PASSWORD")
	viper.BindEnv("database.ssl_mode", "DB_SSL_MODE")
	viper.BindEnv("database.max_open_conns", "DB_MAX_OPEN_CONNS")
	viper.BindEnv("database.max_idle_conns", "DB_MAX_IDLE_CONNS")
	viper.BindEnv("database.conn_max_lifetime", "DB_CONN_MAX_LIFETIME")
	viper.BindEnv("database.conn_max_idle_time", "DB_CONN_MAX_IDLE_TIME")

	// Redis
	viper.BindEnv("redis.host", "REDIS_HOST")
	viper.BindEnv("redis.port", "REDIS_PORT")
	viper.BindEnv("redis.password", "REDIS_PASSWORD")
	viper.BindEnv("redis.db", "REDIS_DB")

	// JWT
	viper.BindEnv("jwt.secret", "JWT_SECRET")
	viper.BindEnv("jwt.expires_in", "JWT_EXPIRES_IN")
	viper.BindEnv("jwt.refresh_expires_in", "JWT_REFRESH_EXPIRES_IN")
	viper.BindEnv("jwt.issuer", "JWT_ISSUER")

	// Logger
	viper.BindEnv("logger.level", "LOG_LEVEL")
	viper.BindEnv("logger.format", "LOG_FORMAT")
	viper.BindEnv("logger.output", "LOG_OUTPUT")

	// Security
	viper.BindEnv("security.cors_allowed_origins", "CORS_ALLOWED_ORIGINS")
	viper.BindEnv("security.rate_limit_requests", "RATE_LIMIT_REQUESTS")
	viper.BindEnv("security.rate_limit_window", "RATE_LIMIT_WINDOW")

	// Metrics
	viper.BindEnv("metrics.enabled", "ENABLE_METRICS")
	viper.BindEnv("metrics.port", "METRICS_PORT")
}

func validateConfig(cfg *Config) error {
	if cfg.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if cfg.Database.Name == "" {
		return fmt.Errorf("database name is required")
	}

	if cfg.JWT.Secret == "" || cfg.JWT.Secret == "your-super-secret-jwt-key" {
		return fmt.Errorf("JWT secret must be set and should not use default value")
	}

	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535")
	}

	return nil
}

// GetDatabaseDSN returns the database connection string
func (cfg *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.Name,
		cfg.SSLMode,
	)
}

// GetRedisAddr returns the Redis address
func (cfg *RedisConfig) GetAddr() string {
	return fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
}

// IsDevelopment returns true if the environment is development
func (cfg *AppConfig) IsDevelopment() bool {
	return cfg.Environment == "development"
}

// IsProduction returns true if the environment is production
func (cfg *AppConfig) IsProduction() bool {
	return cfg.Environment == "production"
}
