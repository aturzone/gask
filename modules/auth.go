package modules

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"task-manager/models"
)

// AuthContext holds authentication information
type AuthContext struct {
	User          *models.User
	IsOwner       bool
	IsGroupAdmin  bool
	AdminGroupIDs []int
}

// AuthMiddleware enforces authentication and authorization
func AuthMiddleware(ownerPassword string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Allow CORS preflight through
			if r.Method == "OPTIONS" {
				next.ServeHTTP(w, r)
				return
			}

			// Allow health check without authentication
			if r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			authCtx, err := authenticate(r, ownerPassword)
			if err != nil {
				w.Header().Set("WWW-Authenticate", `Basic realm="User Area"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Check authorization for the requested resource
			if !isAuthorized(authCtx, r) {
				http.Error(w, "Forbidden: Insufficient permissions", http.StatusForbidden)
				return
			}

			// Store auth context in request context for handlers
			ctx := SetAuthContext(r.Context(), authCtx)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

// authenticate validates credentials and returns AuthContext
func authenticate(r *http.Request, ownerPassword string) (*AuthContext, error) {
	// 1) Owner header check
	if r.Header.Get("X-Owner-Password") == ownerPassword {
		return &AuthContext{
			User:          nil, // Owner doesn't need a user object
			IsOwner:       true,
			IsGroupAdmin:  false,
			AdminGroupIDs: []int{},
		}, nil
	}

	// 2) Basic Auth user check
	userStr, pass, ok := r.BasicAuth()
	if !ok {
		return nil, http.ErrNoCookie
	}

	// Try to get user by ID first
	userID, err := strconv.Atoi(userStr)
	var user *models.User

	if err == nil {
		// userStr is numeric, try to get by ID
		user, err = RedisClient.GetUser(userID)
		if err != nil {
			return nil, err
		}
	} else {
		// userStr is not numeric, try to get by email
		user, err = RedisClient.GetUserByEmail(userStr)
		if err != nil {
			return nil, err
		}
	}

	// Check password
	if user.Password != pass {
		return nil, http.ErrNoCookie
	}

	// Build AuthContext
	authCtx := &AuthContext{
		User:          user,
		IsOwner:       user.Role == "owner",
		IsGroupAdmin:  user.Role == "group_admin",
		AdminGroupIDs: []int{},
	}

	// If user is group admin, find which groups they admin
	if authCtx.IsGroupAdmin {
		groups, err := RedisClient.GetAllGroups()
		if err == nil {
			for _, group := range groups {
				if group.AdminID == user.ID {
					authCtx.AdminGroupIDs = append(authCtx.AdminGroupIDs, group.ID)
				}
			}
		}
	}

	return authCtx, nil
}

// isAuthorized checks if the authenticated user has permission for the requested resource
func isAuthorized(authCtx *AuthContext, r *http.Request) bool {
	path := r.URL.Path
	method := r.Method

	// Owner has access to everything
	if authCtx.IsOwner {
		return true
	}

	// Parse the path to understand what resource is being accessed
	pathInfo := parseResourcePath(path)
	if pathInfo == nil {
		return false
	}

	// Check permissions based on resource type and user role
	switch pathInfo.ResourceType {
	case "users":
		return checkUserPermissions(authCtx, pathInfo, method)
	case "groups":
		return checkGroupPermissions(authCtx, pathInfo, method)
	case "tasks":
		return checkTaskPermissions(authCtx, pathInfo, method)
	case "search":
		return checkSearchPermissions(authCtx, pathInfo, method)
	default:
		return false
	}
}

// ResourcePathInfo holds parsed information about the requested resource
type ResourcePathInfo struct {
	ResourceType  string // "users", "groups", "tasks", "search"
	ResourceID    int    // ID of the main resource
	SubResource   string // "tasks", "worktimes", etc.
	SubResourceID int    // ID of sub-resource
	Action        string // "done", etc.
	IsGlobal      bool   // true for /tasks/search, /users/search
}

// parseResourcePath extracts resource information from URL path
func parseResourcePath(path string) *ResourcePathInfo {
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return nil
	}

	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return nil
	}

	info := &ResourcePathInfo{}

	// Handle global endpoints
	if parts[0] == "tasks" && len(parts) >= 2 && parts[1] == "search" {
		info.ResourceType = "search"
		info.IsGlobal = true
		return info
	}

	if parts[0] == "users" && len(parts) >= 2 && parts[1] == "search" {
		info.ResourceType = "search"
		info.IsGlobal = true
		return info
	}

	// Handle standard resource paths
	info.ResourceType = parts[0]

	if len(parts) >= 2 && parts[1] != "" {
		if id, err := strconv.Atoi(parts[1]); err == nil {
			info.ResourceID = id
		}
	}

	if len(parts) >= 3 && parts[2] != "" {
		info.SubResource = parts[2]
	}

	if len(parts) >= 4 && parts[3] != "" {
		if id, err := strconv.Atoi(parts[3]); err == nil {
			info.SubResourceID = id
		} else {
			info.Action = parts[3]
		}
	}

	if len(parts) >= 5 && parts[4] != "" {
		info.Action = parts[4]
	}

	return info
}

// checkUserPermissions validates permissions for user-related endpoints
func checkUserPermissions(authCtx *AuthContext, pathInfo *ResourcePathInfo, method string) bool {
	user := authCtx.User

	// Global user operations (like /users, /users/search)
	if pathInfo.ResourceID == 0 {
		// Only owner and group admins can list/search all users
		// Group admins can see users in their groups
		return authCtx.IsOwner || authCtx.IsGroupAdmin
	}

	// Specific user operations (/users/{id})
	targetUserID := pathInfo.ResourceID

	// Users can always access their own data
	if user.ID == targetUserID {
		return true
	}

	// Owner can access any user
	if authCtx.IsOwner {
		return true
	}

	// Group admins can access users in their administered groups
	if authCtx.IsGroupAdmin {
		return isUserInAdminGroups(targetUserID, authCtx.AdminGroupIDs)
	}

	return false
}

// checkGroupPermissions validates permissions for group-related endpoints
func checkGroupPermissions(authCtx *AuthContext, pathInfo *ResourcePathInfo, method string) bool {
	// Global group operations
	if pathInfo.ResourceID == 0 {
		// Only owner can create groups or list all groups
		if method == "POST" {
			return authCtx.IsOwner
		}
		// Group admins can see their own groups
		return authCtx.IsOwner || authCtx.IsGroupAdmin
	}

	// Specific group operations
	groupID := pathInfo.ResourceID

	// Owner can access any group
	if authCtx.IsOwner {
		return true
	}

	// Group admins can only access their own administered groups
	if authCtx.IsGroupAdmin {
		for _, adminGroupID := range authCtx.AdminGroupIDs {
			if adminGroupID == groupID {
				return true
			}
		}
	}

	// Regular users can view groups they belong to (read-only)
	if method == "GET" {
		return isUserInGroup(authCtx.User.ID, groupID)
	}

	return false
}

// checkTaskPermissions validates permissions for task-related endpoints
func checkTaskPermissions(authCtx *AuthContext, pathInfo *ResourcePathInfo, method string) bool {
	// This handles both /users/{id}/tasks and direct task access
	user := authCtx.User

	// If accessing via /users/{id}/tasks
	if pathInfo.ResourceType == "users" && pathInfo.SubResource == "tasks" {
		targetUserID := pathInfo.ResourceID

		// Users can access their own tasks
		if user.ID == targetUserID {
			return true
		}

		// Owner can access any user's tasks
		if authCtx.IsOwner {
			return true
		}

		// Group admins can access tasks of users in their groups
		if authCtx.IsGroupAdmin {
			return isUserInAdminGroups(targetUserID, authCtx.AdminGroupIDs)
		}
	}

	// Direct task access would need task ID lookup
	// For now, we'll handle this in the handlers
	return true // Let handlers do detailed task-level permission checks
}

// checkSearchPermissions validates permissions for search endpoints
func checkSearchPermissions(authCtx *AuthContext, pathInfo *ResourcePathInfo, method string) bool {
	// Only GET method for searches
	if method != "GET" {
		return false
	}

	// Owner can search everything
	if authCtx.IsOwner {
		return true
	}

	// Group admins can search within their scope
	if authCtx.IsGroupAdmin {
		return true
	}

	// Regular users cannot use global search
	return false
}

// Helper functions
func isUserInAdminGroups(userID int, adminGroupIDs []int) bool {
	user, err := RedisClient.GetUser(userID)
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

func isUserInGroup(userID int, groupID int) bool {
	user, err := RedisClient.GetUser(userID)
	if err != nil {
		return false
	}

	for _, userGroupID := range user.GroupIDs {
		if userGroupID == groupID {
			return true
		}
	}
	return false
}

// GetAuthContext retrieves AuthContext from request context
func GetAuthContext(r *http.Request) *AuthContext {
	ctx := r.Context()
	if authCtx, ok := ctx.Value(authContextKey).(*AuthContext); ok {
		return authCtx
	}
	return nil
}

// SetAuthContext stores AuthContext in context
func SetAuthContext(ctx context.Context, authCtx *AuthContext) context.Context {
	return context.WithValue(ctx, authContextKey, authCtx)
}

// Context key for auth context
type contextKey string

const authContextKey contextKey = "auth"

// Validation helpers for handlers
func CanCreateUser(authCtx *AuthContext, targetRole string, groupIDs []int) bool {
	// Owner can create anyone
	if authCtx.IsOwner {
		return true
	}

	// Group admins can only create regular users in their groups
	if authCtx.IsGroupAdmin {
		if targetRole != "" && targetRole != "user" {
			return false // Can't create other admins
		}

		// All specified groups must be admin groups
		for _, groupID := range groupIDs {
			found := false
			for _, adminGroupID := range authCtx.AdminGroupIDs {
				if groupID == adminGroupID {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}

	return false
}

func CanModifyTask(authCtx *AuthContext, task *models.Task) bool {
	// Owner can modify any task
	if authCtx.IsOwner {
		return true
	}

	// Users can modify their own tasks
	if authCtx.User.ID == task.UserID {
		return true
	}

	// Group admins can modify tasks in their groups
	if authCtx.IsGroupAdmin {
		for _, adminGroupID := range authCtx.AdminGroupIDs {
			if task.GroupID == adminGroupID {
				return true
			}
		}
	}

	return false
}

func FilterUsersByPermissions(authCtx *AuthContext, users []*models.User) []*models.User {
	if authCtx.IsOwner {
		return users // Owner sees all
	}

	var filtered []*models.User
	for _, user := range users {
		if authCtx.User.ID == user.ID {
			filtered = append(filtered, user) // Own user
			continue
		}

		if authCtx.IsGroupAdmin {
			// Check if user is in admin's groups
			if isUserInAdminGroups(user.ID, authCtx.AdminGroupIDs) {
				filtered = append(filtered, user)
			}
		}
	}

	return filtered
}
