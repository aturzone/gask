package http

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/taskmaster/core/internal/application/services"
	"github.com/taskmaster/core/internal/domain/entities"
	"github.com/taskmaster/core/internal/infrastructure/logger"
	"github.com/taskmaster/core/internal/ports"
)

// Response types
type PaginatedResponse[T any] struct {
	Data   []T `json:"data"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type ErrorResponse struct {
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

type DeadlinesResponse struct {
	NearDeadline []*entities.Task `json:"near_deadline"`
	Overdue      []*entities.Task `json:"overdue"`
	Days         int              `json:"days"`
}

// AuthHandler handles authentication-related requests
type AuthHandler struct {
	authService *services.AuthService
	logger      *logger.Logger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *services.AuthService, logger *logger.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
	}
}

// Register godoc
// @Summary Register a new user
// @Description Create a new user account
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body ports.RegisterRequest true "User registration data"
// @Success 201 {object} ports.AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /auth/register [post]
func (h *AuthHandler) Register(c echo.Context) error {
	var req ports.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	response, err := h.authService.Register(c.Request().Context(), req)
	if err != nil {
		h.logger.Error("Registration failed", "error", err, "email", req.Email)
		if strings.Contains(err.Error(), "already exists") {
			return echo.NewHTTPError(http.StatusConflict, err.Error())
		}
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusCreated, response)
}

// Login godoc
// @Summary User login
// @Description Authenticate user and return access token
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body ports.LoginRequest true "Login credentials"
// @Success 200 {object} ports.AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/login [post]
func (h *AuthHandler) Login(c echo.Context) error {
	var req ports.LoginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	response, err := h.authService.Login(c.Request().Context(), req)
	if err != nil {
		h.logger.Warn("Login failed", "error", err, "email", req.Email, "ip", c.RealIP())
		if strings.Contains(err.Error(), "invalid credentials") || strings.Contains(err.Error(), "not found") {
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid credentials")
		}
		if strings.Contains(err.Error(), "inactive") {
			return echo.NewHTTPError(http.StatusUnauthorized, "Account is inactive")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Login failed")
	}

	return c.JSON(http.StatusOK, response)
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Get a new access token using refresh token
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body map[string]string true "Refresh token"
// @Success 200 {object} ports.AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c echo.Context) error {
	var req map[string]string
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	refreshToken, ok := req["refresh_token"]
	if !ok || refreshToken == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Refresh token is required")
	}

	response, err := h.authService.RefreshToken(c.Request().Context(), refreshToken)
	if err != nil {
		h.logger.Warn("Token refresh failed", "error", err, "ip", c.RealIP())
		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid refresh token")
	}

	return c.JSON(http.StatusOK, response)
}

// Logout godoc
// @Summary User logout
// @Description Logout user and revoke refresh tokens
// @Tags authentication
// @Accept json
// @Produce json
// @Success 200 {object} MessageResponse
// @Failure 401 {object} ErrorResponse
// @Security BearerAuth
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c echo.Context) error {
	userID := getUserIDFromContext(c)

	err := h.authService.Logout(c.Request().Context(), userID)
	if err != nil {
		h.logger.Error("Logout failed", "error", err, "user_id", userID)
		return echo.NewHTTPError(http.StatusInternalServerError, "Logout failed")
	}

	return c.JSON(http.StatusOK, MessageResponse{Message: "Logged out successfully"})
}

// UserHandler handles user-related requests
type UserHandler struct {
	userService *services.UserService
	logger      *logger.Logger
}

// NewUserHandler creates a new user handler
func NewUserHandler(userService *services.UserService, logger *logger.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:      logger,
	}
}

// CreateUser godoc
// @Summary Create a new user
// @Description Create a new user account
// @Tags users
// @Accept json
// @Produce json
// @Param request body ports.CreateUserRequest true "User data"
// @Success 201 {object} entities.User
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Security BearerAuth
// @Router /users [post]
func (h *UserHandler) CreateUser(c echo.Context) error {
	var req ports.CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	user, err := h.userService.CreateUser(c.Request().Context(), req)
	if err != nil {
		h.logger.Error("Create user failed", "error", err)
		if strings.Contains(err.Error(), "already exists") {
			return echo.NewHTTPError(http.StatusConflict, err.Error())
		}
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusCreated, user)
}

// GetMe godoc
// @Summary Get current user
// @Description Get information about the currently authenticated user
// @Tags users
// @Produce json
// @Success 200 {object} entities.User
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /users/me [get]
func (h *UserHandler) GetMe(c echo.Context) error {
	userID := getUserIDFromContext(c)

	user, err := h.userService.GetUser(c.Request().Context(), userID)
	if err != nil {
		h.logger.Error("Get current user failed", "error", err, "user_id", userID)
		return echo.NewHTTPError(http.StatusNotFound, "User not found")
	}

	return c.JSON(http.StatusOK, user)
}

// UpdateMe godoc
// @Summary Update current user
// @Description Update the currently authenticated user's profile
// @Tags users
// @Accept json
// @Produce json
// @Param request body ports.UpdateUserRequest true "User update data"
// @Success 200 {object} entities.User
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Security BearerAuth
// @Router /users/me [put]
func (h *UserHandler) UpdateMe(c echo.Context) error {
	userID := getUserIDFromContext(c)

	var req ports.UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Users cannot change their own role
	req.Role = nil

	user, err := h.userService.UpdateUser(c.Request().Context(), userID, req)
	if err != nil {
		h.logger.Error("Update current user failed", "error", err, "user_id", userID)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, user)
}

// GetUser godoc
// @Summary Get user by ID
// @Description Get user information by ID (admin/manager only)
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} entities.User
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /users/{id} [get]
func (h *UserHandler) GetUser(c echo.Context) error {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid user ID")
	}

	user, err := h.userService.GetUser(c.Request().Context(), userID)
	if err != nil {
		h.logger.Error("Get user failed", "error", err, "user_id", userID)
		return echo.NewHTTPError(http.StatusNotFound, "User not found")
	}

	return c.JSON(http.StatusOK, user)
}

// UpdateUser godoc
// @Summary Update user
// @Description Update user information (admin/manager only)
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body ports.UpdateUserRequest true "User update data"
// @Success 200 {object} entities.User
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /users/{id} [put]
func (h *UserHandler) UpdateUser(c echo.Context) error {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid user ID")
	}

	var req ports.UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	user, err := h.userService.UpdateUser(c.Request().Context(), userID, req)
	if err != nil {
		h.logger.Error("Update user failed", "error", err, "user_id", userID)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, user)
}

// DeleteUser godoc
// @Summary Delete user
// @Description Delete user (admin only)
// @Tags users
// @Param id path string true "User ID"
// @Success 204 "User deleted successfully"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /users/{id} [delete]
func (h *UserHandler) DeleteUser(c echo.Context) error {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid user ID")
	}

	err = h.userService.DeleteUser(c.Request().Context(), userID)
	if err != nil {
		h.logger.Error("Delete user failed", "error", err, "user_id", userID)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

// ListUsers godoc
// @Summary List users
// @Description Get list of users with optional filtering
// @Tags users
// @Produce json
// @Param role query string false "Filter by role"
// @Param is_active query bool false "Filter by active status"
// @Param search query string false "Search in username, email, first name, last name"
// @Param limit query int false "Number of items to return" default(20)
// @Param offset query int false "Number of items to skip" default(0)
// @Param sort_by query string false "Sort field" default(created_at)
// @Param sort_order query string false "Sort order (asc/desc)" default(desc)
// @Success 200 {object} PaginatedResponse[entities.User]
// @Failure 400 {object} ErrorResponse
// @Security BearerAuth
// @Router /users [get]
func (h *UserHandler) ListUsers(c echo.Context) error {
	filter := ports.UserFilter{}

	// Parse query parameters
	if role := c.QueryParam("role"); role != "" {
		userRole := entities.UserRole(role)
		filter.Role = &userRole
	}

	if isActiveStr := c.QueryParam("is_active"); isActiveStr != "" {
		isActive, err := strconv.ParseBool(isActiveStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid is_active parameter")
		}
		filter.IsActive = &isActive
	}

	if search := c.QueryParam("search"); search != "" {
		filter.Search = &search
	}

	if limitStr := c.QueryParam("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid limit parameter")
		}
		filter.Limit = limit
	} else {
		filter.Limit = 20
	}

	if offsetStr := c.QueryParam("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid offset parameter")
		}
		filter.Offset = offset
	}

	filter.SortBy = c.QueryParam("sort_by")
	if filter.SortBy == "" {
		filter.SortBy = "created_at"
	}

	filter.SortOrder = c.QueryParam("sort_order")
	if filter.SortOrder == "" {
		filter.SortOrder = "desc"
	}

	users, total, err := h.userService.ListUsers(c.Request().Context(), filter)
	if err != nil {
		h.logger.Error("List users failed", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve users")
	}

	response := PaginatedResponse[*entities.User]{
		Data:   users,
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}

	return c.JSON(http.StatusOK, response)
}

// Utility functions

func getUserIDFromContext(c echo.Context) uuid.UUID {
	user := c.Get("user")
	if user == nil {
		return uuid.Nil
	}
	
	if userStr, ok := user.(string); ok {
		if id, err := uuid.Parse(userStr); err == nil {
			return id
		}
	}
	
	return uuid.Nil
}

// TaskHandler handles task-related requests
type TaskHandler struct {
	taskService *services.TaskService
	logger      *logger.Logger
}

// NewTaskHandler creates a new task handler
func NewTaskHandler(taskService *services.TaskService, logger *logger.Logger) *TaskHandler {
	return &TaskHandler{
		taskService: taskService,
		logger:      logger,
	}
}

// Create godoc
// @Summary Create a new task
// @Description Create a new task in a project
// @Tags tasks
// @Accept json
// @Produce json
// @Param request body ports.CreateTaskRequest true "Task data"
// @Success 201 {object} entities.Task
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks [post]
func (h *TaskHandler) Create(c echo.Context) error {
	userID := getUserIDFromContext(c)

	var req ports.CreateTaskRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	task, err := h.taskService.CreateTask(c.Request().Context(), req)
	if err != nil {
		h.logger.Error("Create task failed", "error", err, "user_id", userID)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusCreated, task)
}

// GetByID godoc
// @Summary Get task by ID
// @Description Get task details by ID
// @Tags tasks
// @Produce json
// @Param id path int true "Task ID"
// @Success 200 {object} entities.Task
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id} [get]
func (h *TaskHandler) GetByID(c echo.Context) error {
	taskIDStr := c.Param("id")
	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid task ID")
	}

	task, err := h.taskService.GetTask(c.Request().Context(), taskID)
	if err != nil {
		h.logger.Error("Get task failed", "error", err, "task_id", taskID)
		return echo.NewHTTPError(http.StatusNotFound, "Task not found")
	}

	return c.JSON(http.StatusOK, task)
}

// Update godoc
// @Summary Update task
// @Description Update task information
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path int true "Task ID"
// @Param request body ports.UpdateTaskRequest true "Task update data"
// @Success 200 {object} entities.Task
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id} [put]
func (h *TaskHandler) Update(c echo.Context) error {
	taskIDStr := c.Param("id")
	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid task ID")
	}

	var req ports.UpdateTaskRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	task, err := h.taskService.UpdateTask(c.Request().Context(), taskID, req)
	if err != nil {
		h.logger.Error("Update task failed", "error", err, "task_id", taskID)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, task)
}

// Delete godoc
// @Summary Delete task
// @Description Delete task by ID
// @Tags tasks
// @Param id path int true "Task ID"
// @Success 204 "Task deleted successfully"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id} [delete]
func (h *TaskHandler) Delete(c echo.Context) error {
	taskIDStr := c.Param("id")
	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid task ID")
	}

	err = h.taskService.DeleteTask(c.Request().Context(), taskID)
	if err != nil {
		h.logger.Error("Delete task failed", "error", err, "task_id", taskID)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

// List godoc
// @Summary List tasks
// @Description Get list of tasks with optional filtering
// @Tags tasks
// @Produce json
// @Param status query string false "Filter by status"
// @Param priority query string false "Filter by priority"
// @Param project_id query int false "Filter by project"
// @Param assignee_id query string false "Filter by assignee"
// @Param search query string false "Search in title and description"
// @Param limit query int false "Number of items to return" default(20)
// @Param offset query int false "Number of items to skip" default(0)
// @Success 200 {object} PaginatedResponse[entities.Task]
// @Failure 400 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks [get]
func (h *TaskHandler) List(c echo.Context) error {
	filter := ports.TaskFilter{}

	// Parse query parameters
	if status := c.QueryParam("status"); status != "" {
		taskStatus := entities.TaskStatus(status)
		filter.Status = &taskStatus
	}

	if priority := c.QueryParam("priority"); priority != "" {
		taskPriority := entities.TaskPriority(priority)
		filter.Priority = &taskPriority
	}

	if projectIDStr := c.QueryParam("project_id"); projectIDStr != "" {
		projectID, err := strconv.Atoi(projectIDStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid project_id parameter")
		}
		filter.ProjectID = &projectID
	}

	if assigneeIDStr := c.QueryParam("assignee_id"); assigneeIDStr != "" {
		assigneeID, err := uuid.Parse(assigneeIDStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid assignee_id parameter")
		}
		filter.AssigneeID = &assigneeID
	}

	if search := c.QueryParam("search"); search != "" {
		filter.Search = &search
	}

	if limitStr := c.QueryParam("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid limit parameter")
		}
		filter.Limit = limit
	} else {
		filter.Limit = 20
	}

	if offsetStr := c.QueryParam("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid offset parameter")
		}
		filter.Offset = offset
	}

	filter.SortBy = c.QueryParam("sort_by")
	if filter.SortBy == "" {
		filter.SortBy = "created_at"
	}

	filter.SortOrder = c.QueryParam("sort_order")
	if filter.SortOrder == "" {
		filter.SortOrder = "desc"
	}

	tasks, total, err := h.taskService.ListTasks(c.Request().Context(), filter)
	if err != nil {
		h.logger.Error("List tasks failed", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve tasks")
	}

	response := PaginatedResponse[*entities.Task]{
		Data:   tasks,
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}

	return c.JSON(http.StatusOK, response)
}

// UpdateStatus godoc
// @Summary Update task status
// @Description Update task status
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path int true "Task ID"
// @Param request body map[string]string true "Status"
// @Success 200 {object} entities.Task
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id}/status [patch]
func (h *TaskHandler) UpdateStatus(c echo.Context) error {
	taskIDStr := c.Param("id")
	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid task ID")
	}

	var req map[string]string
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	statusStr, ok := req["status"]
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "status is required")
	}

	status := entities.TaskStatus(statusStr)
	task, err := h.taskService.UpdateTaskStatus(c.Request().Context(), taskID, status)
	if err != nil {
		h.logger.Error("Update task status failed", "error", err, "task_id", taskID, "status", status)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, task)
}

// AssignUser godoc
// @Summary Assign task to user
// @Description Assign a task to a specific user
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path int true "Task ID"
// @Param request body map[string]string true "Assignee ID"
// @Success 200 {object} entities.Task
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id}/assign [post]
func (h *TaskHandler) AssignUser(c echo.Context) error {
	taskIDStr := c.Param("id")
	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid task ID")
	}

	var req map[string]string
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	assigneeIDStr, ok := req["assignee_id"]
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "assignee_id is required")
	}

	assigneeID, err := uuid.Parse(assigneeIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid assignee_id")
	}

	task, err := h.taskService.AssignTask(c.Request().Context(), taskID, assigneeID)
	if err != nil {
		h.logger.Error("Assign task failed", "error", err, "task_id", taskID, "assignee_id", assigneeID)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, task)
}

// UnassignUser godoc
// @Summary Unassign task
// @Description Remove assignment from a task
// @Tags tasks
// @Param id path int true "Task ID"
// @Success 200 {object} entities.Task
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks/{id}/assign [delete]
func (h *TaskHandler) UnassignUser(c echo.Context) error {
	taskIDStr := c.Param("id")
	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid task ID")
	}

	// Get task and set assignee to nil
	task, err := h.taskService.GetTask(c.Request().Context(), taskID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Task not found")
	}

	req := ports.UpdateTaskRequest{
		AssigneeID: nil,
	}

	updatedTask, err := h.taskService.UpdateTask(c.Request().Context(), taskID, req)
	if err != nil {
		h.logger.Error("Unassign task failed", "error", err, "task_id", taskID)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, updatedTask)
}

// GetDeadlines godoc
// @Summary Get upcoming deadlines
// @Description Get tasks with approaching deadlines
// @Tags tasks
// @Produce json
// @Param days query int false "Number of days to look ahead" default(7)
// @Success 200 {object} DeadlinesResponse
// @Failure 400 {object} ErrorResponse
// @Security BearerAuth
// @Router /tasks/deadlines [get]
func (h *TaskHandler) GetDeadlines(c echo.Context) error {
	daysStr := c.QueryParam("days")
	days := 7 // default

	if daysStr != "" {
		var err error
		days, err = strconv.Atoi(daysStr)
		if err != nil || days < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid days parameter")
		}
	}

	nearDeadline, err := h.taskService.GetTasksNearDeadline(c.Request().Context(), days)
	if err != nil {
		h.logger.Error("Get tasks near deadline failed", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve deadlines")
	}

	overdue, err := h.taskService.GetOverdueTasks(c.Request().Context())
	if err != nil {
		h.logger.Error("Get overdue tasks failed", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve overdue tasks")
	}

	response := DeadlinesResponse{
		NearDeadline: nearDeadline,
		Overdue:      overdue,
		Days:         days,
	}

	return c.JSON(http.StatusOK, response)
}

func parseTime(timeStr string) (*time.Time, error) {
	if timeStr == "" {
		return nil, nil
	}
	
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return nil, err
	}
	
	return &t, nil
}
