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

// GroupsHandler handles /groups endpoint
func GroupsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		getAllGroups(w, r)
	case "POST":
		createGroup(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// GroupHandler handles /groups/{id} and sub-paths
func GroupHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/groups/")
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
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	// Check if group exists
	_, err = modules.RedisClient.GetGroup(id)
	if err != nil {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}

	if len(parts) == 1 {
		// /groups/{id}
		switch r.Method {
		case "GET":
			getGroup(w, r, id)
		case "PUT":
			updateGroup(w, r, id)
		case "DELETE":
			deleteGroup(w, r, id)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	subPath := parts[1]
	switch subPath {
	case "users":
		handleGroupUsers(w, r, id, parts[2:])
	case "tasks":
		GetTasksByGroupHandler(w, r, id)
	case "stats":
		getGroupStats(w, r, id)
	default:
		http.Error(w, "Invalid sub-path", http.StatusBadRequest)
	}
}

func getAllGroups(w http.ResponseWriter, r *http.Request) {
	authCtx := modules.GetAuthContext(r)

	groups, err := modules.RedisClient.GetAllGroups()
	if err != nil {
		respondWithError(w, fmt.Sprintf("Failed to get groups: %v", err), http.StatusInternalServerError)
		return
	}

	// Filter groups based on permissions
	var filteredGroups []*models.Group

	if authCtx.IsOwner {
		filteredGroups = groups // Owner sees all groups
	} else if authCtx.IsGroupAdmin {
		// Group admin sees their own administered groups
		for _, group := range groups {
			for _, adminGroupID := range authCtx.AdminGroupIDs {
				if group.ID == adminGroupID {
					filteredGroups = append(filteredGroups, group)
					break
				}
			}
		}
	} else {
		// Regular users see groups they belong to
		for _, group := range groups {
			for _, userGroupID := range authCtx.User.GroupIDs {
				if group.ID == userGroupID {
					filteredGroups = append(filteredGroups, group)
					break
				}
			}
		}
	}

	respondWithSuccess(w, map[string]interface{}{
		"groups": filteredGroups,
		"count":  len(filteredGroups),
	})
}

func createGroup(w http.ResponseWriter, r *http.Request) {
	authCtx := modules.GetAuthContext(r)

	// Only owner can create groups
	if !authCtx.IsOwner {
		respondWithError(w, "Only owner can create groups", http.StatusForbidden)
		return
	}

	var req models.CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		respondWithError(w, "Group name is required", http.StatusBadRequest)
		return
	}

	if req.AdminID == 0 {
		respondWithError(w, "Admin ID is required", http.StatusBadRequest)
		return
	}

	// Check if admin user exists and is eligible
	admin, err := modules.RedisClient.GetUser(req.AdminID)
	if err != nil {
		respondWithError(w, "Admin user not found", http.StatusBadRequest)
		return
	}

	// Admin must be at least 'group_admin' or 'owner' role
	if admin.Role != "group_admin" && admin.Role != "owner" {
		respondWithError(w, "Selected user must have 'group_admin' or 'owner' role", http.StatusBadRequest)
		return
	}

	// Check if group name already exists
	existingGroups, _ := modules.RedisClient.GetAllGroups()
	for _, group := range existingGroups {
		if strings.EqualFold(group.Name, req.Name) {
			respondWithError(w, "Group with this name already exists", http.StatusConflict)
			return
		}
	}

	// Get next group ID
	groupID, err := modules.RedisClient.GetNextGroupID()
	if err != nil {
		respondWithError(w, "Failed to generate group ID", http.StatusInternalServerError)
		return
	}

	group := &models.Group{
		ID:        groupID,
		Name:      req.Name,
		AdminID:   req.AdminID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save group
	if err := modules.RedisClient.SaveGroup(group); err != nil {
		respondWithError(w, "Failed to save group", http.StatusInternalServerError)
		return
	}

	// Add admin to group automatically
	admin.GroupIDs = append(admin.GroupIDs, groupID)
	if err := modules.RedisClient.SaveUser(admin); err != nil {
		// Log error but don't fail the group creation
		fmt.Printf("Warning: Failed to add admin to group: %v\n", err)
	}

	// Mark data as dirty for sync
	modules.RedisClient.MarkDirty("groups")
	modules.RedisClient.MarkDirty("users")

	respondWithSuccess(w, map[string]interface{}{
		"message": "Group created successfully",
		"group":   group,
	}, http.StatusCreated)
}

func getGroup(w http.ResponseWriter, r *http.Request, id int) {
	group, err := modules.RedisClient.GetGroup(id)
	if err != nil {
		respondWithError(w, "Group not found", http.StatusNotFound)
		return
	}

	// Get group admin info
	admin, _ := modules.RedisClient.GetUser(group.AdminID)

	// Get group users
	users, _ := modules.RedisClient.GetGroupUsers(id)

	// Get group tasks count
	tasks, _ := modules.RedisClient.GetGroupTasks(id)

	result := map[string]interface{}{
		"id":          group.ID,
		"name":        group.Name,
		"admin_id":    group.AdminID,
		"created_at":  group.CreatedAt,
		"updated_at":  group.UpdatedAt,
		"users_count": len(users),
		"tasks_count": len(tasks),
	}

	if admin != nil {
		result["admin"] = map[string]interface{}{
			"id":        admin.ID,
			"full_name": admin.FullName,
			"email":     admin.Email,
		}
	}

	respondWithSuccess(w, result)
}

func updateGroup(w http.ResponseWriter, r *http.Request, id int) {
	authCtx := modules.GetAuthContext(r)

	// Only owner can update groups
	if !authCtx.IsOwner {
		respondWithError(w, "Only owner can update groups", http.StatusForbidden)
		return
	}

	var req models.UpdateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	group, err := modules.RedisClient.GetGroup(id)
	if err != nil {
		respondWithError(w, "Group not found", http.StatusNotFound)
		return
	}

	// Update fields
	if req.Name != "" {
		// Check if new name already exists (for other groups)
		existingGroups, _ := modules.RedisClient.GetAllGroups()
		for _, existingGroup := range existingGroups {
			if existingGroup.ID != id && strings.EqualFold(existingGroup.Name, req.Name) {
				respondWithError(w, "Group with this name already exists", http.StatusConflict)
				return
			}
		}
		group.Name = req.Name
	}

	if req.AdminID != 0 && req.AdminID != group.AdminID {
		// Validate new admin
		newAdmin, err := modules.RedisClient.GetUser(req.AdminID)
		if err != nil {
			respondWithError(w, "New admin user not found", http.StatusBadRequest)
			return
		}

		if newAdmin.Role != "group_admin" && newAdmin.Role != "owner" {
			respondWithError(w, "New admin must have 'group_admin' or 'owner' role", http.StatusBadRequest)
			return
		}

		// Update admin
		oldAdminID := group.AdminID
		group.AdminID = req.AdminID

		// Add new admin to group if not already a member
		found := false
		for _, groupID := range newAdmin.GroupIDs {
			if groupID == id {
				found = true
				break
			}
		}
		if !found {
			newAdmin.GroupIDs = append(newAdmin.GroupIDs, id)
			modules.RedisClient.SaveUser(newAdmin)
		}

		// Optional: Remove old admin from group (uncomment if needed)
		// oldAdmin, err := modules.RedisClient.GetUser(oldAdminID)
		// if err == nil {
		//     var newGroupIDs []int
		//     for _, groupID := range oldAdmin.GroupIDs {
		//         if groupID != id {
		//             newGroupIDs = append(newGroupIDs, groupID)
		//         }
		//     }
		//     oldAdmin.GroupIDs = newGroupIDs
		//     modules.RedisClient.SaveUser(oldAdmin)
		// }

		fmt.Printf("Group %d admin changed from %d to %d\n", id, oldAdminID, req.AdminID)
	}

	group.UpdatedAt = time.Now()

	// Save group
	if err := modules.RedisClient.SaveGroup(group); err != nil {
		respondWithError(w, "Failed to update group", http.StatusInternalServerError)
		return
	}

	// Mark data as dirty for sync
	modules.RedisClient.MarkDirty("groups")

	respondWithSuccess(w, map[string]interface{}{
		"message": "Group updated successfully",
		"group":   group,
	})
}

func deleteGroup(w http.ResponseWriter, r *http.Request, id int) {
	authCtx := modules.GetAuthContext(r)

	// Only owner can delete groups
	if !authCtx.IsOwner {
		respondWithError(w, "Only owner can delete groups", http.StatusForbidden)
		return
	}

	group, err := modules.RedisClient.GetGroup(id)
	if err != nil {
		respondWithError(w, "Group not found", http.StatusNotFound)
		return
	}

	// Get all users in this group
	users, _ := modules.RedisClient.GetGroupUsers(id)

	// Remove group from all users
	for _, user := range users {
		var newGroupIDs []int
		for _, groupID := range user.GroupIDs {
			if groupID != id {
				newGroupIDs = append(newGroupIDs, groupID)
			}
		}
		user.GroupIDs = models.IntSlice(newGroupIDs)
		modules.RedisClient.SaveUser(user)
	}

	// Delete all tasks in this group
	tasks, _ := modules.RedisClient.GetGroupTasks(id)
	for _, task := range tasks {
		modules.RedisClient.DeleteTask(task.ID)
	}

	// Delete group
	if err := modules.RedisClient.DeleteGroup(id); err != nil {
		respondWithError(w, "Failed to delete group", http.StatusInternalServerError)
		return
	}

	// Mark data as dirty for sync
	modules.RedisClient.MarkDirty("groups")
	modules.RedisClient.MarkDirty("users")
	modules.RedisClient.MarkDirty("tasks")

	respondWithSuccess(w, map[string]interface{}{
		"message":        "Group deleted successfully",
		"group":          group,
		"affected_users": len(users),
		"deleted_tasks":  len(tasks),
	})
}

func handleGroupUsers(w http.ResponseWriter, r *http.Request, groupID int, remainingParts []string) {
	if len(remainingParts) == 0 {
		// /groups/{id}/users
		switch r.Method {
		case "GET":
			getGroupUsers(w, r, groupID)
		case "POST":
			addUserToGroup(w, r, groupID)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	if len(remainingParts) == 1 {
		userIDStr := remainingParts[0]
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		// /groups/{id}/users/{uid}
		switch r.Method {
		case "DELETE":
			removeUserFromGroup(w, r, groupID, userID)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func getGroupUsers(w http.ResponseWriter, r *http.Request, groupID int) {
	users, err := modules.RedisClient.GetGroupUsers(groupID)
	if err != nil {
		respondWithError(w, fmt.Sprintf("Failed to get group users: %v", err), http.StatusInternalServerError)
		return
	}

	// Remove passwords from response
	var safeUsers []map[string]interface{}
	for _, user := range users {
		safeUser := map[string]interface{}{
			"id":         user.ID,
			"full_name":  user.FullName,
			"role":       user.Role,
			"email":      user.Email,
			"number":     user.Number,
			"created_at": user.CreatedAt,
		}
		safeUsers = append(safeUsers, safeUser)
	}

	respondWithSuccess(w, map[string]interface{}{
		"group_id": groupID,
		"users":    safeUsers,
		"count":    len(safeUsers),
	})
}

func addUserToGroup(w http.ResponseWriter, r *http.Request, groupID int) {
	authCtx := modules.GetAuthContext(r)

	// Check permissions: owner or group admin can add users
	canAdd := authCtx.IsOwner
	if authCtx.IsGroupAdmin {
		for _, adminGroupID := range authCtx.AdminGroupIDs {
			if adminGroupID == groupID {
				canAdd = true
				break
			}
		}
	}

	if !canAdd {
		respondWithError(w, "Insufficient permissions to add users to this group", http.StatusForbidden)
		return
	}

	var req struct {
		UserID int `json:"user_id" binding:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.UserID == 0 {
		respondWithError(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Check if user exists
	user, err := modules.RedisClient.GetUser(req.UserID)
	if err != nil {
		respondWithError(w, "User not found", http.StatusBadRequest)
		return
	}

	// Check if user is already in group
	for _, userGroupID := range user.GroupIDs {
		if userGroupID == groupID {
			respondWithError(w, "User is already a member of this group", http.StatusConflict)
			return
		}
	}

	// Add user to group
	user.GroupIDs = append(user.GroupIDs, groupID)
	if err := modules.RedisClient.SaveUser(user); err != nil {
		respondWithError(w, "Failed to add user to group", http.StatusInternalServerError)
		return
	}

	// Mark data as dirty for sync
	modules.RedisClient.MarkDirty("users")

	respondWithSuccess(w, map[string]interface{}{
		"message":  "User added to group successfully",
		"group_id": groupID,
		"user_id":  req.UserID,
	})
}

func removeUserFromGroup(w http.ResponseWriter, r *http.Request, groupID, userID int) {
	authCtx := modules.GetAuthContext(r)

	// Check permissions: owner or group admin can remove users
	canRemove := authCtx.IsOwner
	if authCtx.IsGroupAdmin {
		for _, adminGroupID := range authCtx.AdminGroupIDs {
			if adminGroupID == groupID {
				canRemove = true
				break
			}
		}
	}

	if !canRemove {
		respondWithError(w, "Insufficient permissions to remove users from this group", http.StatusForbidden)
		return
	}

	// Check if user exists
	user, err := modules.RedisClient.GetUser(userID)
	if err != nil {
		respondWithError(w, "User not found", http.StatusBadRequest)
		return
	}

	// Check if user is in group
	found := false
	var newGroupIDs []int
	for _, userGroupID := range user.GroupIDs {
		if userGroupID == groupID {
			found = true
		} else {
			newGroupIDs = append(newGroupIDs, userGroupID)
		}
	}

	if !found {
		respondWithError(w, "User is not a member of this group", http.StatusNotFound)
		return
	}

	// Remove user from group
	user.GroupIDs = models.IntSlice(newGroupIDs)
	if err := modules.RedisClient.SaveUser(user); err != nil {
		respondWithError(w, "Failed to remove user from group", http.StatusInternalServerError)
		return
	}

	// Optionally move or delete user's tasks in this group
	tasks, _ := modules.RedisClient.GetUserTasks(userID)
	var affectedTasks int
	for _, task := range tasks {
		if task.GroupID == groupID {
			// Option 1: Delete task
			// modules.RedisClient.DeleteTask(task.ID)
			// affectedTasks++

			// Option 2: Move task to user's first available group
			if len(user.GroupIDs) > 0 {
				task.GroupID = user.GroupIDs[0]
				modules.RedisClient.SaveTask(task)
				affectedTasks++
			} else {
				// User has no groups, delete the task
				modules.RedisClient.DeleteTask(task.ID)
				affectedTasks++
			}
		}
	}

	// Mark data as dirty for sync
	modules.RedisClient.MarkDirty("users")
	if affectedTasks > 0 {
		modules.RedisClient.MarkDirty("tasks")
	}

	respondWithSuccess(w, map[string]interface{}{
		"message":        "User removed from group successfully",
		"group_id":       groupID,
		"user_id":        userID,
		"affected_tasks": affectedTasks,
	})
}

func getGroupStats(w http.ResponseWriter, r *http.Request, groupID int) {
	authCtx := modules.GetAuthContext(r)

	// Check permissions for group access
	canAccess := authCtx.IsOwner
	if authCtx.IsGroupAdmin {
		for _, adminGroupID := range authCtx.AdminGroupIDs {
			if adminGroupID == groupID {
				canAccess = true
				break
			}
		}
	} else if authCtx.User != nil {
		// Regular users can see stats of groups they belong to
		for _, userGroupID := range authCtx.User.GroupIDs {
			if userGroupID == groupID {
				canAccess = true
				break
			}
		}
	}

	if !canAccess {
		respondWithError(w, "Insufficient permissions to view group stats", http.StatusForbidden)
		return
	}

	// Get group info
	group, err := modules.RedisClient.GetGroup(groupID)
	if err != nil {
		respondWithError(w, "Group not found", http.StatusNotFound)
		return
	}

	// Get group users
	users, err := modules.RedisClient.GetGroupUsers(groupID)
	if err != nil {
		respondWithError(w, "Failed to get group users", http.StatusInternalServerError)
		return
	}

	// Get group tasks
	tasks, err := modules.RedisClient.GetGroupTasks(groupID)
	if err != nil {
		respondWithError(w, "Failed to get group tasks", http.StatusInternalServerError)
		return
	}

	// Calculate stats
	totalTasks := len(tasks)
	completedTasks := 0
	pendingTasks := 0
	userTaskCounts := make(map[string]int)

	for _, task := range tasks {
		if task.Status {
			completedTasks++
		} else {
			pendingTasks++
		}

		// Find user name for task count
		for _, user := range users {
			if user.ID == task.UserID {
				userTaskCounts[user.FullName]++
				break
			}
		}
	}

	completionRate := 0.0
	if totalTasks > 0 {
		completionRate = float64(completedTasks) / float64(totalTasks) * 100
	}

	stats := map[string]interface{}{
		"group": map[string]interface{}{
			"id":   group.ID,
			"name": group.Name,
		},
		"users_count":      len(users),
		"total_tasks":      totalTasks,
		"completed_tasks":  completedTasks,
		"pending_tasks":    pendingTasks,
		"completion_rate":  completionRate,
		"user_task_counts": userTaskCounts,
	}

	respondWithSuccess(w, stats)
}
