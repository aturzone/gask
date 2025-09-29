package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"task-manager/models"
	"task-manager/modules"
	"time"
)

// UsersHandler handles /users endpoint
func UsersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		getAllUsers(w, r)
	case "POST":
		createUser(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// UserHandler handles /users/{id} and sub-paths
func UserHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/users/")
	if path == "" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	parts := strings.Split(path, "/")
	if len(parts) < 1 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	idStr := parts[0]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Check if user exists
	_, err = modules.RedisClient.GetUser(id)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if len(parts) == 1 {
		// /users/{id}
		switch r.Method {
		case "GET":
			getUser(w, r, id)
		case "PUT":
			updateUser(w, r, id)
		case "DELETE":
			deleteUser(w, r, id)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	subPath := parts[1]
	if subPath == "tasks" {
		handleUserTasks(w, r, id, parts[2:])
	} else if subPath == "worktimes" && len(parts) == 2 {
		handleUserWorkTimes(w, r, id)
	} else {
		http.Error(w, "Invalid sub-path", http.StatusBadRequest)
	}
}

// SearchUsersHandler handles /users/search
func SearchUsersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		respondWithError(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	authCtx := modules.GetAuthContext(r)

	users, err := modules.RedisClient.SearchUsers(query)
	if err != nil {
		respondWithError(w, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Filter results based on permissions
	filteredUsers := modules.FilterUsersByPermissions(authCtx, users)

	respondWithSuccess(w, map[string]interface{}{
		"query":   query,
		"results": filteredUsers,
		"count":   len(filteredUsers),
	})
}

func getAllUsers(w http.ResponseWriter, r *http.Request) {
	authCtx := modules.GetAuthContext(r)

	users, err := modules.RedisClient.GetAllUsers()
	if err != nil {
		respondWithError(w, fmt.Sprintf("Failed to get users: %v", err), http.StatusInternalServerError)
		return
	}

	// Filter results based on permissions
	filteredUsers := modules.FilterUsersByPermissions(authCtx, users)

	respondWithSuccess(w, map[string]interface{}{
		"users": filteredUsers,
		"count": len(filteredUsers),
	})
}

func createUser(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.FullName == "" || req.Email == "" || req.Password == "" {
		respondWithError(w, "Full name, email and password are required", http.StatusBadRequest)
		return
	}

	authCtx := modules.GetAuthContext(r)

	// Check if user can create with specified role and groups
	if !modules.CanCreateUser(authCtx, req.Role, req.GroupIDs) {
		respondWithError(w, "Insufficient permissions to create user with specified role/groups", http.StatusForbidden)
		return
	}

	// Set default role if not specified
	if req.Role == "" {
		req.Role = "user"
	}

	// Validate role
	if req.Role != "user" && req.Role != "group_admin" && req.Role != "owner" {
		respondWithError(w, "Invalid role. Must be 'user', 'group_admin', or 'owner'", http.StatusBadRequest)
		return
	}

	// Check if email already exists
	existingUser, _ := modules.RedisClient.GetUserByEmail(req.Email)
	if existingUser != nil {
		respondWithError(w, "User with this email already exists", http.StatusConflict)
		return
	}

	// Validate groups exist
	for _, groupID := range req.GroupIDs {
		_, err := modules.RedisClient.GetGroup(groupID)
		if err != nil {
			respondWithError(w, fmt.Sprintf("Group %d not found", groupID), http.StatusBadRequest)
			return
		}
	}

	// Get next user ID
	userID, err := modules.RedisClient.GetNextUserID()
	if err != nil {
		respondWithError(w, "Failed to generate user ID", http.StatusInternalServerError)
		return
	}

	// Create user
	user := &models.User{
		ID:        userID,
		FullName:  req.FullName,
		Role:      req.Role,
		GroupIDs:  models.IntSlice(req.GroupIDs),
		Number:    req.Number,
		Email:     req.Email,
		Password:  req.Password,
		WorkTimes: models.WorkTimes(req.WorkTimes),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Initialize WorkTimes if nil
	if user.WorkTimes == nil {
		user.WorkTimes = make(models.WorkTimes)
	}

	// Save user
	if err := modules.RedisClient.SaveUser(user); err != nil {
		respondWithError(w, "Failed to save user", http.StatusInternalServerError)
		return
	}

	// Mark data as dirty for sync
	modules.RedisClient.MarkDirty("users")

	// Remove password from response
	user.Password = ""

	respondWithSuccess(w, map[string]interface{}{
		"message": "User created successfully",
		"user":    user,
	}, http.StatusCreated)
}

func getUser(w http.ResponseWriter, r *http.Request, id int) {
	user, err := modules.RedisClient.GetUser(id)
	if err != nil {
		respondWithError(w, "User not found", http.StatusNotFound)
		return
	}

	// Remove password from response
	user.Password = ""

	respondWithSuccess(w, user)
}

func updateUser(w http.ResponseWriter, r *http.Request, id int) {
	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	user, err := modules.RedisClient.GetUser(id)
	if err != nil {
		respondWithError(w, "User not found", http.StatusNotFound)
		return
	}

	authCtx := modules.GetAuthContext(r)

	// Check if trying to change role or groups - need special permissions
	if req.Role != "" && req.Role != user.Role {
		if !authCtx.IsOwner {
			respondWithError(w, "Only owner can change user roles", http.StatusForbidden)
			return
		}
	}

	// Update fields
	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if req.Role != "" {
		if req.Role != "user" && req.Role != "group_admin" && req.Role != "owner" {
			respondWithError(w, "Invalid role", http.StatusBadRequest)
			return
		}
		user.Role = req.Role
	}
	if req.GroupIDs != nil {
		// Validate groups exist
		for _, groupID := range req.GroupIDs {
			_, err := modules.RedisClient.GetGroup(groupID)
			if err != nil {
				respondWithError(w, fmt.Sprintf("Group %d not found", groupID), http.StatusBadRequest)
				return
			}
		}
		user.GroupIDs = models.IntSlice(req.GroupIDs)
	}
	if req.Number != "" {
		user.Number = req.Number
	}
	if req.Email != "" {
		// Check if email already exists (for other users)
		existingUser, _ := modules.RedisClient.GetUserByEmail(req.Email)
		if existingUser != nil && existingUser.ID != user.ID {
			respondWithError(w, "User with this email already exists", http.StatusConflict)
			return
		}
		user.Email = req.Email
	}
	if req.Password != "" {
		user.Password = req.Password
	}
	if req.WorkTimes != nil {
		user.WorkTimes = models.WorkTimes(req.WorkTimes)
	}

	user.UpdatedAt = time.Now()

	// Save user
	if err := modules.RedisClient.SaveUser(user); err != nil {
		respondWithError(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	// Mark data as dirty for sync
	modules.RedisClient.MarkDirty("users")

	// Remove password from response
	user.Password = ""

	respondWithSuccess(w, map[string]interface{}{
		"message": "User updated successfully",
		"user":    user,
	})
}

func deleteUser(w http.ResponseWriter, r *http.Request, id int) {
	authCtx := modules.GetAuthContext(r)

	// Only owner can delete users
	if !authCtx.IsOwner {
		respondWithError(w, "Only owner can delete users", http.StatusForbidden)
		return
	}

	user, err := modules.RedisClient.GetUser(id)
	if err != nil {
		respondWithError(w, "User not found", http.StatusNotFound)
		return
	}

	// Delete all user tasks first
	tasks, _ := modules.RedisClient.GetUserTasks(id)
	for _, task := range tasks {
		modules.RedisClient.DeleteTask(task.ID)
	}

	// Delete user
	if err := modules.RedisClient.DeleteUser(id); err != nil {
		respondWithError(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	// Mark data as dirty for sync
	modules.RedisClient.MarkDirty("users")
	modules.RedisClient.MarkDirty("tasks")

	// Remove password from response
	user.Password = ""

	respondWithSuccess(w, map[string]interface{}{
		"message": "User deleted successfully",
		"user":    user,
	})
}

func handleUserTasks(w http.ResponseWriter, r *http.Request, userID int, remainingParts []string) {
	if len(remainingParts) == 0 {
		// /users/{id}/tasks
		switch r.Method {
		case "GET":
			getUserTasks(w, r, userID)
		case "POST":
			createUserTask(w, r, userID)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	if len(remainingParts) >= 1 {
		taskIDStr := remainingParts[0]
		taskID, err := strconv.Atoi(taskIDStr)
		if err != nil {
			http.Error(w, "Invalid task ID", http.StatusBadRequest)
			return
		}

		if len(remainingParts) == 1 {
			// /users/{id}/tasks/{tid}
			switch r.Method {
			case "GET":
				getUserTask(w, r, userID, taskID)
			case "PUT":
				updateUserTask(w, r, userID, taskID)
			case "DELETE":
				deleteUserTask(w, r, userID, taskID)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		} else if len(remainingParts) == 2 && remainingParts[1] == "done" {
			// /users/{id}/tasks/{tid}/done
			if r.Method == "PUT" {
				markUserTaskDone(w, r, userID, taskID)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		} else {
			http.Error(w, "Invalid task sub-path", http.StatusBadRequest)
		}
	}
}

func handleUserWorkTimes(w http.ResponseWriter, r *http.Request, userID int) {
	switch r.Method {
	case "GET":
		getUserWorkTimes(w, r, userID)
	case "PUT":
		updateUserWorkTimes(w, r, userID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func getUserTasks(w http.ResponseWriter, r *http.Request, userID int) {
	tasks, err := modules.RedisClient.GetUserTasks(userID)
	if err != nil {
		respondWithError(w, fmt.Sprintf("Failed to get tasks: %v", err), http.StatusInternalServerError)
		return
	}

	respondWithSuccess(w, map[string]interface{}{
		"user_id": userID,
		"tasks":   tasks,
		"count":   len(tasks),
	})
}

func createUserTask(w http.ResponseWriter, r *http.Request, userID int) {
	var req models.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		respondWithError(w, "Title is required", http.StatusBadRequest)
		return
	}

	if req.GroupID == 0 {
		respondWithError(w, "Group ID is required", http.StatusBadRequest)
		return
	}

	// Validate group exists
	_, err := modules.RedisClient.GetGroup(req.GroupID)
	if err != nil {
		respondWithError(w, "Group not found", http.StatusBadRequest)
		return
	}

	// Check if user belongs to the group
	user, err := modules.RedisClient.GetUser(userID)
	if err != nil {
		respondWithError(w, "User not found", http.StatusNotFound)
		return
	}

	belongsToGroup := false
	for _, groupID := range user.GroupIDs {
		if groupID == req.GroupID {
			belongsToGroup = true
			break
		}
	}

	authCtx := modules.GetAuthContext(r)
	if !belongsToGroup && !authCtx.IsOwner {
		respondWithError(w, "User does not belong to specified group", http.StatusForbidden)
		return
	}

	// Get next task ID
	taskID, err := modules.RedisClient.GetNextTaskID()
	if err != nil {
		respondWithError(w, "Failed to generate task ID", http.StatusInternalServerError)
		return
	}

	task := &models.Task{
		ID:          taskID,
		Title:       req.Title,
		Priority:    req.Priority,
		Deadline:    req.Deadline,
		Information: req.Information,
		Status:      false,
		UserID:      userID,
		GroupID:     req.GroupID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := modules.RedisClient.SaveTask(task); err != nil {
		respondWithError(w, "Failed to save task", http.StatusInternalServerError)
		return
	}

	// Mark data as dirty for sync
	modules.RedisClient.MarkDirty("tasks")

	respondWithSuccess(w, map[string]interface{}{
		"message": "Task created successfully",
		"task":    task,
	}, http.StatusCreated)
}

func getUserTask(w http.ResponseWriter, r *http.Request, userID, taskID int) {
	task, err := modules.RedisClient.GetTask(taskID)
	if err != nil {
		respondWithError(w, "Task not found", http.StatusNotFound)
		return
	}

	if task.UserID != userID {
		respondWithError(w, "Task does not belong to this user", http.StatusNotFound)
		return
	}

	respondWithSuccess(w, task)
}

func updateUserTask(w http.ResponseWriter, r *http.Request, userID, taskID int) {
	var req models.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	task, err := modules.RedisClient.GetTask(taskID)
	if err != nil {
		respondWithError(w, "Task not found", http.StatusNotFound)
		return
	}

	if task.UserID != userID {
		respondWithError(w, "Task does not belong to this user", http.StatusNotFound)
		return
	}

	authCtx := modules.GetAuthContext(r)
	if !modules.CanModifyTask(authCtx, task) {
		respondWithError(w, "Insufficient permissions to modify this task", http.StatusForbidden)
		return
	}

	// Update fields
	if req.Title != "" {
		task.Title = req.Title
	}
	if req.Priority != 0 {
		task.Priority = req.Priority
	}
	if req.Deadline != "" {
		task.Deadline = req.Deadline
	}
	if req.Information != "" {
		task.Information = req.Information
	}
	if req.Status != nil {
		task.Status = *req.Status
	}
	if req.GroupID != 0 {
		// Validate group exists and user belongs to it
		_, err := modules.RedisClient.GetGroup(req.GroupID)
		if err != nil {
			respondWithError(w, "Group not found", http.StatusBadRequest)
			return
		}
		task.GroupID = req.GroupID
	}

	task.UpdatedAt = time.Now()

	if err := modules.RedisClient.SaveTask(task); err != nil {
		respondWithError(w, "Failed to update task", http.StatusInternalServerError)
		return
	}

	// Mark data as dirty for sync
	modules.RedisClient.MarkDirty("tasks")

	respondWithSuccess(w, map[string]interface{}{
		"message": "Task updated successfully",
		"task":    task,
	})
}

func deleteUserTask(w http.ResponseWriter, r *http.Request, userID, taskID int) {
	task, err := modules.RedisClient.GetTask(taskID)
	if err != nil {
		respondWithError(w, "Task not found", http.StatusNotFound)
		return
	}

	if task.UserID != userID {
		respondWithError(w, "Task does not belong to this user", http.StatusNotFound)
		return
	}

	authCtx := modules.GetAuthContext(r)
	if !modules.CanModifyTask(authCtx, task) {
		respondWithError(w, "Insufficient permissions to delete this task", http.StatusForbidden)
		return
	}

	if err := modules.RedisClient.DeleteTask(taskID); err != nil {
		respondWithError(w, "Failed to delete task", http.StatusInternalServerError)
		return
	}

	// Mark data as dirty for sync
	modules.RedisClient.MarkDirty("tasks")

	respondWithSuccess(w, map[string]interface{}{
		"message": "Task deleted successfully",
		"task":    task,
	})
}

func markUserTaskDone(w http.ResponseWriter, r *http.Request, userID, taskID int) {
	task, err := modules.RedisClient.GetTask(taskID)
	if err != nil {
		respondWithError(w, "Task not found", http.StatusNotFound)
		return
	}

	if task.UserID != userID {
		respondWithError(w, "Task does not belong to this user", http.StatusNotFound)
		return
	}

	authCtx := modules.GetAuthContext(r)
	if !modules.CanModifyTask(authCtx, task) {
		respondWithError(w, "Insufficient permissions to modify this task", http.StatusForbidden)
		return
	}

	task.Status = true
	task.UpdatedAt = time.Now()

	if err := modules.RedisClient.SaveTask(task); err != nil {
		respondWithError(w, "Failed to update task", http.StatusInternalServerError)
		return
	}

	// Mark data as dirty for sync
	modules.RedisClient.MarkDirty("tasks")

	respondWithSuccess(w, map[string]interface{}{
		"message": "Task marked as done",
		"task":    task,
	})
}

func getUserWorkTimes(w http.ResponseWriter, r *http.Request, userID int) {
	user, err := modules.RedisClient.GetUser(userID)
	if err != nil {
		respondWithError(w, "User not found", http.StatusNotFound)
		return
	}

	respondWithSuccess(w, map[string]interface{}{
		"user_id":    userID,
		"work_times": user.WorkTimes,
	})
}

func updateUserWorkTimes(w http.ResponseWriter, r *http.Request, userID int) {
	var req models.WorkTimesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	user, err := modules.RedisClient.GetUser(userID)
	if err != nil {
		respondWithError(w, "User not found", http.StatusNotFound)
		return
	}

	user.WorkTimes = models.WorkTimes(req.WorkTimes)
	user.UpdatedAt = time.Now()

	if err := modules.RedisClient.SaveUser(user); err != nil {
		respondWithError(w, "Failed to update work times", http.StatusInternalServerError)
		return
	}

	// Mark data as dirty for sync
	modules.RedisClient.MarkDirty("users")

	respondWithSuccess(w, map[string]interface{}{
		"message":    "Work times updated successfully",
		"work_times": user.WorkTimes,
	})
}

// Helper functions
func respondWithSuccess(w http.ResponseWriter, data interface{}, statusCode ...int) {
	code := http.StatusOK
	if len(statusCode) > 0 {
		code = statusCode[0]
	}

	response := models.APIResponse{
		Success: true,
		Data:    data,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(response)
}

func respondWithError(w http.ResponseWriter, message string, statusCode int) {
	response := models.APIResponse{
		Success: false,
		Error:   message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}
