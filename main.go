package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"task-manager/config"
	"task-manager/handlers"
	"task-manager/models"
	"task-manager/modules"
	"time"
)

func main() {
	// ASCII Art Banner
	printBanner()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}
	cfg.Print()

	// Initialize Redis
	if err := modules.InitRedis(cfg); err != nil {
		log.Fatalf("❌ Failed to initialize Redis: %v", err)
	}

	// Initialize PostgreSQL
	if err := modules.InitPostgres(cfg); err != nil {
		log.Fatalf("❌ Failed to initialize PostgreSQL: %v", err)
	}

	// Initialize Sync Service
	modules.InitSyncService()

	// Load data from PostgreSQL to Redis on startup
	if err := loadInitialData(); err != nil {
		log.Printf("⚠️  Warning: Failed to load initial data: %v", err)
	}

	// Ensure owner user exists
	if err := ensureOwnerExists(cfg.OwnerEmail, cfg.OwnerPassword); err != nil {
		log.Fatalf("❌ Failed to create owner user: %v", err)
	}

	// Start sync service
	modules.Syncer.Start()

	// Set up HTTP server
	server := setupServer(cfg)

	// Graceful shutdown setup
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		fmt.Printf("\n🚀 GASK API Server running at http://%s\n\n", cfg.GetAPIAddr())
		printEndpoints()

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server failed to start: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-stop
	fmt.Println("\n🔄 Shutting down server...")

	// Stop sync service
	modules.Syncer.Stop()

	// Force final sync before shutdown
	fmt.Println("📤 Performing final sync...")
	if err := modules.Syncer.ForceSyncNow(); err != nil {
		log.Printf("⚠️  Final sync failed: %v", err)
	} else {
		fmt.Println("✅ Final sync completed")
	}

	// Close database connections
	if modules.RedisClient != nil {
		modules.RedisClient.Close()
	}
	if modules.PostgresClient != nil {
		modules.PostgresClient.Close()
	}

	// Shutdown server gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("⚠️  Server forced to shutdown: %v", err)
	} else {
		fmt.Println("✅ Server shutdown completed")
	}
}

func setupServer(cfg *config.Config) *http.Server {
	mux := http.NewServeMux()

	// User routes
	mux.HandleFunc("/users", handlers.UsersHandler)
	mux.HandleFunc("/users/", handlers.UserHandler)
	mux.HandleFunc("/users/search", handlers.SearchUsersHandler)

	// Group routes
	mux.HandleFunc("/groups", handlers.GroupsHandler)
	mux.HandleFunc("/groups/", handlers.GroupHandler)

	// Global task routes
	mux.HandleFunc("/tasks/search", handlers.SearchTasksHandler)
	mux.HandleFunc("/tasks/stats", handlers.GetTaskStatsHandler)
	mux.HandleFunc("/tasks/batch", handlers.BatchUpdateTasksHandler)
	mux.HandleFunc("/tasks/filter", handlers.GetTasksWithFiltersHandler)

	// Admin/monitoring routes
	mux.HandleFunc("/admin/sync", adminSyncHandler)
	mux.HandleFunc("/admin/status", adminStatusHandler)
	mux.HandleFunc("/admin/stats", adminStatsHandler)
	mux.HandleFunc("/health", healthCheckHandler)

	// Apply middleware: CORS -> Auth -> Logging
	handler := loggingMiddleware(corsMiddleware(modules.AuthMiddleware(cfg.OwnerPassword)(mux)))

	return &http.Server{
		Addr:         cfg.GetAPIAddr(),
		Handler:      handler,
		ReadTimeout:  cfg.APITimeout,
		WriteTimeout: cfg.APITimeout,
		IdleTimeout:  60 * time.Second,
	}
}

func loadInitialData() error {
	users, err := modules.RedisClient.GetAllUsers()
	if err != nil {
		return err
	}

	if len(users) == 0 {
		fmt.Println("🔄 Loading initial data from PostgreSQL to Redis...")
		return modules.Syncer.SyncFromPostgresToRedis()
	}

	fmt.Println("✅ Redis already has data, skipping initial load")
	return nil
}

func ensureOwnerExists(ownerEmail, ownerPassword string) error {
	users, err := modules.RedisClient.GetAllUsers()
	if err != nil {
		return err
	}

	hasOwner := false
	for _, user := range users {
		if user.Role == "owner" {
			hasOwner = true
			break
		}
	}

	if !hasOwner {
		ownerID, err := modules.RedisClient.GetNextUserID()
		if err != nil {
			return err
		}

		owner := &models.User{
			ID:        ownerID,
			FullName:  "System Owner",
			Role:      "owner",
			GroupIDs:  models.IntSlice{},
			Email:     ownerEmail,
			Password:  ownerPassword,
			WorkTimes: make(models.WorkTimes),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := modules.RedisClient.SaveUser(owner); err != nil {
			return err
		}

		modules.RedisClient.MarkDirty("users")

		fmt.Printf("✅ Created initial owner user with email: %s\n", ownerEmail)
	}

	return nil
}

// Admin handlers
func adminSyncHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authCtx := modules.GetAuthContext(r)
	if !authCtx.IsOwner {
		http.Error(w, "Only owner can trigger manual sync", http.StatusForbidden)
		return
	}

	action := r.URL.Query().Get("action")

	var err error
	switch action {
	case "force":
		err = modules.Syncer.ForceSyncNow()
	case "restore":
		err = modules.Syncer.RestoreFromPostgreSQL()
	case "backup":
		err = modules.Syncer.EmergencyBackup()
	default:
		err = modules.Syncer.ForceSyncNow()
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Sync failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success": true, "message": "Sync completed successfully", "action": "%s"}`, action)
}

func adminStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authCtx := modules.GetAuthContext(r)
	if !authCtx.IsOwner {
		http.Error(w, "Only owner can view admin status", http.StatusForbidden)
		return
	}

	status := modules.Syncer.GetSyncStatus()

	// Add connection info
	if modules.RedisClient != nil {
		status["redis"] = modules.RedisClient.GetConnectionInfo()
	}
	if modules.PostgresClient != nil {
		status["postgres"] = modules.PostgresClient.GetConnectionInfo()
	}

	// Add configuration info
	cfg := config.AppConfig
	status["configuration"] = map[string]interface{}{
		"api_port":      cfg.APIPort,
		"sync_interval": cfg.SyncInterval.String(),
		"environment":   cfg.Environment,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    status,
	})
}

func adminStatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authCtx := modules.GetAuthContext(r)
	if !authCtx.IsOwner {
		http.Error(w, "Only owner can view admin stats", http.StatusForbidden)
		return
	}

	pgStats, err := modules.PostgresClient.GetStats()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get stats: %v", err), http.StatusInternalServerError)
		return
	}

	users, _ := modules.RedisClient.GetAllUsers()
	groups, _ := modules.RedisClient.GetAllGroups()

	var totalTasks int
	for _, user := range users {
		tasks, _ := modules.RedisClient.GetUserTasks(user.ID)
		totalTasks += len(tasks)
	}

	redisStats := map[string]interface{}{
		"users":  len(users),
		"groups": len(groups),
		"tasks":  totalTasks,
	}

	stats := map[string]interface{}{
		"postgresql": pgStats,
		"redis":      redisStats,
		"sync":       modules.Syncer.GetSyncStatus(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    stats,
	})
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := "healthy"
	code := http.StatusOK
	issues := []string{}

	// Check Redis
	if modules.RedisClient != nil {
		if err := modules.RedisClient.Ping(); err != nil {
			status = "unhealthy"
			code = http.StatusServiceUnavailable
			issues = append(issues, "Redis connection failed")
		}
	}

	// Check PostgreSQL
	if modules.PostgresClient != nil {
		if err := modules.PostgresClient.Ping(); err != nil {
			status = "unhealthy"
			code = http.StatusServiceUnavailable
			issues = append(issues, "PostgreSQL connection failed")
		}
	}

	// Check sync service
	if !modules.Syncer.IsHealthy() {
		if status != "unhealthy" {
			status = "degraded"
			code = http.StatusPartialContent
		}
		issues = append(issues, "Sync service degraded")
	}

	response := map[string]interface{}{
		"status":    status,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	if len(issues) > 0 {
		response["issues"] = issues
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(response)
}

// Middleware
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)
		duration := time.Since(start)
		log.Printf("%s %s %d %v", r.Method, r.URL.Path, rw.statusCode, duration)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Owner-Password")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func printBanner() {
	banner := `
╔═══════════════════════════════════════════════════════════╗
║                                                           ║
║    ██████╗  █████╗ ███████╗██╗  ██╗                     ║
║   ██╔════╝ ██╔══██╗██╔════╝██║ ██╔╝                     ║
║   ██║  ███╗███████║███████╗█████╔╝                      ║
║   ██║   ██║██╔══██║╚════██║██╔═██╗                      ║
║   ╚██████╔╝██║  ██║███████║██║  ██╗                     ║
║    ╚═════╝ ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝                     ║
║                                                           ║
║   Go-based Advanced taSK management system                ║
║   Version 2.0 - Production Ready                         ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝
`
	fmt.Println(banner)
}

func printEndpoints() {
	fmt.Println("📚 Available Endpoints:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("👥 Users:      GET/POST /users")
	fmt.Println("👔 Groups:     GET/POST /groups")
	fmt.Println("📋 Tasks:      GET/POST /users/{id}/tasks")
	fmt.Println("🔍 Search:     GET /tasks/search?q=...")
	fmt.Println("📊 Stats:      GET /tasks/stats")
	fmt.Println("🔧 Admin:      POST /admin/sync")
	fmt.Println("🏥 Health:     GET /health")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📖 Full API documentation in README.md")
	fmt.Println()
}
