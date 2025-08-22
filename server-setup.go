package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	echoSwagger "github.com/swaggo/echo-swagger"
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
		AllowOrigins:     s.config.Security.CORSAllowedOrigins,
		AllowMethods:     s.config.Security.CORSAllowedMethods,
		AllowHeaders:     s.config.Security.CORSAllowedHeaders,
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Security headers middleware
	s.echo.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "DENY",
		HSTSMaxAge:            31536000,
		HSTSExcludeSubdomains: false,
		ContentSecurityPolicy: "default-src 'self'",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
	}))

	// Rate limiting middleware
	s.echo.Use(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{
				Rate:      rate.Limit(float64(s.config.Security.RateLimitRequests) / s.config.Security.RateLimitWindow.Minutes()),
				Burst:     s.config.Security.RateLimitRequests,
				ExpiresIn: s.config.Security.RateLimitWindow,
			},
		),
		IdentifierExtractor: func(c echo.Context) (string, error) {
			return c.RealIP(), nil
		},
		ErrorHandler: func(context echo.Context, err error) error {
			return echo.NewHTTPError(http.StatusTooManyRequests, "Rate limit exceeded")
		},
	}))

	// Request ID middleware
	s.echo.Use(middleware.RequestIDWithConfig(middleware.RequestIDConfig{
		Generator: func() string {
			return uuid.New().String()
		},
	}))

	// Timeout middleware
	s.echo.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: 30 * time.Second,
	}))

	// Gzip compression
	s.echo.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: 5,
	}))
}

// setupRoutes configures API routes
func (s *Server) setupRoutes(
	authHandler *httpHandlers.AuthHandler,
	userHandler *httpHandlers.UserHandler,
	taskHandler *httpHandlers.TaskHandler,
	projectHandler *httpHandlers.ProjectHandler,
	timeHandler *httpHandlers.TimeHandler,
	authService *services.AuthService,
) {
	// Health check routes
	s.echo.GET("/health", s.healthCheck)
	s.echo.GET("/health/detailed", s.detailedHealthCheck)
	s.echo.GET("/ready", s.readinessCheck)

	// API documentation
	s.echo.GET("/docs/*", echoSwagger.WrapHandler)

	// API routes
	api := s.echo.Group("/api/v1")

	// Authentication routes (public)
	auth := api.Group("/auth")
	auth.POST("/login", authHandler.Login)
	auth.POST("/refresh", authHandler.RefreshToken)

	// Protected routes
	protected := api.Group("")
	protected.Use(s.authMiddleware(authService))

	// Auth protected routes
	authProtected := protected.Group("/auth")
	authProtected.POST("/logout", authHandler.Logout)

	// User routes
	users := protected.Group("/users")
	users.POST("", userHandler.CreateUser, s.requireRole("admin", "project_manager"))
	users.GET("", userHandler.ListUsers)
	users.GET("/me", userHandler.GetCurrentUser)
	users.PUT("/me", userHandler.UpdateCurrentUser)
	users.GET("/:id", userHandler.GetUser)

	// Project routes
	projects := protected.Group("/projects")
	projects.POST("", projectHandler.CreateProject, s.requireRole("admin", "project_manager"))
	projects.GET("", projectHandler.ListProjects)
	projects.GET("/me", projectHandler.GetMyProjects)
	projects.GET("/stats", projectHandler.GetProjectStats)
	projects.GET("/:id", projectHandler.GetProject)
	projects.PUT("/:id", projectHandler.UpdateProject, s.requireRole("admin", "project_manager"))
	projects.DELETE("/:id", projectHandler.DeleteProject, s.requireRole("admin", "project_manager"))
	projects.GET("/:id/tasks", projectHandler.GetProjectTasks)
	projects.POST("/:id/members", projectHandler.AddProjectMember, s.requireRole("admin", "project_manager"))
	projects.DELETE("/:id/members/:user_id", projectHandler.RemoveProjectMember, s.requireRole("admin", "project_manager"))
	projects.POST("/:id/activate", projectHandler.ActivateProject, s.requireRole("admin", "project_manager"))
	projects.POST("/:id/complete", projectHandler.CompleteProject, s.requireRole("admin", "project_manager"))

	// Task routes
	tasks := protected.Group("/tasks")
	tasks.POST("", taskHandler.CreateTask)
	tasks.GET("", taskHandler.ListTasks)
	tasks.GET("/deadlines", taskHandler.GetDeadlines)
	tasks.GET("/:id", taskHandler.GetTask)
	tasks.PUT("/:id", taskHandler.UpdateTask)
	tasks.POST("/:id/assign", taskHandler.AssignTask, s.requireRole("admin", "project_manager", "team_lead"))
	tasks.POST("/:id/start", taskHandler.StartTask)
	tasks.POST("/:id/complete", taskHandler.CompleteTask)

	// Time tracking routes
	timeEntries := protected.Group("/time-entries")
	timeEntries.POST("", timeHandler.CreateTimeEntry)
	timeEntries.GET("", timeHandler.ListTimeEntries)
	timeEntries.GET("/active", timeHandler.GetActiveTimeEntry)
	timeEntries.GET("/report", timeHandler.GetTimeReport)
	timeEntries.POST("/start", timeHandler.StartTimeTracking)
	timeEntries.POST("/stop", timeHandler.StopTimeTracking)
	timeEntries.GET("/:id", timeHandler.GetTimeEntry)
	timeEntries.PUT("/:id", timeHandler.UpdateTimeEntry)
	timeEntries.DELETE("/:id", timeHandler.DeleteTimeEntry)
}

// setupMetrics configures Prometheus metrics
func (s *Server) setupMetrics() {
	// Create metrics registry
	registry := prometheus.NewRegistry()

	// Register default metrics
	registry.MustRegister(prometheus.NewGoCollector())
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	// Custom metrics
	requestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	activeConnections := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_active_connections",
			Help: "Number of active HTTP connections",
		},
	)

	registry.MustRegister(requestsTotal, requestDuration, activeConnections)

	// Metrics middleware
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
		} else {
			msg = err.Error()
		}

		// Log error
		if code >= 500 {
			logger.Error("HTTP error", 
				"error", err,
				"method", c.Request().Method,
				"uri", c.Request().RequestURI,
				"status", code,
				"ip", c.RealIP(),
			)
		} else if code >= 400 {
			logger.Warn("HTTP client error",
				"error", err,
				"method", c.Request().Method,
				"uri", c.Request().RequestURI,
				"status", code,
				"ip", c.RealIP(),
			)
		}

		// Send error response
		if !c.Response().Committed {
			if c.Request().Method == http.MethodHead {
				err = c.NoContent(code)
			} else {
				response := map[string]interface{}{
					"error": msg,
					"code":  code,
				}

				// Add request ID for debugging
				if reqID := c.Response().Header().Get(echo.HeaderXRequestID); reqID != "" {
					response["request_id"] = reqID
				}

				err = c.JSON(code, response)
			}
			if err != nil {
				logger.Error("Failed to send error response", "error", err)
			}
		}
	}
}