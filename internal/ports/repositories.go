package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/taskmaster/core/internal/domain/entities"
)

// UserRepository interface for user data operations
type UserRepository interface {
	Create(ctx context.Context, user *entities.User) (*entities.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error)
	GetByEmail(ctx context.Context, email string) (*entities.User, error)
	GetByUsername(ctx context.Context, username string) (*entities.User, error)
	Update(ctx context.Context, user *entities.User) (*entities.User, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter UserFilter) ([]*entities.User, int, error)
	UpdateLastLogin(ctx context.Context, id uuid.UUID, loginTime time.Time) error
}

// ProjectRepository interface for project data operations
type ProjectRepository interface {
	Create(ctx context.Context, project *entities.Project) (*entities.Project, error)
	GetByID(ctx context.Context, id int) (*entities.Project, error)
	Update(ctx context.Context, project *entities.Project) (*entities.Project, error)
	Delete(ctx context.Context, id int) error
	List(ctx context.Context, filter ProjectFilter) ([]*entities.Project, int, error)
	GetUserProjects(ctx context.Context, userID uuid.UUID) ([]*entities.Project, error)
}

// TaskRepository interface for task data operations
type TaskRepository interface {
	Create(ctx context.Context, task *entities.Task) (*entities.Task, error)
	GetByID(ctx context.Context, id int) (*entities.Task, error)
	Update(ctx context.Context, task *entities.Task) (*entities.Task, error)
	Delete(ctx context.Context, id int) error
	List(ctx context.Context, filter TaskFilter) ([]*entities.Task, int, error)
	GetByProject(ctx context.Context, projectID int) ([]*entities.Task, error)
	GetByAssignee(ctx context.Context, userID uuid.UUID) ([]*entities.Task, error)
	GetOverdue(ctx context.Context) ([]*entities.Task, error)
	GetNearDeadline(ctx context.Context, days int) ([]*entities.Task, error)
	UpdateStatus(ctx context.Context, id int, status entities.TaskStatus) error
	Assign(ctx context.Context, id int, assigneeID uuid.UUID) error
}

// TimeEntryRepository interface for time tracking operations
type TimeEntryRepository interface {
	Create(ctx context.Context, entry *entities.TimeEntry) (*entities.TimeEntry, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entities.TimeEntry, error)
	Update(ctx context.Context, entry *entities.TimeEntry) (*entities.TimeEntry, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter TimeEntryFilter) ([]*entities.TimeEntry, int, error)
	GetByUser(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]*entities.TimeEntry, error)
	GetByTask(ctx context.Context, taskID int) ([]*entities.TimeEntry, error)
	GetByProject(ctx context.Context, projectID int, from, to time.Time) ([]*entities.TimeEntry, error)
	GetActiveEntry(ctx context.Context, userID uuid.UUID) (*entities.TimeEntry, error)
	GetTotalHours(ctx context.Context, userID uuid.UUID, from, to time.Time) (float64, error)
}

// AuthRepository interface for authentication operations
type AuthRepository interface {
	CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	GetRefreshToken(ctx context.Context, tokenHash string) (*entities.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error
	CleanupExpiredTokens(ctx context.Context) error
}

// Filter types for repository queries

// UserFilter represents filters for user queries
type UserFilter struct {
	Role      *entities.UserRole `json:"role"`
	IsActive  *bool              `json:"is_active"`
	Search    *string            `json:"search"`
	Limit     int                `json:"limit"`
	Offset    int                `json:"offset"`
	SortBy    string             `json:"sort_by"`
	SortOrder string             `json:"sort_order"`
}

// ProjectFilter represents filters for project queries
type ProjectFilter struct {
	Status     *entities.ProjectStatus `json:"status"`
	ManagerID  *uuid.UUID              `json:"manager_id"`
	Search     *string                 `json:"search"`
	StartDate  *time.Time              `json:"start_date"`
	EndDate    *time.Time              `json:"end_date"`
	Limit      int                     `json:"limit"`
	Offset     int                     `json:"offset"`
	SortBy     string                  `json:"sort_by"`
	SortOrder  string                  `json:"sort_order"`
}

// TaskFilter represents filters for task queries
type TaskFilter struct {
	Status     *entities.TaskStatus   `json:"status"`
	Priority   *entities.TaskPriority `json:"priority"`
	ProjectID  *int                   `json:"project_id"`
	AssigneeID *uuid.UUID             `json:"assignee_id"`
	CreatedBy  *uuid.UUID             `json:"created_by"`
	DueDateFrom *time.Time            `json:"due_date_from"`
	DueDateTo   *time.Time            `json:"due_date_to"`
	Search     *string                `json:"search"`
	Limit      int                    `json:"limit"`
	Offset     int                    `json:"offset"`
	SortBy     string                 `json:"sort_by"`
	SortOrder  string                 `json:"sort_order"`
}

// TimeEntryFilter represents filters for time entry queries
type TimeEntryFilter struct {
	UserID    *uuid.UUID `json:"user_id"`
	TaskID    *int       `json:"task_id"`
	ProjectID *int       `json:"project_id"`
	StartDate *time.Time `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`
	Limit     int        `json:"limit"`
	Offset    int        `json:"offset"`
	SortBy    string     `json:"sort_by"`
	SortOrder string     `json:"sort_order"`
}
