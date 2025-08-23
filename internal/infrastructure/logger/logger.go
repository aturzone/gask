package logger

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/taskmaster/core/internal/infrastructure/config"
)

// Logger wraps slog.Logger with additional functionality
type Logger struct {
	*slog.Logger
}

// New creates a new logger instance
func New(cfg config.LoggerConfig) (*Logger, error) {
	// Configure log level
	var level slog.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Configure output
	var output *os.File = os.Stdout
	if cfg.Output != "" && cfg.Output != "stdout" {
		if cfg.Output == "stderr" {
			output = os.Stderr
		} else {
			// File output
			file, err := os.OpenFile(cfg.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				return nil, fmt.Errorf("failed to open log file: %w", err)
			}
			output = file
		}
	}

	// Configure handler based on format
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: level,
		AddSource: cfg.Level == "debug",
	}

	switch strings.ToLower(cfg.Format) {
	case "json":
		handler = slog.NewJSONHandler(output, opts)
	case "text":
		handler = slog.NewTextHandler(output, opts)
	default:
		handler = slog.NewJSONHandler(output, opts)
	}

	logger := slog.New(handler)

	return &Logger{
		Logger: logger,
	}, nil
}

// Info logs an info message with key-value pairs
func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	l.Logger.Info(msg, keysAndValues...)
}

// Debug logs a debug message with key-value pairs
func (l *Logger) Debug(msg string, keysAndValues ...interface{}) {
	l.Logger.Debug(msg, keysAndValues...)
}

// Warn logs a warning message with key-value pairs
func (l *Logger) Warn(msg string, keysAndValues ...interface{}) {
	l.Logger.Warn(msg, keysAndValues...)
}

// Error logs an error message with key-value pairs
func (l *Logger) Error(msg string, keysAndValues ...interface{}) {
	l.Logger.Error(msg, keysAndValues...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, keysAndValues ...interface{}) {
	l.Logger.Error(msg, keysAndValues...)
	os.Exit(1)
}

// Infow logs an info message with structured fields (for compatibility)
func (l *Logger) Infow(msg string, keysAndValues ...interface{}) {
	l.Logger.Info(msg, keysAndValues...)
}

// Debugw logs a debug message with structured fields (for compatibility)
func (l *Logger) Debugw(msg string, keysAndValues ...interface{}) {
	l.Logger.Debug(msg, keysAndValues...)
}

// Warnw logs a warning message with structured fields (for compatibility)
func (l *Logger) Warnw(msg string, keysAndValues ...interface{}) {
	l.Logger.Warn(msg, keysAndValues...)
}

// Errorw logs an error message with structured fields (for compatibility)
func (l *Logger) Errorw(msg string, keysAndValues ...interface{}) {
	l.Logger.Error(msg, keysAndValues...)
}

// WithFields returns a logger with additional fields
func (l *Logger) WithFields(keysAndValues ...interface{}) *Logger {
	return &Logger{
		Logger: l.Logger.With(keysAndValues...),
	}
}

// LogSecurityEvent logs a security-related event
func (l *Logger) LogSecurityEvent(event string, userID string, ip string, details map[string]interface{}) {
	args := []interface{}{
		"event", event,
		"user_id", userID,
		"ip", ip,
	}

	for key, value := range details {
		args = append(args, key, value)
	}

	l.Logger.Warn("Security event", args...)
}

// LogHTTPRequest logs HTTP request details
func (l *Logger) LogHTTPRequest(method, path string, statusCode int, duration float64, userID string) {
	l.Logger.Info("HTTP request",
		"method", method,
		"path", path,
		"status", statusCode,
		"duration_ms", duration,
		"user_id", userID,
	)
}

// LogDatabaseQuery logs database query details (for debugging)
func (l *Logger) LogDatabaseQuery(query string, duration float64, err error) {
	if err != nil {
		l.Logger.Error("Database query failed",
			"query", query,
			"duration_ms", duration,
			"error", err,
		)
	} else {
		l.Logger.Debug("Database query executed",
			"query", query,
			"duration_ms", duration,
		)
	}
}
