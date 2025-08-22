package http

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/taskmaster/core/internal/application/services"
	"github.com/taskmaster/core/internal/domain/entities"
	"github.com/taskmaster/core/internal/infrastructure/logger"
	"github.com/taskmaster/core/internal/ports"
)

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

// Login handles user login
func (h *AuthHandler) Login(c echo.Context) error {
	var req services.LoginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	response, err := h.authService.Login(c.Request().Context(), req)
	if err != nil {
		h.logger.Error("Login failed", "error", err, "email", req.Email)
		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid credentials")
	}

	return c.JSON(http.StatusOK, response)
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(c echo.Context) error {
	var req RefreshTokenRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	response, err := h.authService.RefreshToken(c.Request().Context(), req.RefreshToken)
	if err != nil {
		h.logger.Error("Token refresh failed", "error", err)
		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid refresh token")
	}

	return c.JSON(http.StatusOK, response)
}

// Logout handles user logout
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

// CreateUser handles user creation
func (h *UserHandler) CreateUser(c echo.Context) error {
	var req services.CreateUserRequest
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

// GetCurrentUser handles getting current user info
func (h *UserHandler) GetCurrentUser(c echo.Context) error {
	userID := getUserIDFromContext(c)

	user, err := h.userService.GetUser(c.Request().Context(), userID)
	if err != nil {
		h.logger.Error("Get current user failed", "error", err, "user_id", userID)
		return echo.NewHTTPError(http.StatusNotFound, "User not found")
	}

	return c.JSON(http.StatusOK, user)
}

// GetUser handles getting user by ID
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

// UpdateCurrentUser handles updating current user
func (h *UserHandler) UpdateCurrentUser(c echo.Context) error {
	userID := getUserIDFromContext(c)

	var req services.UpdateUserRequest
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

// ListUsers handles listing users
func (h *UserHandler) ListUsers(c echo.Context) error {
	filter := ports.UserFilter{}

	if role := c.QueryParam("role"); role != "" {
		userRole := entities.UserRole(role)
		filter.Role = &userRole
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

// Placeholder methods - will implement later
func (h *TaskHandler) CreateTask(c echo.Context) error {
	return echo.NewHTTPError(http.StatusNotImplemented, "Not implemented yet")
}

func (h *TaskHandler) GetTask(c echo.Context) error {
	return echo.NewHTTPError(http.StatusNotImplemented, "Not implemented yet")
}

func (h *TaskHandler) UpdateTask(c echo.Context) error {
	return echo.NewHTTPError(http.StatusNotImplemented, "Not implemented yet")
}

func (h *TaskHandler) AssignTask(c echo.Context) error {
	return echo.NewHTTPError(http.StatusNotImplemented, "Not implemented yet")
}

func (h *TaskHandler) StartTask(c echo.Context) error {
	return echo.NewHTTPError(http.StatusNotImplemented, "Not implemented yet")
}

func (h *TaskHandler) CompleteTask(c echo.Context) error {
	return echo.NewHTTPError(http.StatusNotImplemented, "Not implemented yet")
}

func (h *TaskHandler) ListTasks(c echo.Context) error {
	return echo.NewHTTPError(http.StatusNotImplemented, "Not implemented yet")
}

func (h *TaskHandler) GetDeadlines(c echo.Context) error {
	return echo.NewHTTPError(http.StatusNotImplemented, "Not implemented yet")
}

// Utility functions and helper types

func getUserIDFromContext(c echo.Context) uuid.UUID {
	user := c.Get("user")
	if user == nil {
		return uuid.Nil
	}
	
	if userStr, ok := user.(string); ok {
		userID, _ := uuid.Parse(userStr)
		return userID
	}
	
	return uuid.Nil
}

// Request/Response types
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

type PaginatedResponse[T any] struct {
	Data   []T   `json:"data"`
	Total  int64 `json:"total"`
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
}
