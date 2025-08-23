package entities

import (
	"time"

	"github.com/google/uuid"
)

// UserRole represents user roles in the system
type UserRole string

const (
	UserRoleAdmin     UserRole = "admin"
	UserRoleManager   UserRole = "manager"
	UserRoleDeveloper UserRole = "developer"
	UserRoleTester    UserRole = "tester"
	UserRoleViewer    UserRole = "viewer"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusInReview   TaskStatus = "in_review"
	TaskStatusDone       TaskStatus = "done"
	TaskStatusBlocked    TaskStatus = "blocked"
)

// TaskPriority represents task priority levels
type TaskPriority string

const (
	TaskPriorityLow      TaskPriority = "low"
	TaskPriorityMedium   TaskPriority = "medium"
	TaskPriorityHigh     TaskPriority = "high"
	TaskPriorityCritical TaskPriority = "critical"
)

// ProjectStatus represents project status
type ProjectStatus string

const (
	ProjectStatusPlanning   ProjectStatus = "planning"
	ProjectStatusActive     ProjectStatus = "active"
	ProjectStatusOnHold     ProjectStatus = "on_hold"
	ProjectStatusCompleted  ProjectStatus = "completed"
	ProjectStatusCancelled  ProjectStatus = "cancelled"
)

// User represents a user in the system
type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	Username     string    `json:"username" db:"username"`
	PasswordHash string    `json:"-" db:"password_hash"` // Never expose password hash
	FirstName    *string   `json:"first_name" db:"first_name"`
	LastName     *string   `json:"last_name" db:"last_name"`
	Role         UserRole  `json:"role" db:"role"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	LastLoginAt  *time.Time `json:"last_login_at" db:"last_login_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// Project represents a project in the system
type Project struct {
	ID          int           `json:"id" db:"id"`
	Name        string        `json:"name" db:"name"`
	Description *string       `json:"description" db:"description"`
	Status      ProjectStatus `json:"status" db:"status"`
	StartDate   *time.Time    `json:"start_date" db:"start_date"`
	EndDate     *time.Time    `json:"end_date" db:"end_date"`
	ManagerID   uuid.UUID     `json:"manager_id" db:"manager_id"`
	CreatedBy   uuid.UUID     `json:"created_by" db:"created_by"`
	CreatedAt   time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at" db:"updated_at"`
	
	// Associations
	Manager *User `json:"manager,omitempty"`
	Creator *User `json:"creator,omitempty"`
}

// Task represents a task in the system
type Task struct {
	ID          int          `json:"id" db:"id"`
	Title       string       `json:"title" db:"title"`
	Description *string      `json:"description" db:"description"`
	Status      TaskStatus   `json:"status" db:"status"`
	Priority    TaskPriority `json:"priority" db:"priority"`
	ProjectID   int          `json:"project_id" db:"project_id"`
	AssigneeID  *uuid.UUID   `json:"assignee_id" db:"assignee_id"`
	CreatedBy   uuid.UUID    `json:"created_by" db:"created_by"`
	DueDate     *time.Time   `json:"due_date" db:"due_date"`
	CreatedAt   time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at" db:"updated_at"`
	
	// Associations
	Project  *Project `json:"project,omitempty"`
	Assignee *User    `json:"assignee,omitempty"`
	Creator  *User    `json:"creator,omitempty"`
}

// TimeEntry represents a time tracking entry
type TimeEntry struct {
	ID          uuid.UUID `json:"id" db:"id"`
	TaskID      int       `json:"task_id" db:"task_id"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	Description *string   `json:"description" db:"description"`
	Hours       float64   `json:"hours" db:"hours"`
	StartTime   time.Time `json:"start_time" db:"start_time"`
	EndTime     *time.Time `json:"end_time" db:"end_time"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	
	// Associations
	Task *Task `json:"task,omitempty"`
	User *User `json:"user,omitempty"`
}

// RefreshToken represents a refresh token for JWT authentication
type RefreshToken struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Token     string    `json:"token" db:"token"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	RevokedAt *time.Time `json:"revoked_at" db:"revoked_at"`
}

// ProjectMember represents project membership
type ProjectMember struct {
	ProjectID uuid.UUID `json:"project_id" db:"project_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Role      string    `json:"role" db:"role"`
	JoinedAt  time.Time `json:"joined_at" db:"joined_at"`
	
	// Associations
	Project *Project `json:"project,omitempty"`
	User    *User    `json:"user,omitempty"`
}

// TaskComment represents comments on tasks
type TaskComment struct {
	ID        uuid.UUID `json:"id" db:"id"`
	TaskID    int       `json:"task_id" db:"task_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Content   string    `json:"content" db:"content"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	
	// Associations
	Task *Task `json:"task,omitempty"`
	User *User `json:"user,omitempty"`
}

// GetFullName returns the full name of a user
func (u *User) GetFullName() string {
	if u.FirstName != nil && u.LastName != nil {
		return *u.FirstName + " " + *u.LastName
	}
	if u.FirstName != nil {
		return *u.FirstName
	}
	if u.LastName != nil {
		return *u.LastName
	}
	return u.Username
}

// IsOverdue checks if a task is overdue
func (t *Task) IsOverdue() bool {
	if t.DueDate == nil {
		return false
	}
	return time.Now().After(*t.DueDate) && t.Status != TaskStatusDone
}

// GetDaysUntilDue returns the number of days until the task is due
func (t *Task) GetDaysUntilDue() *int {
	if t.DueDate == nil {
		return nil
	}
	days := int(time.Until(*t.DueDate).Hours() / 24)
	return &days
}

// IsActive checks if a project is currently active
func (p *Project) IsActive() bool {
	return p.Status == ProjectStatusActive
}

// GetDuration returns the duration of a time entry
func (te *TimeEntry) GetDuration() time.Duration {
	if te.EndTime == nil {
		return time.Since(te.StartTime)
	}
	return te.EndTime.Sub(te.StartTime)
}

// IsActive checks if a time entry is currently active (not ended)
func (te *TimeEntry) IsActive() bool {
	return te.EndTime == nil
}

// IsExpired checks if a refresh token is expired
func (rt *RefreshToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}

// IsRevoked checks if a refresh token is revoked
func (rt *RefreshToken) IsRevoked() bool {
	return rt.RevokedAt != nil
}
