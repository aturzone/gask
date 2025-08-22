package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/taskmaster/core/internal/domain/entities"
	"github.com/taskmaster/core/internal/infrastructure/config"
	"github.com/taskmaster/core/internal/infrastructure/logger"
	"github.com/taskmaster/core/internal/ports"
)

// AuthService handles authentication and authorization
type AuthService struct {
	userRepo     ports.UserRepository
	authRepo     ports.AuthRepository
	jwtConfig    config.JWTConfig
	logger       *logger.Logger
}

// NewAuthService creates a new authentication service
func NewAuthService(
	userRepo ports.UserRepository,
	authRepo ports.AuthRepository,
	jwtConfig config.JWTConfig,
	logger *logger.Logger,
) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		authRepo:  authRepo,
		jwtConfig: jwtConfig,
		logger:    logger,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	AccessToken  string                `json:"access_token"`
	RefreshToken string                `json:"refresh_token"`
	ExpiresIn    int64                 `json:"expires_in"`
	User         *entities.User        `json:"user"`
}

// Claims represents JWT claims
type Claims struct {
	UserID   string            `json:"user_id"`
	Email    string            `json:"email"`
	Role     entities.UserRole `json:"role"`
	jwt.RegisteredClaims
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		s.logger.LogSecurityEvent("login_failed", "", "", map[string]interface{}{
			"email": req.Email,
			"reason": "user_not_found",
		})
		return nil, fmt.Errorf("invalid credentials")
	}

	if !user.IsActive {
		s.logger.LogSecurityEvent("login_failed", user.ID.String(), "", map[string]interface{}{
			"email": req.Email,
			"reason": "user_inactive",
		})
		return nil, fmt.Errorf("account is inactive")
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		s.logger.LogSecurityEvent("login_failed", user.ID.String(), "", map[string]interface{}{
			"email": req.Email,
			"reason": "invalid_password",
		})
		return nil, fmt.Errorf("invalid credentials")
	}

	// Generate tokens
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	s.logger.LogUserAction(user.ID.String(), "login", map[string]interface{}{
		"email": user.Email,
	})

	// Remove password from response
	user.PasswordHash = ""

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.jwtConfig.ExpiresIn.Seconds()),
		User:         user,
	}, nil
}

// RefreshToken refreshes an access token using a refresh token
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	// Hash the refresh token to compare with stored hash
	hasher := sha256.New()
	hasher.Write([]byte(refreshToken))
	tokenHash := hex.EncodeToString(hasher.Sum(nil))

	// Get refresh token from database
	storedToken, err := s.authRepo.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	if !storedToken.IsValid() {
		return nil, fmt.Errorf("refresh token expired or revoked")
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, storedToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	if !user.IsActive {
		return nil, fmt.Errorf("account is inactive")
	}

	// Generate new tokens
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, err := s.generateRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Revoke old refresh token
	err = s.authRepo.RevokeRefreshToken(ctx, tokenHash)
	if err != nil {
		s.logger.Error("Failed to revoke old refresh token", "error", err)
	}

	s.logger.LogUserAction(user.ID.String(), "token_refresh", nil)

	// Remove password from response
	user.PasswordHash = ""

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(s.jwtConfig.ExpiresIn.Seconds()),
		User:         user,
	}, nil
}

// Logout revokes all user tokens
func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID) error {
	err := s.authRepo.RevokeAllUserTokens(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke user tokens: %w", err)
	}

	s.logger.LogUserAction(userID.String(), "logout", nil)
	return nil
}

// ValidateToken validates and parses a JWT token
func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtConfig.Secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

func (s *AuthService) generateAccessToken(user *entities.User) (string, error) {
	claims := &Claims{
		UserID: user.ID.String(),
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.jwtConfig.ExpiresIn)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    s.jwtConfig.Issuer,
			Subject:   user.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtConfig.Secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

func (s *AuthService) generateRefreshToken(ctx context.Context, userID uuid.UUID) (string, error) {
	// Generate random token
	tokenBytes := make([]byte, 32)
	_, err := rand.Read(tokenBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}

	token := hex.EncodeToString(tokenBytes)

	// Hash token for storage
	hasher := sha256.New()
	hasher.Write([]byte(token))
	tokenHash := hex.EncodeToString(hasher.Sum(nil))

	// Store in database
	expiresAt := time.Now().Add(s.jwtConfig.RefreshExpiresIn)
	err = s.authRepo.CreateRefreshToken(ctx, userID, tokenHash, expiresAt)
	if err != nil {
		return "", fmt.Errorf("failed to store refresh token: %w", err)
	}

	return token, nil
}

// UserService handles user-related operations
type UserService struct {
	userRepo ports.UserRepository
	logger   *logger.Logger
}

// NewUserService creates a new user service
func NewUserService(userRepo ports.UserRepository, logger *logger.Logger) *UserService {
	return &UserService{
		userRepo: userRepo,
		logger:   logger,
	}
}

// CreateUserRequest represents a user creation request
type CreateUserRequest struct {
	Email             string                `json:"email" validate:"required,email"`
	Username          string                `json:"username" validate:"required,min=3,max=50"`
	Password          string                `json:"password" validate:"required,min=8"`
	FirstName         *string               `json:"first_name" validate:"omitempty,max=100"`
	LastName          *string               `json:"last_name" validate:"omitempty,max=100"`
	Role              entities.UserRole     `json:"role" validate:"required"`
	WorkingHoursStart *time.Time            `json:"working_hours_start"`
	WorkingHoursEnd   *time.Time            `json:"working_hours_end"`
	WorkingDays       []int                 `json:"working_days"`
	Timezone          string                `json:"timezone" validate:"required"`
	HourlyRate        *float64              `json:"hourly_rate" validate:"omitempty,min=0"`
}

// UpdateUserRequest represents a user update request
type UpdateUserRequest struct {
	Email             *string           `json:"email" validate:"omitempty,email"`
	Username          *string           `json:"username" validate:"omitempty,min=3,max=50"`
	FirstName         *string           `json:"first_name" validate:"omitempty,max=100"`
	LastName          *string           `json:"last_name" validate:"omitempty,max=100"`
	WorkingHoursStart *time.Time        `json:"working_hours_start"`
	WorkingHoursEnd   *time.Time        `json:"working_hours_end"`
	WorkingDays       []int             `json:"working_days"`
	Timezone          *string           `json:"timezone"`
	HourlyRate        *float64          `json:"hourly_rate" validate:"omitempty,min=0"`
}

// CreateUser creates a new user
func (s *UserService) CreateUser(ctx context.Context, req CreateUserRequest) (*entities.User, error) {
	// Check if user already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("user with email %s already exists", req.Email)
	}

	existingUser, err = s.userRepo.GetByUsername(ctx, req.Username)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("user with username %s already exists", req.Username)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user entity
	user := &entities.User{
		ID:                uuid.New(),
		Email:             req.Email,
		Username:          req.Username,
		PasswordHash:      string(hashedPassword),
		FirstName:         req.FirstName,
		LastName:          req.LastName,
		Role:              req.Role,
		IsActive:          true,
		WorkingHoursStart: req.WorkingHoursStart,
		WorkingHoursEnd:   req.WorkingHoursEnd,
		WorkingDays:       req.WorkingDays,
		Timezone:          req.Timezone,
		HourlyRate:        req.HourlyRate,
	}

	// Save user
	err = s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.LogUserAction(user.ID.String(), "user_created", map[string]interface{}{
		"email": user.Email,
		"role":  user.Role,
	})

	// Remove password hash from response
	user.PasswordHash = ""

	return user, nil
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(ctx context.Context, userID uuid.UUID) (*entities.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Remove password hash from response
	user.PasswordHash = ""

	return user, nil
}

// UpdateUser updates a user
func (s *UserService) UpdateUser(ctx context.Context, userID uuid.UUID, req UpdateUserRequest) (*entities.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Update fields if provided
	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.Username != nil {
		user.Username = *req.Username
	}
	if req.FirstName != nil {
		user.FirstName = req.FirstName
	}
	if req.LastName != nil {
		user.LastName = req.LastName
	}
	if req.WorkingHoursStart != nil {
		user.WorkingHoursStart = req.WorkingHoursStart
	}
	if req.WorkingHoursEnd != nil {
		user.WorkingHoursEnd = req.WorkingHoursEnd
	}
	if req.WorkingDays != nil {
		user.WorkingDays = req.WorkingDays
	}
	if req.Timezone != nil {
		user.Timezone = *req.Timezone
	}
	if req.HourlyRate != nil {
		user.HourlyRate = req.HourlyRate
	}

	err = s.userRepo.Update(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	s.logger.LogUserAction(userID.String(), "user_updated", nil)

	// Remove password hash from response
	user.PasswordHash = ""

	return user, nil
}

// ListUsers lists users with filtering
func (s *UserService) ListUsers(ctx context.Context, filter ports.UserFilter) ([]*entities.User, int64, error) {
	users, err := s.userRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	count, err := s.userRepo.Count(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Remove password hashes from response
	for _, user := range users {
		user.PasswordHash = ""
	}

	return users, count, nil
}

// TaskService handles task-related operations
type TaskService struct {
	taskRepo    ports.TaskRepository
	projectRepo ports.ProjectRepository
	userRepo    ports.UserRepository
	logger      *logger.Logger
}

// NewTaskService creates a new task service
func NewTaskService(
	taskRepo ports.TaskRepository,
	projectRepo ports.ProjectRepository,
	userRepo ports.UserRepository,
	logger *logger.Logger,
) *TaskService {
	return &TaskService{
		taskRepo:    taskRepo,
		projectRepo: projectRepo,
		userRepo:    userRepo,
		logger:      logger,
	}
}

// CreateTaskRequest represents a task creation request
type CreateTaskRequest struct {
	ProjectID      int                   `json:"project_id" validate:"required"`
	ParentTaskID   *int                  `json:"parent_task_id"`
	Title          string                `json:"title" validate:"required,min=3,max=255"`
	Description    *string               `json:"description" validate:"omitempty,max=2000"`
	Priority       entities.Priority     `json:"priority" validate:"required"`
	AssigneeID     *uuid.UUID            `json:"assignee_id"`
	EstimatedHours *float64              `json:"estimated_hours" validate:"omitempty,min=0"`
	StartDate      *time.Time            `json:"start_date"`
	DueDate        *time.Time            `json:"due_date"`
	Tags           []string              `json:"tags"`
}

// UpdateTaskRequest represents a task update request
type UpdateTaskRequest struct {
	Title          *string           `json:"title" validate:"omitempty,min=3,max=255"`
	Description    *string           `json:"description" validate:"omitempty,max=2000"`
	Status         *entities.TaskStatus `json:"status"`
	Priority       *entities.Priority `json:"priority"`
	AssigneeID     *uuid.UUID        `json:"assignee_id"`
	EstimatedHours *float64          `json:"estimated_hours" validate:"omitempty,min=0"`
	StartDate      *time.Time        `json:"start_date"`
	DueDate        *time.Time        `json:"due_date"`
	Tags           []string          `json:"tags"`
}

// CreateTask creates a new task
func (s *TaskService) CreateTask(ctx context.Context, req CreateTaskRequest, createdBy uuid.UUID) (*entities.Task, error) {
	// Verify project exists
	project, err := s.projectRepo.GetByID(ctx, req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	if !project.CanAddTask() {
		return nil, fmt.Errorf("cannot add tasks to project in status: %s", project.Status)
	}

	// Verify assignee exists if provided
	if req.AssigneeID != nil {
		_, err := s.userRepo.GetByID(ctx, *req.AssigneeID)
		if err != nil {
			return nil, fmt.Errorf("assignee not found: %w", err)
		}
	}

	// Verify parent task exists if provided
	if req.ParentTaskID != nil {
		_, err := s.taskRepo.GetByID(ctx, *req.ParentTaskID)
		if err != nil {
			return nil, fmt.Errorf("parent task not found: %w", err)
		}
	}

	// Create task entity
	task := &entities.Task{
		ProjectID:      req.ProjectID,
		ParentTaskID:   req.ParentTaskID,
		Title:          req.Title,
		Description:    req.Description,
		Status:         entities.TaskStatusTodo,
		Priority:       req.Priority,
		AssigneeID:     req.AssigneeID,
		ReporterID:     &createdBy,
		EstimatedHours: req.EstimatedHours,
		StartDate:      req.StartDate,
		DueDate:        req.DueDate,
		Tags:           req.Tags,
	}

	// Validate due date
	if task.DueDate != nil {
		if err := task.SetDueDate(*task.DueDate); err != nil {
			return nil, fmt.Errorf("invalid due date: %w", err)
		}
	}

	// Save task
	err = s.taskRepo.Create(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	s.logger.LogUserAction(createdBy.String(), "task_created", map[string]interface{}{
		"task_id":    task.ID,
		"project_id": task.ProjectID,
		"title":      task.Title,
	})

	return task, nil
}

// GetTask retrieves a task by ID
func (s *TaskService) GetTask(ctx context.Context, taskID int) (*entities.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	return task, nil
}

// UpdateTask updates a task
func (s *TaskService) UpdateTask(ctx context.Context, taskID int, req UpdateTaskRequest, updatedBy uuid.UUID) (*entities.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// Update fields if provided
	if req.Title != nil {
		task.Title = *req.Title
	}
	if req.Description != nil {
		task.Description = req.Description
	}
	if req.Status != nil {
		task.Status = *req.Status
	}
	if req.Priority != nil {
		task.Priority = *req.Priority
	}
	if req.AssigneeID != nil {
		// Verify assignee exists
		_, err := s.userRepo.GetByID(ctx, *req.AssigneeID)
		if err != nil {
			return nil, fmt.Errorf("assignee not found: %w", err)
		}
		task.AssigneeID = req.AssigneeID
	}
	if req.EstimatedHours != nil {
		task.EstimatedHours = req.EstimatedHours
	}
	if req.StartDate != nil {
		task.StartDate = req.StartDate
	}
	if req.DueDate != nil {
		if err := task.SetDueDate(*req.DueDate); err != nil {
			return nil, fmt.Errorf("invalid due date: %w", err)
		}
	}
	if req.Tags != nil {
		task.Tags = req.Tags
	}

	err = s.taskRepo.Update(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	s.logger.LogUserAction(updatedBy.String(), "task_updated", map[string]interface{}{
		"task_id": task.ID,
	})

	return task, nil
}

// AssignTask assigns a task to a user
func (s *TaskService) AssignTask(ctx context.Context, taskID int, assigneeID uuid.UUID, assignedBy uuid.UUID) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Verify assignee exists
	assignee, err := s.userRepo.GetByID(ctx, assigneeID)
	if err != nil {
		return fmt.Errorf("assignee not found: %w", err)
	}

	if !assignee.IsActive {
		return fmt.Errorf("cannot assign task to inactive user")
	}

	// Assign task
	err = task.AssignTo(assigneeID)
	if err != nil {
		return fmt.Errorf("failed to assign task: %w", err)
	}

	err = s.taskRepo.Update(ctx, task)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	s.logger.LogUserAction(assignedBy.String(), "task_assigned", map[string]interface{}{
		"task_id":     task.ID,
		"assignee_id": assigneeID.String(),
	})

	return nil
}

// StartTask marks a task as in progress
func (s *TaskService) StartTask(ctx context.Context, taskID int, userID uuid.UUID) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	if task.AssigneeID == nil || *task.AssigneeID != userID {
		return fmt.Errorf("task is not assigned to you")
	}

	err = task.Start()
	if err != nil {
		return fmt.Errorf("failed to start task: %w", err)
	}

	err = s.taskRepo.Update(ctx, task)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	s.logger.LogUserAction(userID.String(), "task_started", map[string]interface{}{
		"task_id": task.ID,
	})

	return nil
}

// CompleteTask marks a task as completed
func (s *TaskService) CompleteTask(ctx context.Context, taskID int, actualHours float64, userID uuid.UUID) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	if task.AssigneeID == nil || *task.AssigneeID != userID {
		return fmt.Errorf("task is not assigned to you")
	}

	err = task.Complete(actualHours)
	if err != nil {
		return fmt.Errorf("failed to complete task: %w", err)
	}

	err = s.taskRepo.Update(ctx, task)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	s.logger.LogUserAction(userID.String(), "task_completed", map[string]interface{}{
		"task_id":      task.ID,
		"actual_hours": actualHours,
	})

	return nil
}

// ListTasks lists tasks with filtering
func (s *TaskService) ListTasks(ctx context.Context, filter ports.TaskFilter) ([]*entities.Task, int64, error) {
	tasks, err := s.taskRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tasks: %w", err)
	}

	count, err := s.taskRepo.Count(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	return tasks, count, nil
}

// GetTasksNearDeadline gets tasks approaching their deadline
func (s *TaskService) GetTasksNearDeadline(ctx context.Context, days int) ([]*entities.Task, error) {
	tasks, err := s.taskRepo.GetTasksNearDeadline(ctx, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks near deadline: %w", err)
	}

	return tasks, nil
}

// GetOverdueTasks gets overdue tasks
func (s *TaskService) GetOverdueTasks(ctx context.Context) ([]*entities.Task, error) {
	tasks, err := s.taskRepo.GetOverdueTasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get overdue tasks: %w", err)
	}

	return tasks, nil
}