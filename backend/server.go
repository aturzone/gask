package main

import (
    "log"
    "net/http"

    "github.com/gorilla/mux"
)

func setupServer() *http.Server {
    r := mux.NewRouter()

    // Enable CORS for frontend
    r.Use(func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.Header().Set("Access-Control-Allow-Origin", "*")
            w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
            w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
            
            // Handle preflight requests
            if r.Method == "OPTIONS" {
                w.WriteHeader(http.StatusOK)
                return
            }
            next.ServeHTTP(w, r)
        })
    })

    // API routes with auth
    api := r.PathPrefix("/api").Subrouter()
    
    // Task routes
    api.HandleFunc("/tasks", basicAuth(GetTasks)).Methods("GET")
    api.HandleFunc("/tasks", basicAuth(CreateTask)).Methods("POST")
    api.HandleFunc("/tasks/{id}", basicAuth(UpdateTask)).Methods("PUT")
    api.HandleFunc("/tasks/{id}", basicAuth(DeleteTask)).Methods("DELETE")
    
    // Note routes
    api.HandleFunc("/notes", basicAuth(GetNotes)).Methods("GET")
    api.HandleFunc("/notes", basicAuth(CreateNote)).Methods("POST")
    api.HandleFunc("/notes/{id}", basicAuth(DeleteNote)).Methods("DELETE")
    
    // Box routes
    api.HandleFunc("/boxes", basicAuth(GetBoxes)).Methods("GET")
    api.HandleFunc("/boxes", basicAuth(CreateBox)).Methods("POST")
    api.HandleFunc("/boxes/{id}", basicAuth(DeleteBox)).Methods("DELETE")

    // Health check endpoint (no auth required)
    r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"status":"ok"}`))
    }).Methods("GET")

    srv := &http.Server{
        Handler: r,
        Addr:    ":7887",
    }
    
    log.Println("Server configured with routes:")
    log.Println("  GET    /api/tasks")
    log.Println("  POST   /api/tasks")
    log.Println("  PUT    /api/tasks/{id}")
    log.Println("  DELETE /api/tasks/{id}")
    log.Println("  GET    /api/notes")
    log.Println("  POST   /api/notes")
    log.Println("  DELETE /api/notes/{id}")
    log.Println("  GET    /api/boxes")
    log.Println("  POST   /api/boxes")
    log.Println("  DELETE /api/boxes/{id}")
    log.Println("  GET    /health")
    
    return srv
}
