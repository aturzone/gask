package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/taskmaster/core/internal/domain/entities"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	Create(ctx context.Context, user *entities.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error)
	GetByEmail(ctx context.Context, email string) (*entities.User, error)
	GetByUsername(ctx context.Context, username string) (*entities.User, error)
	Update(ctx context.Context, user *entities.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter UserFilter) ([]*entities.User, error)
	Count(ctx context.Context, filter UserFilter) (int64, error)
	GetUserSkills(ctx context.Context, userID uuid.UUID) ([]entities.UserSkill, error)
	AddUserSkill(ctx context.Context, skill *entities.UserSkill) error
	UpdateUserSkill(ctx context.Context, skill *entities.UserSkill) error
	RemoveUserSkill(ctx context.Context, userID uuid.UUID, skillName string) error
}

// ProjectRepository defines the interface for project data operations
type ProjectRepository interface {
	Create(ctx context.Context, project *entities.Project) error
	GetByID(ctx context.Context, id int) (*entities.Project, error)
	GetByCode(ctx context.Context, code string) (*entities.Project, error)
	Update(ctx context.Context, project *entities.Project) error
	Delete(ctx context.Context, id int) error
	List(ctx context.Context, filter ProjectFilter) ([]*entities.Project, error)
	Count(ctx context.Context, filter ProjectFilter) (int64, error)
	GetProjectMembers(ctx context.Context, projectID int) ([]entities.ProjectMember, error)
	AddProjectMember(ctx context.Context, member *entities.ProjectMember) error
	UpdateProjectMember(ctx context.Context, member *entities.ProjectMember) error
	RemoveProjectMember(ctx context.Context, projectID int, userID uuid.UUID) error
	GetUserProjects(ctx context.Context, userID uuid.UUID) ([]*entities.Project, error)
}

// TaskRepository defines the interface for task data operations
type TaskRepository interface {
	Create(ctx context.Context, task *entities.Task) error
	GetByID(ctx context.Context, id int) (*entities.Task, error)
	Update(ctx context.Context, task *entities.Task) error
	Delete(ctx context.Context, id int) error
	List(ctx context.Context, filter TaskFilter) ([]*entities.Task, error)
	Count(ctx context.Context, filter TaskFilter) (int64, error)
	GetProjectTasks(ctx context.Context, projectID int, filter TaskFilter) ([]*entities.Task, error)
	GetUserTasks(ctx context.Context, userID uuid.UUID, filter TaskFilter) ([]*entities.Task, error)
	GetSubtasks(ctx context.Context, parentTaskID int) ([]*entities.Task, error)
	GetTasksNearDeadline(ctx context.Context, days int) ([]*entities.Task, error)
	GetOverdueTasks(ctx context.Context) ([]*entities.Task, error)
	BulkUpdateStatus(ctx context.Context, taskIDs []int, status entities.TaskStatus) error
}

// TimeEntryRepository defines the interface for time entry data operations
type TimeEntryRepository interface {
	Create(ctx context.Context, entry *entities.TimeEntry) error
	GetByID(ctx context.Context, id int) (*entities.TimeEntry, error)
	Update(ctx context.Context, entry *entities.TimeEntry) error
	Delete(ctx context.Context, id int) error
	List(ctx context.Context, filter TimeEntryFilter) ([]*entities.TimeEntry, error)
	GetUserEntries(ctx context.Context, userID uuid.UUID, filter TimeEntryFilter) ([]*entities.TimeEntry, error)
	GetProjectEntries(ctx context.Context, projectID int, filter TimeEntryFilter) ([]*entities.TimeEntry, error)
	GetTaskEntries(ctx context.Context, taskID int) ([]*entities.TimeEntry, error)
	GetActiveEntry(ctx context.Context, userID uuid.UUID) (*entities.TimeEntry, error)
	GetTotalHoursForPeriod(ctx context.Context, userID uuid.UUID, start, end time.Time) (float64, error)
}

// NoteRepository defines the interface for note data operations
type NoteRepository interface {
	Create(ctx context.Context, note *entities.Note) error
	GetByID(ctx context.Context, id int) (*entities.Note, error)
	Update(ctx context.Context, note *entities.Note) error
	Delete(ctx context.Context, id int) error
	List(ctx context.Context, filter NoteFilter) ([]*entities.Note, error)
	GetProjectNotes(ctx context.Context, projectID int) ([]*entities.Note, error)
	GetTaskNotes(ctx context.Context, taskID int) ([]*entities.Note, error)
	GetUserNotes(ctx context.Context, userID uuid.UUID) ([]*entities.Note, error)
	Search(ctx context.Context, query string, userID uuid.UUID) ([]*entities.Note, error)
}

// AuthRepository defines the interface for authentication operations
type AuthRepository interface {
	CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error
	CleanupExpiredTokens(ctx context.Context) error
}

// CacheRepository defines the interface for caching operations
type CacheRepository interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	Delete(ctx context.Context, key string) error
	DeletePattern(ctx context.Context, pattern string) error
	Exists(ctx context.Context, key string) (bool, error)
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error)
	Increment(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, expiration time.Duration) error
}

// Filter types for repository queries
type UserFilter struct {
	Role      *entities.UserRole
	IsActive  *bool
	Skills    []string
	Search    *string
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}

type ProjectFilter struct {
	Status    *entities.ProjectStatus
	OwnerID   *uuid.UUID
	Priority  *entities.Priority
	Search    *string
	StartDate *time.Time
	EndDate   *time.Time
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}

type TaskFilter struct {
	ProjectID  *int
	AssigneeID *uuid.UUID
	ReporterID *uuid.UUID
	Status     *entities.TaskStatus
	Priority   *entities.Priority
	DueBefore  *time.Time
	DueAfter   *time.Time
	Tags       []string
	Search     *string
	Limit      int
	Offset     int
	SortBy     string
	SortOrder  string
}

type TimeEntryFilter struct {
	UserID    *uuid.UUID
	ProjectID *int
	TaskID    *int
	StartDate *time.Time
	EndDate   *time.Time
	Billable  *bool
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}

type NoteFilter struct {
	AuthorID  *uuid.UUID
	ProjectID *int
	TaskID    *int
	UserID    *uuid.UUID
	IsPrivate *bool
	Tags      []string
	Search    *string
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}

// RefreshToken represents a refresh token record
type RefreshToken struct {
	ID        int       `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	TokenHash string    `json:"token_hash" db:"token_hash"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	RevokedAt *time.Time `json:"revoked_at" db:"revoked_at"`
}

// IsExpired checks if the refresh token is expired
func (rt *RefreshToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}

// IsRevoked checks if the refresh token is revoked
func (rt *RefreshToken) IsRevoked() bool {
	return rt.RevokedAt != nil
}

// IsValid checks if the refresh token is valid
func (rt *RefreshToken) IsValid() bool {
	return !rt.IsExpired() && !rt.IsRevoked()
}