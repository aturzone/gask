package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"task-manager/models"
	"task-manager/modules"
)

// SearchTasksHandler handles global task search /tasks/search
func SearchTasksHandler(w http.ResponseWriter, r *http.Request) {
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

	searchResults, err := modules.RedisClient.SearchTasks(query)
	if err != nil {
		respondWithError(w, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Filter results based on permissions
	filteredResults := filterTaskSearchResults(authCtx, searchResults)

	respondWithSuccess(w, map[string]interface{}{
		"query":   query,
		"results": filteredResults,
		"count":   len(filteredResults),
	})
}

// GetTaskStatsHandler provides task statistics
func GetTaskStatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authCtx := modules.GetAuthContext(r)

	// Get statistics based on user permissions
	stats := make(map[string]interface{})

	if authCtx.IsOwner {
		// Owner sees global stats
		globalStats, err := getGlobalTaskStats()
		if err != nil {
			respondWithError(w, fmt.Sprintf("Failed to get stats: %v", err), http.StatusInternalServerError)
			return
		}
		stats = globalStats
	} else if authCtx.IsGroupAdmin {
		// Group admin sees their groups' stats
		groupStats, err := getGroupAdminTaskStats(authCtx.AdminGroupIDs)
		if err != nil {
			respondWithError(w, fmt.Sprintf("Failed to get stats: %v", err), http.StatusInternalServerError)
			return
		}
		stats = groupStats
	} else {
		// Regular user sees only their own stats
		userStats, err := getUserTaskStats(authCtx.User.ID)
		if err != nil {
			respondWithError(w, fmt.Sprintf("Failed to get stats: %v", err), http.StatusInternalServerError)
			return
		}
		stats = userStats
	}

	respondWithSuccess(w, stats)
}

// GetTasksByGroupHandler handles /groups/{id}/tasks
func GetTasksByGroupHandler(w http.ResponseWriter, r *http.Request, groupID int) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authCtx := modules.GetAuthContext(r)

	// Check permissions for group access
	if !authCtx.IsOwner {
		hasAccess := false

		if authCtx.IsGroupAdmin {
			for _, adminGroupID := range authCtx.AdminGroupIDs {
				if adminGroupID == groupID {
					hasAccess = true
					break
				}
			}
		} else {
			// Regular users can see tasks from groups they belong to
			for _, userGroupID := range authCtx.User.GroupIDs {
				if userGroupID == groupID {
					hasAccess = true
					break
				}
			}
		}

		if !hasAccess {
			respondWithError(w, "Insufficient permissions to view group tasks", http.StatusForbidden)
			return
		}
	}

	tasks, err := modules.RedisClient.GetGroupTasks(groupID)
	if err != nil {
		respondWithError(w, fmt.Sprintf("Failed to get group tasks: %v", err), http.StatusInternalServerError)
		return
	}

	// Further filter tasks if user is not owner/group admin
	if !authCtx.IsOwner && !authCtx.IsGroupAdmin {
		var filteredTasks []*models.Task
		for _, task := range tasks {
			if task.UserID == authCtx.User.ID {
				filteredTasks = append(filteredTasks, task)
			}
		}
		tasks = filteredTasks
	}

	respondWithSuccess(w, map[string]interface{}{
		"group_id": groupID,
		"tasks":    tasks,
		"count":    len(tasks),
	})
}

// Helper function for checking if user is in admin groups
func isUserInAdminGroups(userID int, adminGroupIDs []int) bool {
	user, err := modules.RedisClient.GetUser(userID)
	if err != nil {
		return false
	}

	for _, userGroupID := range user.GroupIDs {
		for _, adminGroupID := range adminGroupIDs {
			if userGroupID == adminGroupID {
				return true
			}
		}
	}
	return false
}

// Utility functions

func filterTaskSearchResults(authCtx *modules.AuthContext, searchResults []*models.SearchTask) []*models.SearchTask {
	if authCtx.IsOwner {
		return searchResults // Owner sees all
	}

	var filtered []*models.SearchTask
	for _, result := range searchResults {
		canSee := false

		// Users can see their own tasks
		if result.Task.UserID == authCtx.User.ID {
			canSee = true
		} else if authCtx.IsGroupAdmin {
			// Group admins can see tasks from their administered groups
			for _, adminGroupID := range authCtx.AdminGroupIDs {
				if result.Task.GroupID == adminGroupID {
					canSee = true
					break
				}
			}
		} else {
			// Regular users can see tasks from groups they belong to (but only their own tasks were already filtered above)
			for _, userGroupID := range authCtx.User.GroupIDs {
				if result.Task.GroupID == userGroupID && result.Task.UserID == authCtx.User.ID {
					canSee = true
					break
				}
			}
		}

		if canSee {
			filtered = append(filtered, result)
		}
	}

	return filtered
}

func getGlobalTaskStats() (map[string]interface{}, error) {
	users, err := modules.RedisClient.GetAllUsers()
	if err != nil {
		return nil, err
	}

	stats := make(map[string]interface{})
	totalTasks := 0
	completedTasks := 0
	pendingTasks := 0
	userTaskCounts := make(map[string]int)
	groupTaskCounts := make(map[int]int)

	for _, user := range users {
		tasks, err := modules.RedisClient.GetUserTasks(user.ID)
		if err != nil {
			continue
		}

		userTaskCount := len(tasks)
		userTaskCounts[user.FullName] = userTaskCount
		totalTasks += userTaskCount

		for _, task := range tasks {
			if task.Status {
				completedTasks++
			} else {
				pendingTasks++
			}

			groupTaskCounts[task.GroupID]++
		}
	}

	stats["total_tasks"] = totalTasks
	stats["completed_tasks"] = completedTasks
	stats["pending_tasks"] = pendingTasks
	stats["completion_rate"] = 0.0

	if totalTasks > 0 {
		stats["completion_rate"] = float64(completedTasks) / float64(totalTasks) * 100
	}

	stats["user_task_counts"] = userTaskCounts
	stats["group_task_counts"] = groupTaskCounts
	stats["total_users"] = len(users)

	return stats, nil
}

func getGroupAdminTaskStats(adminGroupIDs []int) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	totalTasks := 0
	completedTasks := 0
	pendingTasks := 0
	userTaskCounts := make(map[string]int)
	groupTaskCounts := make(map[int]int)

	for _, groupID := range adminGroupIDs {
		tasks, err := modules.RedisClient.GetGroupTasks(groupID)
		if err != nil {
			continue
		}

		groupTaskCount := len(tasks)
		groupTaskCounts[groupID] = groupTaskCount
		totalTasks += groupTaskCount

		for _, task := range tasks {
			if task.Status {
				completedTasks++
			} else {
				pendingTasks++
			}

			// Get user name for stats
			user, err := modules.RedisClient.GetUser(task.UserID)
			if err == nil {
				userTaskCounts[user.FullName]++
			}
		}
	}

	stats["total_tasks"] = totalTasks
	stats["completed_tasks"] = completedTasks
	stats["pending_tasks"] = pendingTasks
	stats["completion_rate"] = 0.0

	if totalTasks > 0 {
		stats["completion_rate"] = float64(completedTasks) / float64(totalTasks) * 100
	}

	stats["user_task_counts"] = userTaskCounts
	stats["group_task_counts"] = groupTaskCounts
	stats["administered_groups"] = adminGroupIDs

	return stats, nil
}

func getUserTaskStats(userID int) (map[string]interface{}, error) {
	tasks, err := modules.RedisClient.GetUserTasks(userID)
	if err != nil {
		return nil, err
	}

	stats := make(map[string]interface{})
	totalTasks := len(tasks)
	completedTasks := 0
	pendingTasks := 0
	priorityCounts := make(map[int]int)
	groupTaskCounts := make(map[int]int)

	for _, task := range tasks {
		if task.Status {
			completedTasks++
		} else {
			pendingTasks++
		}

		priorityCounts[task.Priority]++
		groupTaskCounts[task.GroupID]++
	}

	stats["total_tasks"] = totalTasks
	stats["completed_tasks"] = completedTasks
	stats["pending_tasks"] = pendingTasks
	stats["completion_rate"] = 0.0

	if totalTasks > 0 {
		stats["completion_rate"] = float64(completedTasks) / float64(totalTasks) * 100
	}

	stats["priority_counts"] = priorityCounts
	stats["group_task_counts"] = groupTaskCounts
	stats["user_id"] = userID

	return stats, nil
}

// Task batch operations

// BatchUpdateTasksHandler handles batch operations on tasks
func BatchUpdateTasksHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TaskIDs []int                    `json:"task_ids" binding:"required"`
		Updates models.UpdateTaskRequest `json:"updates" binding:"required"`
		Action  string                   `json:"action"` // "update", "delete", "mark_done"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.TaskIDs) == 0 {
		respondWithError(w, "Task IDs are required", http.StatusBadRequest)
		return
	}

	authCtx := modules.GetAuthContext(r)
	var updatedTasks []*models.Task
	var errors []string

	for _, taskID := range req.TaskIDs {
		task, err := modules.RedisClient.GetTask(taskID)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Task %d not found", taskID))
			continue
		}

		// Check permissions
		if !modules.CanModifyTask(authCtx, task) {
			errors = append(errors, fmt.Sprintf("Insufficient permissions for task %d", taskID))
			continue
		}

		// Perform action
		switch req.Action {
		case "mark_done":
			task.Status = true
		case "delete":
			if err := modules.RedisClient.DeleteTask(taskID); err != nil {
				errors = append(errors, fmt.Sprintf("Failed to delete task %d", taskID))
			}
			continue
		default: // "update" or empty (default to update)
			// Apply updates
			if req.Updates.Title != "" {
				task.Title = req.Updates.Title
			}
			if req.Updates.Priority != 0 {
				task.Priority = req.Updates.Priority
			}
			if req.Updates.Deadline != "" {
				task.Deadline = req.Updates.Deadline
			}
			if req.Updates.Information != "" {
				task.Information = req.Updates.Information
			}
			if req.Updates.Status != nil {
				task.Status = *req.Updates.Status
			}
			if req.Updates.GroupID != 0 {
				task.GroupID = req.Updates.GroupID
			}
		}

		// Save updated task
		if err := modules.RedisClient.SaveTask(task); err != nil {
			errors = append(errors, fmt.Sprintf("Failed to save task %d", taskID))
			continue
		}

		updatedTasks = append(updatedTasks, task)
	}

	// Mark data as dirty for sync
	modules.RedisClient.MarkDirty("tasks")

	result := map[string]interface{}{
		"updated_count":   len(updatedTasks),
		"updated_tasks":   updatedTasks,
		"total_requested": len(req.TaskIDs),
	}

	if len(errors) > 0 {
		result["errors"] = errors
		result["error_count"] = len(errors)
	}

	respondWithSuccess(w, result)
}

// GetTasksWithFiltersHandler provides advanced task filtering
func GetTasksWithFiltersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	query := r.URL.Query()

	status := query.Get("status")     // "completed", "pending", "all"
	priority := query.Get("priority") // "1", "2", "3", etc.
	groupID := query.Get("group_id")  // filter by group
	userID := query.Get("user_id")    // filter by user (if permitted)

	authCtx := modules.GetAuthContext(r)

	var allTasks []*models.Task
	var err error

	// Determine which tasks to fetch based on permissions
	if authCtx.IsOwner {
		// Owner can see all tasks
		users, err := modules.RedisClient.GetAllUsers()
		if err != nil {
			respondWithError(w, "Failed to get users", http.StatusInternalServerError)
			return
		}

		for _, user := range users {
			tasks, err := modules.RedisClient.GetUserTasks(user.ID)
			if err != nil {
				continue
			}
			allTasks = append(allTasks, tasks...)
		}
	} else if authCtx.IsGroupAdmin {
		// Group admin sees tasks from their groups
		for _, adminGroupID := range authCtx.AdminGroupIDs {
			tasks, err := modules.RedisClient.GetGroupTasks(adminGroupID)
			if err != nil {
				continue
			}
			allTasks = append(allTasks, tasks...)
		}
	} else {
		// Regular user sees only their own tasks
		allTasks, err = modules.RedisClient.GetUserTasks(authCtx.User.ID)
		if err != nil {
			respondWithError(w, "Failed to get user tasks", http.StatusInternalServerError)
			return
		}
	}

	// Apply filters
	var filteredTasks []*models.Task

	for _, task := range allTasks {
		// Status filter
		if status != "" && status != "all" {
			if status == "completed" && !task.Status {
				continue
			}
			if status == "pending" && task.Status {
				continue
			}
		}

		// Priority filter
		if priority != "" {
			var requestedPriority int
			if n, err := fmt.Sscanf(priority, "%d", &requestedPriority); err != nil || n != 1 || task.Priority != requestedPriority {
				continue
			}
		}

		// Group filter
		if groupID != "" {
			var requestedGroupID int
			if n, err := fmt.Sscanf(groupID, "%d", &requestedGroupID); err != nil || n != 1 || task.GroupID != requestedGroupID {
				continue
			}
		}

		// User filter (only if permitted)
		if userID != "" {
			requestedUserID := 0
			fmt.Sscanf(userID, "%d", &requestedUserID)

			// Check if requester can see this user's tasks
			if !authCtx.IsOwner && authCtx.User.ID != requestedUserID {
				if !authCtx.IsGroupAdmin || !isUserInAdminGroups(requestedUserID, authCtx.AdminGroupIDs) {
					continue
				}
			}

			if task.UserID != requestedUserID {
				continue
			}
		}

		filteredTasks = append(filteredTasks, task)
	}

	respondWithSuccess(w, map[string]interface{}{
		"tasks": filteredTasks,
		"count": len(filteredTasks),
		"filters": map[string]string{
			"status":   status,
			"priority": priority,
			"group_id": groupID,
			"user_id":  userID,
		},
	})
}
