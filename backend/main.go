package main

import "log"

func main() {
    if err := LoadData(); err != nil {
        log.Fatalf("Failed to load data: %v", err)
    }
    srv := setupServer()
    log.Println("Server starting on :7887")
    if err := srv.ListenAndServe(); err != nil {
        log.Fatalf("Server failed: %v", err)
    }
}
