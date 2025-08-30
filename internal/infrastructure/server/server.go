// internal/infrastructure/server/server.go
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
				s.logger.Error("HTTP request failed", fields...)
			} else {
				s.logger.Info("HTTP request", fields...)
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
	// Root endpoint
	s.echo.GET("/", func(c echo.Context) error {
		return c.HTML(http.StatusOK, `
<!DOCTYPE html>
<html lang="fa" dir="rtl">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Gask - سیستم مدیریت پروژه</title>
    <link rel="icon" type="image/x-icon" href="data:image/x-icon;base64,">
    <style>
        body { 
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; 
            margin: 40px; 
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
        }
        .container { 
            max-width: 900px; 
            margin: 0 auto; 
            background: white; 
            padding: 40px; 
            border-radius: 15px; 
            box-shadow: 0 10px 30px rgba(0,0,0,0.2);
        }
        h1 { 
            color: #2c5aa0; 
            text-align: center; 
            margin-bottom: 10px; 
            font-size: 2.5em;
        }
        .subtitle {
            text-align: center;
            color: #666;
            font-size: 1.2em;
            margin-bottom: 30px;
        }
        .status { 
            background: linear-gradient(135deg, #a8e6cf, #dcedc8); 
            padding: 20px; 
            border-radius: 10px; 
            margin: 20px 0;
            border-left: 5px solid #4caf50;
        }
        .links { 
            display: grid; 
            grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); 
            gap: 20px; 
            margin: 30px 0; 
        }
        .link-card { 
            padding: 25px; 
            border: 2px solid #e0e0e0; 
            border-radius: 12px; 
            text-align: center; 
            transition: all 0.3s ease;
            background: linear-gradient(135deg, #f8f9ff, #ffffff);
        }
        .link-card:hover { 
            border-color: #2c5aa0; 
            transform: translateY(-5px); 
            box-shadow: 0 8px 25px rgba(44,90,160,0.15);
        }
        .link-card a { 
            text-decoration: none; 
            color: #2c5aa0; 
            font-weight: bold; 
            font-size: 1.3em;
            display: block;
            margin-bottom: 10px;
        }
        .link-card p {
            color: #666;
            margin: 0;
            line-height: 1.5;
        }
        .info { 
            background: linear-gradient(135deg, #e3f2fd, #bbdefb); 
            padding: 25px; 
            border-radius: 10px; 
            margin: 20px 0;
            border-left: 5px solid #2196f3;
        }
        .info h3 {
            color: #1976d2;
            margin-top: 0;
        }
        .info ol {
            line-height: 1.8;
        }
        .info a {
            color: #1976d2;
            text-decoration: none;
            font-weight: bold;
        }
        .info a:hover {
            text-decoration: underline;
        }
        .version-info {
            text-align: center;
            margin-top: 30px;
            padding: 15px;
            background: #f5f5f5;
            border-radius: 8px;
            color: #666;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 Gask</h1>
        <div class="subtitle">سیستم مدیریت پروژه و کارها</div>
        
        <div class="status">
            <strong>✅ وضعیت:</strong> سرور با موفقیت در حال اجرا<br>
            <strong>🕐 زمان:</strong> `+time.Now().Format("2006-01-02 15:04:05")+`<br>
            <strong>🌍 محیط:</strong> `+s.config.App.Environment+`<br>
            <strong>📦 نسخه:</strong> `+s.config.App.Version+`<br>
            <strong>🌐 Host:</strong> `+s.config.Server.Host+`:`+fmt.Sprintf("%d", s.config.Server.Port)+`
        </div>
        
        <div class="links">
            <div class="link-card">
                <a href="/docs/simple-swagger.html">📚 مستندات API</a>
                <p>رابط تعاملی برای تست و یادگیری تمام endpoints</p>
            </div>
            <div class="link-card">
                <a href="/swagger.json">📄 Swagger JSON</a>
                <p>فایل مستندات خام برای developers</p>
            </div>
            <div class="link-card">
                <a href="/health">❤️ بررسی سلامت</a>
                <p>وضعیت کلی سرور</p>
            </div>
            <div class="link-card">
                <a href="/health/detailed">🔍 سلامت تفصیلی</a>
                <p>بررسی دقیق تمام سیستم‌ها</p>
            </div>
        </div>
        
        <div class="info">
            <h3>🎯 چگونه شروع کنیم؟</h3>
            <ol>
                <li><strong>مستندات:</strong> <a href="/docs/simple-swagger.html">به صفحه مستندات برو</a></li>
                <li><strong>ثبت‌نام:</strong> از <code>POST /api/v1/auth/register</code> استفاده کن</li>
                <li><strong>ورود:</strong> با <code>POST /api/v1/auth/login</code> توکن دریافت کن</li>
                <li><strong>پروژه:</strong> با <code>POST /api/v1/projects</code> پروژه جدید بساز</li>
                <li><strong>کار:</strong> با <code>POST /api/v1/tasks</code> کار جدید اضافه کن</li>
                <li><strong>زمان‌سنجی:</strong> با <code>/api/v1/time/*</code> زمان کار رو رهگیری کن</li>
            </ol>
        </div>

        <div class="version-info">
            <strong>Gask API Server</strong> - Enterprise Task & Project Management System<br>
            Powered by Go + Echo + PostgreSQL
        </div>
    </div>
</body>
</html>
        `)
	})

	// Health check routes
	s.echo.GET("/health", s.healthCheck)
	s.echo.GET("/health/detailed", s.detailedHealthCheck)
	s.echo.GET("/ready", s.readinessCheck)

	// Favicon handler
	s.echo.GET("/favicon.ico", func(c echo.Context) error {
		return c.Blob(http.StatusOK, "image/x-icon", []byte{})
	})

	// Swagger documentation routes
	s.echo.Static("/docs", "docs")
	s.echo.GET("/swagger.json", func(c echo.Context) error {
		return c.File("docs/swagger.json")
	})
	
	// Swagger redirects
	s.echo.GET("/swagger", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/docs/simple-swagger.html")
	})
	s.echo.GET("/swagger/", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/docs/simple-swagger.html")
	})
	s.echo.GET("/swagger/index.html", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/docs/simple-swagger.html")
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
	userGroup.GET("/me", userHandler.GetMe)
	userGroup.PUT("/me", userHandler.UpdateMe)
	userGroup.GET("", userHandler.ListUsers, s.requireRole(entities.UserRoleAdmin))
	userGroup.GET("/:id", userHandler.GetUser, s.requireRole(entities.UserRoleAdmin, entities.UserRoleManager))
	userGroup.PUT("/:id", userHandler.UpdateUser, s.requireRole(entities.UserRoleAdmin))
	userGroup.DELETE("/:id", userHandler.DeleteUser, s.requireRole(entities.UserRoleAdmin))
	userGroup.POST("", userHandler.CreateUser, s.requireRole(entities.UserRoleAdmin))

	// Project routes (authenticated)
	projectGroup := v1.Group("/projects", s.authMiddleware(authService))
	projectGroup.POST("", projectHandler.Create)
	projectGroup.GET("", projectHandler.List)
	projectGroup.GET("/:id", projectHandler.GetByID)
	projectGroup.PUT("/:id", projectHandler.Update)
	projectGroup.DELETE("/:id", projectHandler.Delete, s.requireRole(entities.UserRoleAdmin, entities.UserRoleManager))
	projectGroup.GET("/:id/tasks", projectHandler.GetProjectTasks)
	
	// Project member routes
	projectGroup.POST("/:id/members", projectHandler.AddMember, s.requireRole(entities.UserRoleAdmin, entities.UserRoleManager))
	projectGroup.DELETE("/:id/members/:user_id", projectHandler.RemoveMember, s.requireRole(entities.UserRoleAdmin, entities.UserRoleManager))

	// Task routes (authenticated)
	taskGroup := v1.Group("/tasks", s.authMiddleware(authService))
	taskGroup.POST("", taskHandler.Create)
	taskGroup.GET("", taskHandler.List)
	taskGroup.GET("/:id", taskHandler.GetByID)
	taskGroup.PUT("/:id", taskHandler.Update)
	taskGroup.DELETE("/:id", taskHandler.Delete)
	taskGroup.PATCH("/:id/status", taskHandler.UpdateStatus)
	taskGroup.POST("/:id/assign", taskHandler.AssignUser)
	taskGroup.DELETE("/:id/assign", taskHandler.UnassignUser)
	taskGroup.GET("/deadlines", taskHandler.GetDeadlines)

	// Time tracking routes (authenticated)
	timeGroup := v1.Group("/time", s.authMiddleware(authService))
	timeGroup.POST("", timeHandler.CreateTimeEntry)
	timeGroup.POST("/start", timeHandler.StartTime)
	timeGroup.POST("/stop", timeHandler.StopTime)
	timeGroup.GET("/active", timeHandler.GetActiveTimeEntry)
	timeGroup.GET("/entries", timeHandler.ListEntries)
	timeGroup.GET("/entries/:id", timeHandler.GetEntry)
	timeGroup.PUT("/entries/:id", timeHandler.UpdateEntry)
	timeGroup.DELETE("/entries/:id", timeHandler.DeleteEntry)
	timeGroup.POST("/reports", timeHandler.GetTimeReport)
}

// setupMetrics configures Prometheus metrics
func (s *Server) setupMetrics() {
	// Create metrics endpoint
	metricsHandler := echo.WrapHandler(promhttp.Handler())
	s.echo.GET("/metrics", metricsHandler)

	// Add custom metrics middleware
	s.echo.Use(s.metricsMiddleware())
}

// metricsMiddleware adds Prometheus metrics
func (s *Server) metricsMiddleware() echo.MiddlewareFunc {
	requestCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)
	
	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	prometheus.MustRegister(requestCounter, requestDuration)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			
			err := next(c)
			
			duration := time.Since(start)
			status := c.Response().Status
			
			requestCounter.WithLabelValues(
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
		} else {
			msg = err.Error()
		}

		if code == http.StatusInternalServerError {
			logger.Error("Internal server error", "error", err.Error(), "path", c.Request().URL.Path)
		}

		// Send response
		if !c.Response().Committed {
			if c.Request().Method == echo.HEAD {
				err = c.NoContent(code)
			} else {
				err = c.JSON(code, map[string]interface{}{
					"error": msg,
					"code":  code,
				})
			}
			if err != nil {
				logger.Error("Failed to send error response", "error", err.Error())
			}
		}
	}
}
