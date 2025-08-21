package main

import (
    "encoding/json"
    "net/http"
    "time"

    "github.com/gorilla/mux"
    "github.com/google/uuid"
)

// Middleware for Basic Auth.
func basicAuth(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        user, pass, ok := r.BasicAuth()
        if !ok || user != "admin" || pass != "securepass" {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        next(w, r)
    }
}

// GetTasks handler.
func GetTasks(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(store.Tasks)
}

// CreateTask handler (similar for others).
func CreateTask(w http.ResponseWriter, r *http.Request) {
    var task Task
    if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    task.ID = uuid.New().String()
    if task.DueDate.IsZero() {
        task.DueDate = time.Now() // Default
    }
    store.Tasks = append(store.Tasks, task)
    SaveData()
    json.NewEncoder(w).Encode(task)
}

// Similar handlers for Notes, Boxes, Update, Delete...
// For Boxes: CreateBox, AddTaskToBox (find box by ID, append task)
// For date sync: In Create, if Shamsi sent, convert using FromShamsi
