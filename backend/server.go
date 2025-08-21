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
            w.Header().Set("Access-Control-Allow-Origin", "*") // Adjust for production
            w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
            w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
            if r.Method == "OPTIONS" {
                return
            }
            next.ServeHTTP(w, r)
        })
    })

    // API routes with auth
    api := r.PathPrefix("/api").Subrouter()
    api.HandleFunc("/tasks", basicAuth(GetTasks)).Methods("GET")
    api.HandleFunc("/tasks", basicAuth(CreateTask)).Methods("POST")
    // Add more: /notes, /boxes, /tasks/{id}, etc.

    srv := &http.Server{
        Handler: r,
        Addr:    ":7887",
    }
    return srv
}
