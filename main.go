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
	"task-manager/handlers"
	"task-manager/models"
	"task-manager/modules"
	"time"
)

var ownerPassword string

func main() {
	// Read configuration from environment variables
	ownerPassword = os.Getenv("OWNER_PASSWORD")
	if ownerPassword == "" {
		ownerPassword = "admin1234"
	}
	ownerEmail := os.Getenv("OWNER_EMAIL")
	if ownerEmail == "" {
		ownerEmail = "admin@gmail.com"
	}

	// Initialize Redis
	if err := modules.InitRedis(); err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}

	// Initialize PostgreSQL
	if err := modules.InitPostgres(); err != nil {
		log.Fatalf("Failed to initialize PostgreSQL: %v", err)
	}

	// Initialize Sync Service
	modules.InitSyncService()

	// Load data from PostgreSQL to Redis on startup (if Redis is empty)
	if err := loadInitialData(); err != nil {
		log.Printf("Warning: Failed to load initial data: %v", err)
	}

	// Ensure owner user exists
	if err := ensureOwnerExists(ownerEmail); err != nil {
		log.Fatalf("Failed to create owner user: %v", err)
	}

	// Start sync service
	modules.Syncer.Start()

	// Set up HTTP server
	server := setupServer()

	// Graceful shutdown setup
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		fmt.Println("🚀 Task Management API Server running on http://localhost:7890")
		printEndpoints()

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
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
		log.Printf("⚠️ Final sync failed: %v", err)
	} else {
		fmt.Println("✅ Final sync completed")
	}

	// Shutdown server gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("⚠️ Server forced to shutdown: %v", err)
	} else {
		fmt.Println("✅ Server shutdown completed")
	}
}

func setupServer() *http.Server {
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
	handler := loggingMiddleware(corsMiddleware(modules.AuthMiddleware(ownerPassword)(mux)))

	return &http.Server{
		Addr:         ":7890",
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

func loadInitialData() error {
	// Check if Redis has any data
	users, err := modules.RedisClient.GetAllUsers()
	if err != nil {
		return err
	}

	// If Redis is empty, load from PostgreSQL
	if len(users) == 0 {
		fmt.Println("🔄 Loading initial data from PostgreSQL to Redis...")
		return modules.Syncer.SyncFromPostgresToRedis()
	}

	fmt.Println("✅ Redis already has data, skipping initial load")
	return nil
}

func ensureOwnerExists(ownerEmail string) error {
	// Check if any owner exists
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
		// Create owner user
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

		// Mark for sync
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

	// Add Redis and PostgreSQL status
	redisStatus := "connected"
	if _, err := modules.RedisClient.GetLastSyncTime(); err != nil {
		redisStatus = "error"
	}

	pgStatus := "connected"
	if err := modules.PostgresClient.Ping(); err != nil {
		pgStatus = "error"
	}

	status["redis_status"] = redisStatus
	status["postgres_status"] = pgStatus

	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"success": true,
		"data":    status,
	}
	json.NewEncoder(w).Encode(response)
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

	// Get stats from PostgreSQL
	pgStats, err := modules.PostgresClient.GetStats()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get stats: %v", err), http.StatusInternalServerError)
		return
	}

	// Get Redis stats
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

	result := map[string]interface{}{
		"success": true,
		"data":    stats,
	}
	json.NewEncoder(w).Encode(result)
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := "healthy"
	code := http.StatusOK

	// Check Redis
	if _, err := modules.RedisClient.GetLastSyncTime(); err != nil {
		status = "unhealthy"
		code = http.StatusServiceUnavailable
	}

	// Check PostgreSQL
	if err := modules.PostgresClient.Ping(); err != nil {
		status = "unhealthy"
		code = http.StatusServiceUnavailable
	}

	// Check sync service
	if !modules.Syncer.IsHealthy() {
		status = "degraded"
		if code == http.StatusOK {
			code = http.StatusPartialContent
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"status": "%s", "timestamp": "%s"}`, status, time.Now().Format(time.RFC3339))
}

// Middleware
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
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
		w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Response writer wrapper for logging
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func printEndpoints() {
	fmt.Println("👥 User Endpoints:")
	fmt.Println("  GET    /users                      - List all users")
	fmt.Println("  POST   /users                      - Create new user")
	fmt.Println("  GET    /users/{id}                 - Get specific user")
	fmt.Println("  PUT    /users/{id}                 - Update user")
	fmt.Println("  DELETE /users/{id}                 - Delete user")
	fmt.Println("  GET    /users/{id}/tasks           - List user's tasks")
	fmt.Println("  POST   /users/{id}/tasks           - Create new task for user")
	fmt.Println("  GET    /users/{id}/tasks/{tid}     - Get specific task")
	fmt.Println("  PUT    /users/{id}/tasks/{tid}     - Update task")
	fmt.Println("  DELETE /users/{id}/tasks/{tid}     - Delete task")
	fmt.Println("  PUT    /users/{id}/tasks/{tid}/done - Mark task as done")
	fmt.Println("  GET    /users/{id}/worktimes       - Get user's work times")
	fmt.Println("  PUT    /users/{id}/worktimes       - Update user's work times")
	fmt.Println("  GET    /users/search?q=query       - Search users")

	fmt.Println("👔 Group Endpoints:")
	fmt.Println("  GET    /groups                     - List all groups")
	fmt.Println("  POST   /groups                     - Create new group")
	fmt.Println("  GET    /groups/{id}                - Get specific group")
	fmt.Println("  PUT    /groups/{id}                - Update group")
	fmt.Println("  DELETE /groups/{id}                - Delete group")
	fmt.Println("  GET    /groups/{id}/users          - List group users")
	fmt.Println("  POST   /groups/{id}/users          - Add user to group")
	fmt.Println("  DELETE /groups/{id}/users/{uid}    - Remove user from group")
	fmt.Println("  GET    /groups/{id}/tasks          - List group tasks")
	fmt.Println("  GET    /groups/{id}/stats          - Get group statistics")

	fmt.Println("📋 Task Endpoints:")
	fmt.Println("  GET    /tasks/search?q=query       - Search tasks across all users")
	fmt.Println("  GET    /tasks/stats                - Get task statistics")
	fmt.Println("  POST   /tasks/batch                - Batch update tasks")
	fmt.Println("  GET    /tasks/filter               - Filter tasks with advanced options")

	fmt.Println("🔧 Admin Endpoints:")
	fmt.Println("  POST   /admin/sync?action=force    - Force sync to PostgreSQL")
	fmt.Println("  POST   /admin/sync?action=restore  - Restore from PostgreSQL")
	fmt.Println("  POST   /admin/sync?action=backup   - Emergency backup")
	fmt.Println("  GET    /admin/status               - Get system status")
	fmt.Println("  GET    /admin/stats                - Get system statistics")

	fmt.Println("🏥 Health Check:")
	fmt.Println("  GET    /health                     - Health check endpoint")

	fmt.Println("\n🔐 Authentication:")
	fmt.Println("  Owner: X-Owner-Password header")
	fmt.Println("  Users: Basic Auth (userID:password or email:password)")

	fmt.Println("\n📊 Data Flow:")
	fmt.Println("  • All operations go through Redis (fast)")
	fmt.Println("  • Background sync to PostgreSQL every 15 minutes")
	fmt.Println("  • Real-time data in Redis, persistent storage in PostgreSQL")
}

// Context key type for auth context
type contextKey string

const authContextKey contextKey = "auth"

// Helper to set auth context in request context
func setAuthContext(r *http.Request, authCtx *modules.AuthContext) *http.Request {
	ctx := context.WithValue(r.Context(), authContextKey, authCtx)
	return r.WithContext(ctx)
}

// Helper to get auth context from request context
func getAuthContext(r *http.Request) *modules.AuthContext {
	if authCtx, ok := r.Context().Value(authContextKey).(*modules.AuthContext); ok {
		return authCtx
	}
	return nil
}
