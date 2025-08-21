package main

import "time"

// Task represents a task with dates in ISO (Miladi) format; convert to Shamsi on demand.
type Task struct {
    ID          string    `json:"id"`
    Title       string    `json:"title"`
    Description string    `json:"description"`
    DueDate     time.Time `json:"due_date"` // Stored as Miladi; frontend can convert
    BoxID       string    `json:"box_id"`   // Links to a box
}

// Note similar to Task but without due date/box.
type Note struct {
    ID          string    `json:"id"`
    Title       string    `json:"title"`
    Content     string    `json:"content"`
    CreatedDate time.Time `json:"created_date"`
}

// Box for organizing tasks (like columns in a board).
type Box struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Tasks []Task `json:"tasks"` // Embedded for simplicity
}

// DataStore holds all data.
type DataStore struct {
    Tasks []Task `json:"tasks"`
    Notes []Note `json:"notes"`
    Boxes []Box  `json:"boxes"`
}
