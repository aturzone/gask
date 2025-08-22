package logger

import (
	"fmt"
	"os"

	"github.com/taskmaster/core/internal/infrastructure/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps zap.SugaredLogger to provide application-specific logging
type Logger struct {
	*zap.SugaredLogger
}

// New creates a new logger instance
func New(cfg config.LoggerConfig) (*Logger, error) {
	var zapConfig zap.Config

	if cfg.Format == "json" {
		zapConfig = zap.NewProductionConfig()
	} else {
		zapConfig = zap.NewDevelopmentConfig()
	}

	// Set log level
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}
	zapConfig.Level = zap.NewAtomicLevelAt(level)

	// Configure output
	if cfg.Output == "file" && cfg.Filename != "" {
		zapConfig.OutputPaths = []string{cfg.Filename}
		zapConfig.ErrorOutputPaths = []string{cfg.Filename}
	} else {
		zapConfig.OutputPaths = []string{"stdout"}
		zapConfig.ErrorOutputPaths = []string{"stderr"}
	}

	// Add caller information in development
	if cfg.Format != "json" {
		zapConfig.Development = true
		zapConfig.DisableStacktrace = false
	}

	// Build logger
	zapLogger, err := zapConfig.Build(
		zap.AddCallerSkip(1), // Skip one level to show the actual caller
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	return &Logger{
		SugaredLogger: zapLogger.Sugar(),
	}, nil
}

// WithFields adds structured fields to the logger
func (l *Logger) WithFields(fields ...interface{}) *Logger {
	return &Logger{
		SugaredLogger: l.SugaredLogger.With(fields...),
	}
}

// WithError adds an error field to the logger
func (l *Logger) WithError(err error) *Logger {
	return l.WithFields("error", err.Error())
}

// WithRequestID adds a request ID field to the logger
func (l *Logger) WithRequestID(requestID string) *Logger {
	return l.WithFields("request_id", requestID)
}

// WithUserID adds a user ID field to the logger
func (l *Logger) WithUserID(userID string) *Logger {
	return l.WithFields("user_id", userID)
}

// WithComponent adds a component field to the logger
func (l *Logger) WithComponent(component string) *Logger {
	return l.WithFields("component", component)
}

// HTTP request logging helpers
func (l *Logger) LogHTTPRequest(method, path, userAgent, ip string, statusCode int, duration float64) {
	l.Infow("HTTP request",
		"method", method,
		"path", path,
		"status_code", statusCode,
		"duration_ms", duration,
		"user_agent", userAgent,
		"ip", ip,
	)
}

// Database operation logging helpers
func (l *Logger) LogDatabaseQuery(query string, duration float64, err error) {
	fields := []interface{}{
		"query", query,
		"duration_ms", duration,
	}

	if err != nil {
		fields = append(fields, "error", err.Error())
		l.Errorw("Database query failed", fields...)
	} else {
		l.Debugw("Database query executed", fields...)
	}
}

// Business logic logging helpers
func (l *Logger) LogUserAction(userID, action string, metadata map[string]interface{}) {
	fields := []interface{}{
		"user_id", userID,
		"action", action,
	}

	for k, v := range metadata {
		fields = append(fields, k, v)
	}

	l.Infow("User action", fields...)
}

func (l *Logger) LogSecurityEvent(event, userID, ip string, details map[string]interface{}) {
	fields := []interface{}{
		"security_event", event,
		"user_id", userID,
		"ip", ip,
	}

	for k, v := range details {
		fields = append(fields, k, v)
	}

	l.Warnw("Security event", fields...)
}

// Close flushes any buffered log entries
func (l *Logger) Close() error {
	return l.SugaredLogger.Sync()
}

// Global logger instance for application-wide use
var globalLogger *Logger

// InitGlobal initializes the global logger
func InitGlobal(cfg config.LoggerConfig) error {
	logger, err := New(cfg)
	if err != nil {
		return err
	}
	globalLogger = logger
	return nil
}

// GetGlobal returns the global logger instance
func GetGlobal() *Logger {
	if globalLogger == nil {
		// Fallback to development logger if global logger is not initialized
		logger, _ := New(config.LoggerConfig{
			Level:  "info",
			Format: "console",
			Output: "stdout",
		})
		return logger
	}
	return globalLogger
}

// Structured logging functions for common use cases
func Info(msg string, fields ...interface{}) {
	GetGlobal().Infow(msg, fields...)
}

func Debug(msg string, fields ...interface{}) {
	GetGlobal().Debugw(msg, fields...)
}

func Warn(msg string, fields ...interface{}) {
	GetGlobal().Warnw(msg, fields...)
}

func Error(msg string, fields ...interface{}) {
	GetGlobal().Errorw(msg, fields...)
}

func Fatal(msg string, fields ...interface{}) {
	GetGlobal().Fatalw(msg, fields...)
}

// HTTP middleware logger that can be used with Echo
func HTTPMiddleware() echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:      true,
		LogStatus:   true,
		LogMethod:   true,
		LogLatency:  true,
		LogError:    true,
		LogRemoteIP: true,
		LogUserAgent: true,
		LogValuesFunc: func(c echo.Context, values middleware.RequestLoggerValues) error {
			logger := GetGlobal()
			
			fields := []interface{}{
				"method", values.Method,
				"uri", values.URI,
				"status", values.Status,
				"latency_ms", float64(values.Latency.Nanoseconds()) / 1000000,
				"remote_ip", values.RemoteIP,
				"user_agent", values.UserAgent,
			}

			if values.Error != nil {
				fields = append(fields, "error", values.Error.Error())
				logger.Errorw("HTTP request failed", fields...)
			} else {
				logger.Infow("HTTP request", fields...)
			}

			return nil
		},
	})
}