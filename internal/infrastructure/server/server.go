package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/time/rate"

	httpHandlers "github.com/taskmaster/core/internal/adapters/http"
	"github.com/taskmaster/core/internal/adapters/repository"
	"github.com/taskmaster/core/internal/application/services"
	"github.com/taskmaster/core/internal/domain/entities"
	"github.com/taskmaster/core/internal/infrastructure/config"
	"github.com/taskmaster/core/internal/infrastructure/database"
	"github.com/taskmaster/core/internal/infrastructure/logger"
)

// Server represents the HTTP server
type Server struct {
	echo   *echo.Echo
	config *config.Config
	logger *logger.Logger
	db     *database.DB
}

// CustomValidator wraps the validator
type CustomValidator struct {
	validator *validator.Validate
}

// Validate validates structs
func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

// New creates a new server instance
func New(cfg *config.Config, db *database.DB, appLogger *logger.Logger) (*Server, error) {
	e := echo.New()

	// Set custom validator
	e.Validator = &CustomValidator{validator: validator.New()}

	// Configure Echo
	e.HideBanner = true
	e.HidePort = true

	// Custom error handler
	e.HTTPErrorHandler = customErrorHandler(appLogger)

	// Initialize repositories
	userRepo := repository.NewUserRepository(db.DB)
	taskRepo := repository.NewTaskRepository(db.DB)
	projectRepo := repository.NewProjectRepository(db.DB)
	authRepo := repository.NewAuthRepository(db.DB)
	timeRepo := repository.NewTimeEntryRepository(db.DB)

	// Initialize services
	authService := services.NewAuthService(userRepo, authRepo, cfg.JWT, appLogger)
	userService := services.NewUserService(userRepo, appLogger)
	taskService := services.NewTaskService(taskRepo, projectRepo, userRepo, appLogger)
	projectService := services.NewProjectService(projectRepo, userRepo, appLogger)
	timeService := services.NewTimeService(timeRepo, taskRepo, projectRepo, userRepo, appLogger)

	// Initialize handlers
	authHandler := httpHandlers.NewAuthHandler(authService, appLogger)
	userHandler := httpHandlers.NewUserHandler(userService, appLogger)
	taskHandler := httpHandlers.NewTaskHandler(taskService, appLogger)
	projectHandler := httpHandlers.NewProjectHandler(projectService, appLogger)
	timeHandler := httpHandlers.NewTimeHandler(timeService, appLogger)

	server := &Server{
		echo:   e,
		config: cfg,
		logger: appLogger,
		db:     db,
	}

	// Setup middleware
	server.setupMiddleware()

	// Setup routes
	server.setupRoutes(authHandler, userHandler, taskHandler, projectHandler, timeHandler, authService)

	// Setup metrics
	if cfg.Metrics.Enabled {
		server.setupMetrics()
	}

	return server, nil
}

// setupMiddleware configures middleware
func (s *Server) setupMiddleware() {
	// Recovery middleware
	s.echo.Use(middleware.Recover())

	// Logger middleware
	s.echo.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:       true,
		LogStatus:    true,
		LogMethod:    true,
		LogLatency:   true,
		LogError:     true,
		LogRemoteIP:  true,
		LogUserAgent: true,
		LogValuesFunc: func(c echo.Context, values middleware.RequestLoggerValues) error {
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
				s.logger.Errorw("HTTP request failed", fields...)
			} else {
				s.logger.Infow("HTTP request", fields...)
			}

			return nil
		},
	}))

	// CORS middleware
	s.echo.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: strings.Split(s.config.Security.CORSAllowedOrigins, ","),
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		AllowMethods: []string{echo.GET, echo.HEAD, echo.PUT, echo.PATCH, echo.POST, echo.DELETE},
	}))

	// Rate limiting middleware
	s.echo.Use(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{Rate: rate.Limit(s.config.Security.RateLimitRequests), Burst: s.config.Security.RateLimitRequests, ExpiresIn: s.config.Security.RateLimitWindow},
		),
		IdentifierExtractor: func(ctx echo.Context) (string, error) {
			id := ctx.RealIP()
			return id, nil
		},
		ErrorHandler: func(context echo.Context, err error) error {
			return context.JSON(http.StatusForbidden, map[string]string{"message": "rate limit exceeded"})
		},
		DenyHandler: func(context echo.Context, identifier string, err error) error {
			return context.JSON(http.StatusTooManyRequests, map[string]string{"message": "rate limit exceeded"})
		},
	}))

	// Security headers
	s.echo.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "DENY",
		HSTSMaxAge:            31536000,
		ContentSecurityPolicy: "default-src 'self'",
	}))

	// Request ID middleware
	s.echo.Use(middleware.RequestID())

	// Timeout middleware
	s.echo.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: 30 * time.Second,
	}))
}

// setupRoutes configures all routes
func (s *Server) setupRoutes(authHandler *httpHandlers.AuthHandler, userHandler *httpHandlers.UserHandler, taskHandler *httpHandlers.TaskHandler, projectHandler *httpHandlers.ProjectHandler, timeHandler *httpHandlers.TimeHandler, authService *services.AuthService) {
	// Health check routes
	s.echo.GET("/health", s.healthCheck)
	s.echo.GET("/health/detailed", s.detailedHealthCheck)
	s.echo.GET("/ready", s.readinessCheck)

	// Swagger documentation
	s.echo.Static("/docs", "docs")
	s.echo.GET("/swagger.json", func(c echo.Context) error {
		return c.File("docs/swagger.json")
	})
	s.echo.GET("/swagger", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/docs/simple-swagger.html")
	})
	s.echo.GET("/swagger/", func(c echo.Context) error {
		return c.File("docs/simple-swagger.html")
	})
	s.echo.GET("/api-docs", func(c echo.Context) error {
		return c.File("docs/simple-swagger.html")
	})
	s.echo.GET("/documentation", func(c echo.Context) error {
		return c.File("docs/simple-swagger.html")
	})

	// API v1 routes
	v1 := s.echo.Group("/api/v1")

	// Auth routes (public)
	authGroup := v1.Group("/auth")
	authGroup.POST("/register", authHandler.Register)
	authGroup.POST("/login", authHandler.Login)
	authGroup.POST("/refresh", authHandler.RefreshToken)
	authGroup.POST("/logout", authHandler.Logout, s.authMiddleware(authService))

	// User routes (authenticated)
	userGroup := v1.Group("/users", s.authMiddleware(authService))
	userGroup.GET("/me", userHandler.GetCurrentUser)
	userGroup.PUT("/me", userHandler.UpdateCurrentUser)
	userGroup.GET("", userHandler.ListUsers, s.requireRole("admin"))
	userGroup.POST("", userHandler.CreateUser, s.requireRole("admin"))
	userGroup.GET("/:id", userHandler.GetUser, s.requireRole("admin", "manager"))
	userGroup.PUT("/:id", userHandler.UpdateUser, s.requireRole("admin", "manager"))
	userGroup.DELETE("/:id", userHandler.DeleteUser, s.requireRole("admin"))

	// Project routes (authenticated)
	projectGroup := v1.Group("/projects", s.authMiddleware(authService))
	projectGroup.GET("", projectHandler.ListProjects)
	projectGroup.POST("", projectHandler.CreateProject, s.requireRole("admin", "manager"))
	projectGroup.GET("/:id", projectHandler.GetProject)
	projectGroup.PUT("/:id", projectHandler.UpdateProject, s.requireRole("admin", "manager"))
	projectGroup.DELETE("/:id", projectHandler.DeleteProject, s.requireRole("admin"))
	projectGroup.GET("/:id/tasks", projectHandler.GetProjectTasks)

	// Task routes (authenticated)
	taskGroup := v1.Group("/tasks", s.authMiddleware(authService))
	taskGroup.GET("", taskHandler.ListTasks)
	taskGroup.POST("", taskHandler.CreateTask)
	taskGroup.GET("/deadlines", taskHandler.GetDeadlines)
	taskGroup.GET("/:id", taskHandler.GetTask)
	taskGroup.PUT("/:id", taskHandler.UpdateTask)
	taskGroup.DELETE("/:id", taskHandler.DeleteTask)
	taskGroup.POST("/:id/assign", taskHandler.AssignTask, s.requireRole("admin", "manager"))

	// Time tracking routes (authenticated)
	timeGroup := v1.Group("/time", s.authMiddleware(authService))
	timeGroup.GET("", timeHandler.ListTimeEntries)
	timeGroup.POST("", timeHandler.CreateTimeEntry)
	timeGroup.GET("/:id", timeHandler.GetTimeEntry)
	timeGroup.PUT("/:id", timeHandler.UpdateTimeEntry)
	timeGroup.DELETE("/:id", timeHandler.DeleteTimeEntry)
	timeGroup.GET("/reports", timeHandler.GetTimeReport)
}

// setupMetrics configures Prometheus metrics
func (s *Server) setupMetrics() {
	registry := prometheus.NewRegistry()

	requestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	registry.MustRegister(requestsTotal, requestDuration)

	// Custom metrics middleware
	s.echo.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			duration := time.Since(start)
			status := c.Response().Status

			requestsTotal.WithLabelValues(
				c.Request().Method,
				c.Path(),
				fmt.Sprintf("%d", status),
			).Inc()

			requestDuration.WithLabelValues(
				c.Request().Method,
				c.Path(),
			).Observe(duration.Seconds())

			return err
		}
	})

	// Metrics endpoint
	metricsHandler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	s.echo.GET("/metrics", echo.WrapHandler(metricsHandler))
}

// authMiddleware validates JWT tokens
func (s *Server) authMiddleware(authService *services.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Missing authorization header")
			}

			// Extract token from "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid authorization header format")
			}

			token := parts[1]

			// Validate token
			claims, err := authService.ValidateToken(token)
			if err != nil {
				s.logger.Warn("Invalid token", "error", err, "ip", c.RealIP())
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
			}

			// Set user information in context
			c.Set("user", claims.UserID)
			c.Set("user_email", claims.Email)
			c.Set("user_role", claims.Role)

			return next(c)
		}
	}
}

// requireRole middleware checks if user has required role
func (s *Server) requireRole(roles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userRole := c.Get("user_role")
			if userRole == nil {
				return echo.NewHTTPError(http.StatusForbidden, "Role information not found")
			}

			role := string(userRole.(entities.UserRole))
			for _, requiredRole := range roles {
				if role == requiredRole {
					return next(c)
				}
			}

			s.logger.LogSecurityEvent("insufficient_permissions", 
				c.Get("user").(string), 
				c.RealIP(), 
				map[string]interface{}{
					"required_roles": roles,
					"user_role": role,
					"endpoint": c.Request().URL.Path,
				})

			return echo.NewHTTPError(http.StatusForbidden, "Insufficient permissions")
		}
	}
}

// Health check handlers
func (s *Server) healthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) detailedHealthCheck(c echo.Context) error {
	status := "ok"
	checks := make(map[string]interface{})

	// Database health check
	if err := s.db.HealthCheck(); err != nil {
		status = "error"
		checks["database"] = map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		}
	} else {
		checks["database"] = map[string]interface{}{
			"status": "ok",
			"stats":  s.db.GetConnectionInfo(),
		}
	}

	// Add more health checks here (Redis, external services, etc.)

	response := map[string]interface{}{
		"status": status,
		"time":   time.Now().UTC().Format(time.RFC3339),
		"checks": checks,
		"version": map[string]string{
			"app":     s.config.App.Version,
			"go":      "1.21",
		},
	}

	if status == "ok" {
		return c.JSON(http.StatusOK, response)
	}
	return c.JSON(http.StatusServiceUnavailable, response)
}

func (s *Server) readinessCheck(c echo.Context) error {
	// Check if server is ready to accept requests
	if err := s.db.Ping(); err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"status": "not_ready",
			"reason": "database_not_ready",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status": "ready",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// Start starts the HTTP server
func (s *Server) Start(address string) error {
	s.logger.Info("Starting server", "address", address)
	return s.echo.Start(address)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down server")
	return s.echo.Shutdown(ctx)
}

// customErrorHandler handles HTTP errors
func customErrorHandler(logger *logger.Logger) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		var (
			code = http.StatusInternalServerError
			msg  interface{}
		)

		if he, ok := err.(*echo.HTTPError); ok {
			code = he.Code
			msg = he.Message
			if he.Internal != nil {
				err = fmt.Errorf("%v, %v", err, he.Internal)
			}
		} else if e, ok := err.(*validator.ValidationErrors); ok {
			code = http.StatusBadRequest
			msg = map[string]string{"message": "validation failed", "details": e.Error()}
		} else {
			msg = map[string]string{"message": http.StatusText(code)}
		}

		if code == http.StatusInternalServerError {
			logger.Error("Internal server error", "error", err, "path", c.Request().URL.Path)
		}

		// Send response
		if !c.Response().Committed {
			if c.Request().Method == echo.HEAD {
				err = c.NoContent(code)
			} else {
				err = c.JSON(code, msg)
			}
			if err != nil {
				logger.Error("Error sending response", "error", err)
			}
		}
	}
}
