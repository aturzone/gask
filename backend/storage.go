package main

import (
    "encoding/json"
    "os"
    "sync"
)

var (
    dataFile = "data.json"
    store    DataStore
    mu       sync.Mutex // For thread-safety
)

// LoadData loads from file.
func LoadData() error {
    mu.Lock()
    defer mu.Unlock()
    file, err := os.ReadFile(dataFile)
    if err != nil {
        if os.IsNotExist(err) {
            store = DataStore{} // Initialize empty
            return SaveData()
        }
        return err
    }
    return json.Unmarshal(file, &store)
}

// SaveData saves to file atomically.
func SaveData() error {
    mu.Lock()
    defer mu.Unlock()
    data, err := json.MarshalIndent(store, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(dataFile, data, 0644)
}
