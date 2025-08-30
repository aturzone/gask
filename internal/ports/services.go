package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/taskmaster/core/internal/domain/entities"
)

// AuthService interface for authentication operations
type AuthService interface {
	Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error)
	Login(ctx context.Context, req LoginRequest) (*AuthResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*AuthResponse, error)
	Logout(ctx context.Context, userID uuid.UUID) error
	ValidateToken(tokenString string) (*Claims, error)
}

// UserService interface for user management operations
type UserService interface {
	CreateUser(ctx context.Context, req CreateUserRequest) (*entities.User, error)
	GetUser(ctx context.Context, id uuid.UUID) (*entities.User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, req UpdateUserRequest) (*entities.User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
	ListUsers(ctx context.Context, filter UserFilter) ([]*entities.User, int, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error
	GetUserProfile(ctx context.Context, userID uuid.UUID) (*entities.User, error)
}

// ProjectService interface for project management operations
type ProjectService interface {
	CreateProject(ctx context.Context, req CreateProjectRequest) (*entities.Project, error)
	GetProject(ctx context.Context, id int) (*entities.Project, error)
	UpdateProject(ctx context.Context, id int, req UpdateProjectRequest) (*entities.Project, error)
	DeleteProject(ctx context.Context, id int) error
	ListProjects(ctx context.Context, filter ProjectFilter) ([]*entities.Project, int, error)
	GetUserProjects(ctx context.Context, userID uuid.UUID) ([]*entities.Project, error)
}

// TaskService interface for task management operations
type TaskService interface {
	CreateTask(ctx context.Context, req CreateTaskRequest) (*entities.Task, error)
	GetTask(ctx context.Context, id int) (*entities.Task, error)
	UpdateTask(ctx context.Context, id int, req UpdateTaskRequest) (*entities.Task, error)
	DeleteTask(ctx context.Context, id int) error
	ListTasks(ctx context.Context, filter TaskFilter) ([]*entities.Task, int, error)
	AssignTask(ctx context.Context, taskID int, assigneeID uuid.UUID) (*entities.Task, error)
	UpdateTaskStatus(ctx context.Context, taskID int, status entities.TaskStatus) (*entities.Task, error)
	GetTasksNearDeadline(ctx context.Context, days int) ([]*entities.Task, error)
	GetOverdueTasks(ctx context.Context) ([]*entities.Task, error)
	GetUserTasks(ctx context.Context, userID uuid.UUID) ([]*entities.Task, error)
}

// TimeService interface for time tracking operations
type TimeService interface {
	CreateTimeEntry(ctx context.Context, req CreateTimeEntryRequest) (*entities.TimeEntry, error)
	GetTimeEntry(ctx context.Context, id uuid.UUID) (*entities.TimeEntry, error)
	UpdateTimeEntry(ctx context.Context, id uuid.UUID, req UpdateTimeEntryRequest) (*entities.TimeEntry, error)
	DeleteTimeEntry(ctx context.Context, id uuid.UUID) error
	ListTimeEntries(ctx context.Context, filter TimeEntryFilter) ([]*entities.TimeEntry, int, error)
	StartTimeTracking(ctx context.Context, userID uuid.UUID, taskID int) (*entities.TimeEntry, error)
	StopTimeTracking(ctx context.Context, userID uuid.UUID) (*entities.TimeEntry, error)
	GetActiveTimeEntry(ctx context.Context, userID uuid.UUID) (*entities.TimeEntry, error)
	GetTimeReport(ctx context.Context, req TimeReportRequest) (*TimeReport, error)
}

// Request/Response Types

// Auth related types
type RegisterRequest struct {
	Email     string               `json:"email" validate:"required,email"`
	Username  string               `json:"username" validate:"required,min=3,max=50"`
	Password  string               `json:"password" validate:"required,min=8"`
	FirstName *string              `json:"first_name" validate:"omitempty,max=100"`
	LastName  *string              `json:"last_name" validate:"omitempty,max=100"`
	Role      entities.UserRole    `json:"role" validate:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type AuthResponse struct {
	AccessToken  string         `json:"access_token"`
	RefreshToken string         `json:"refresh_token"`
	TokenType    string         `json:"token_type"`
	ExpiresIn    int64          `json:"expires_in"`
	User         *entities.User `json:"user"`
}

type Claims struct {
	UserID string            `json:"user_id"`
	Email  string            `json:"email"`
	Role   entities.UserRole `json:"role"`
}

// User related types
type CreateUserRequest struct {
	Email             string                `json:"email" validate:"required,email"`
	Username          string                `json:"username" validate:"required,min=3,max=50"`
	Password          string                `json:"password" validate:"required,min=8"`
	FirstName         *string               `json:"first_name" validate:"omitempty,max=100"`
	LastName          *string               `json:"last_name" validate:"omitempty,max=100"`
	Role              entities.UserRole     `json:"role" validate:"required"`
	IsActive          bool                  `json:"is_active"`
}

type UpdateUserRequest struct {
	Email     *string            `json:"email" validate:"omitempty,email"`
	Username  *string            `json:"username" validate:"omitempty,min=3,max=50"`
	FirstName *string            `json:"first_name" validate:"omitempty,max=100"`
	LastName  *string            `json:"last_name" validate:"omitempty,max=100"`
	Role      *entities.UserRole `json:"role" validate:"omitempty"`
	IsActive  *bool              `json:"is_active"`
}

// Project related types
type CreateProjectRequest struct {
	Name        string                 `json:"name" validate:"required,max=200"`
	Description *string                `json:"description" validate:"omitempty,max=1000"`
	Status      entities.ProjectStatus `json:"status" validate:"required"`
	StartDate   *time.Time             `json:"start_date"`
	EndDate     *time.Time             `json:"end_date"`
	ManagerID   uuid.UUID              `json:"manager_id" validate:"required"`
}

type UpdateProjectRequest struct {
	Name        *string                 `json:"name" validate:"omitempty,max=200"`
	Description *string                 `json:"description" validate:"omitempty,max=1000"`
	Status      *entities.ProjectStatus `json:"status" validate:"omitempty"`
	StartDate   *time.Time              `json:"start_date"`
	EndDate     *time.Time              `json:"end_date"`
	ManagerID   *uuid.UUID              `json:"manager_id"`
}

// Task related types
type CreateTaskRequest struct {
	Title       string                `json:"title" validate:"required,max=500"`
	Description *string               `json:"description" validate:"omitempty,max=2000"`
	Status      entities.TaskStatus   `json:"status" validate:"required"`
	Priority    entities.TaskPriority `json:"priority" validate:"required"`
	ProjectID   int                   `json:"project_id" validate:"required"`
	AssigneeID  *uuid.UUID            `json:"assignee_id"`
	DueDate     *time.Time            `json:"due_date"`
}

type UpdateTaskRequest struct {
	Title       *string                `json:"title" validate:"omitempty,max=500"`
	Description *string                `json:"description" validate:"omitempty,max=2000"`
	Status      *entities.TaskStatus   `json:"status" validate:"omitempty"`
	Priority    *entities.TaskPriority `json:"priority" validate:"omitempty"`
	AssigneeID  *uuid.UUID             `json:"assignee_id"`
	DueDate     *time.Time             `json:"due_date"`
}

// Time tracking related types
type CreateTimeEntryRequest struct {
	TaskID      int        `json:"task_id" validate:"required"`
	Description *string    `json:"description" validate:"omitempty,max=1000"`
	Hours       *float64   `json:"hours" validate:"omitempty,min=0"`
	StartTime   time.Time  `json:"start_time" validate:"required"`
	EndTime     *time.Time `json:"end_time"`
}

type UpdateTimeEntryRequest struct {
	Description *string    `json:"description" validate:"omitempty,max=1000"`
	Hours       *float64   `json:"hours" validate:"omitempty,min=0"`
	StartTime   *time.Time `json:"start_time"`
	EndTime     *time.Time `json:"end_time"`
}

type TimeReportRequest struct {
	UserID    *uuid.UUID `json:"user_id"`
	ProjectID *int       `json:"project_id"`
	StartDate time.Time  `json:"start_date" validate:"required"`
	EndDate   time.Time  `json:"end_date" validate:"required"`
	GroupBy   string     `json:"group_by" validate:"omitempty,oneof=user project task day week month"`
}

type TimeReport struct {
	TotalHours float64                `json:"total_hours"`
	Entries    []TimeReportEntry      `json:"entries"`
	Summary    map[string]interface{} `json:"summary"`
}

type TimeReportEntry struct {
	Label      string  `json:"label"`
	Hours      float64 `json:"hours"`
	Percentage float64 `json:"percentage"`
}

// Response types for pagination and common structures
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
