package entities

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Common errors
var (
	ErrTaskNotFound           = errors.New("task not found")
	ErrProjectNotFound        = errors.New("project not found")
	ErrUserNotFound           = errors.New("user not found")
	ErrInvalidStatus          = errors.New("invalid status")
	ErrUnauthorized           = errors.New("unauthorized")
	ErrTaskNotAssignable      = errors.New("task is not assignable")
	ErrInsufficientSkills     = errors.New("insufficient skills for task")
	ErrBudgetExceeded         = errors.New("budget exceeded")
	ErrInvalidTimeEntry       = errors.New("invalid time entry")
	ErrDeadlineInPast         = errors.New("deadline cannot be in the past")
	ErrTaskAlreadyCompleted   = errors.New("task is already completed")
	ErrInvalidWorkingHours    = errors.New("invalid working hours")
)

// Enums and types
type UserRole string

const (
	UserRoleAdmin          UserRole = "admin"
	UserRoleProjectManager UserRole = "project_manager"
	UserRoleTeamLead       UserRole = "team_lead"
	UserRoleDeveloper      UserRole = "developer"
	UserRoleViewer         UserRole = "viewer"
)

type ProjectStatus string

const (
	ProjectStatusPlanning   ProjectStatus = "planning"
	ProjectStatusActive     ProjectStatus = "active"
	ProjectStatusOnHold     ProjectStatus = "on_hold"
	ProjectStatusCompleted  ProjectStatus = "completed"
	ProjectStatusCancelled  ProjectStatus = "cancelled"
)

type TaskStatus string

const (
	TaskStatusTodo        TaskStatus = "todo"
	TaskStatusInProgress  TaskStatus = "in_progress"
	TaskStatusReview      TaskStatus = "review"
	TaskStatusTesting     TaskStatus = "testing"
	TaskStatusCompleted   TaskStatus = "completed"
	TaskStatusCancelled   TaskStatus = "cancelled"
)

type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// User represents a user in the system
type User struct {
	ID                uuid.UUID    `json:"id" db:"id"`
	Email             string       `json:"email" db:"email"`
	Username          string       `json:"username" db:"username"`
	PasswordHash      string       `json:"-" db:"password_hash"`
	FirstName         *string      `json:"first_name" db:"first_name"`
	LastName          *string      `json:"last_name" db:"last_name"`
	Role              UserRole     `json:"role" db:"role"`
	IsActive          bool         `json:"is_active" db:"is_active"`
	WorkingHoursStart *time.Time   `json:"working_hours_start" db:"working_hours_start"`
	WorkingHoursEnd   *time.Time   `json:"working_hours_end" db:"working_hours_end"`
	WorkingDays       []int        `json:"working_days" db:"working_days"`
	Timezone          string       `json:"timezone" db:"timezone"`
	HourlyRate        *float64     `json:"hourly_rate" db:"hourly_rate"`
	Skills            []UserSkill  `json:"skills"`
	CreatedAt         time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at" db:"updated_at"`
	DeletedAt         *time.Time   `json:"deleted_at" db:"deleted_at"`
}

// UserSkill represents a user's skill and proficiency
type UserSkill struct {
	ID                  int     `json:"id" db:"id"`
	UserID              uuid.UUID `json:"user_id" db:"user_id"`
	SkillName           string  `json:"skill_name" db:"skill_name"`
	ProficiencyLevel    int     `json:"proficiency_level" db:"proficiency_level"` // 1-5 scale
	YearsOfExperience   *float64 `json:"years_of_experience" db:"years_of_experience"`
	IsCertified         bool    `json:"is_certified" db:"is_certified"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
}

// Project represents a project in the system
type Project struct {
	ID          int             `json:"id" db:"id"`
	Name        string          `json:"name" db:"name"`
	ProjectCode string          `json:"project_code" db:"project_code"`
	Description *string         `json:"description" db:"description"`
	Status      ProjectStatus   `json:"status" db:"status"`
	Priority    Priority        `json:"priority" db:"priority"`
	StartDate   *time.Time      `json:"start_date" db:"start_date"`
	EndDate     *time.Time      `json:"end_date" db:"end_date"`
	Budget      *float64        `json:"budget" db:"budget"`
	SpentBudget float64         `json:"spent_budget" db:"spent_budget"`
	CurrencyCode string         `json:"currency_code" db:"currency_code"`
	OwnerID     *uuid.UUID      `json:"owner_id" db:"owner_id"`
	ClientName  *string         `json:"client_name" db:"client_name"`
	Tasks       []Task          `json:"tasks"`
	TeamMembers []ProjectMember `json:"team_members"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
	DeletedAt   *time.Time      `json:"deleted_at" db:"deleted_at"`
	Version     int             `json:"version" db:"version"`
}

// ProjectMember represents a team member assigned to a project
type ProjectMember struct {
	ID           int       `json:"id" db:"id"`
	ProjectID    int       `json:"project_id" db:"project_id"`
	UserID       uuid.UUID `json:"user_id" db:"user_id"`
	Role         string    `json:"role" db:"role"`
	AllocationPercentage float64 `json:"allocation_percentage" db:"allocation_percentage"`
	JoinedAt     time.Time `json:"joined_at" db:"joined_at"`
	LeftAt       *time.Time `json:"left_at" db:"left_at"`
}

// Task represents a task in the system
type Task struct {
	ID           int        `json:"id" db:"id"`
	ProjectID    int        `json:"project_id" db:"project_id"`
	ParentTaskID *int       `json:"parent_task_id" db:"parent_task_id"`
	Title        string     `json:"title" db:"title"`
	Description  *string    `json:"description" db:"description"`
	Status       TaskStatus `json:"status" db:"status"`
	Priority     Priority   `json:"priority" db:"priority"`
	AssigneeID   *uuid.UUID `json:"assignee_id" db:"assignee_id"`
	ReporterID   *uuid.UUID `json:"reporter_id" db:"reporter_id"`
	EstimatedHours *float64 `json:"estimated_hours" db:"estimated_hours"`
	ActualHours  float64    `json:"actual_hours" db:"actual_hours"`
	StartDate    *time.Time `json:"start_date" db:"start_date"`
	DueDate      *time.Time `json:"due_date" db:"due_date"`
	CompletedAt  *time.Time `json:"completed_at" db:"completed_at"`
	Tags         []string   `json:"tags" db:"tags"`
	Subtasks     []Task     `json:"subtasks"`
	TimeEntries  []TimeEntry `json:"time_entries"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at" db:"deleted_at"`
	Version      int        `json:"version" db:"version"`
}

// TimeEntry represents a time tracking entry
type TimeEntry struct {
	ID              int       `json:"id" db:"id"`
	UserID          uuid.UUID `json:"user_id" db:"user_id"`
	TaskID          *int      `json:"task_id" db:"task_id"`
	ProjectID       int       `json:"project_id" db:"project_id"`
	StartTime       time.Time `json:"start_time" db:"start_time"`
	EndTime         *time.Time `json:"end_time" db:"end_time"`
	DurationMinutes *int      `json:"duration_minutes" db:"duration_minutes"`
	Description     *string   `json:"description" db:"description"`
	EntryDate       time.Time `json:"entry_date" db:"entry_date"`
	Billable        bool      `json:"billable" db:"billable"`
	HourlyRate      *float64  `json:"hourly_rate" db:"hourly_rate"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// Note represents a note in the system
type Note struct {
	ID         int        `json:"id" db:"id"`
	Title      string     `json:"title" db:"title"`
	Content    string     `json:"content" db:"content"`
	AuthorID   uuid.UUID  `json:"author_id" db:"author_id"`
	ProjectID  *int       `json:"project_id" db:"project_id"`
	TaskID     *int       `json:"task_id" db:"task_id"`
	UserID     *uuid.UUID `json:"user_id" db:"user_id"`
	Tags       []string   `json:"tags" db:"tags"`
	IsPrivate  bool       `json:"is_private" db:"is_private"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at" db:"deleted_at"`
}

// Business logic methods for User
func (u *User) CanAssignTask(task *Task) bool {
	if !u.IsActive {
		return false
	}
	
	switch u.Role {
	case UserRoleAdmin, UserRoleProjectManager:
		return true
	case UserRoleTeamLead:
		// Team leads can assign tasks in their projects
		return true
	default:
		return false
	}
}

func (u *User) CanEditProject(project *Project) bool {
	if !u.IsActive {
		return false
	}
	
	switch u.Role {
	case UserRoleAdmin:
		return true
	case UserRoleProjectManager:
		return project.OwnerID != nil && *project.OwnerID == u.ID
	default:
		return false
	}
}

func (u *User) HasSkill(skillName string, minLevel int) bool {
	for _, skill := range u.Skills {
		if skill.SkillName == skillName && skill.ProficiencyLevel >= minLevel {
			return true
		}
	}
	return false
}

func (u *User) GetWeeklyCapacity() float64 {
	if u.WorkingHoursStart == nil || u.WorkingHoursEnd == nil {
		return 40.0 // Default 40 hours per week
	}
	
	dailyHours := u.WorkingHoursEnd.Sub(*u.WorkingHoursStart).Hours()
	workingDays := float64(len(u.WorkingDays))
	
	return dailyHours * workingDays
}

// Business logic methods for Project
func (p *Project) IsActive() bool {
	return p.Status == ProjectStatusActive
}

func (p *Project) IsOverBudget() bool {
	return p.Budget != nil && p.SpentBudget > *p.Budget
}

func (p *Project) GetBudgetUtilization() float64 {
	if p.Budget == nil || *p.Budget == 0 {
		return 0
	}
	return (p.SpentBudget / *p.Budget) * 100
}

func (p *Project) IsOverdue() bool {
	if p.EndDate == nil {
		return false
	}
	return time.Now().After(*p.EndDate) && p.Status != ProjectStatusCompleted
}

func (p *Project) CanAddTask() bool {
	return p.Status == ProjectStatusPlanning || p.Status == ProjectStatusActive
}

func (p *Project) AddBudgetExpense(amount float64) error {
	if p.Budget != nil && (p.SpentBudget+amount) > *p.Budget {
		return ErrBudgetExceeded
	}
	p.SpentBudget += amount
	return nil
}

// Business logic methods for Task
func (t *Task) CanBeAssigned() bool {
	return t.Status == TaskStatusTodo
}

func (t *Task) CanBeStarted() bool {
	return t.Status == TaskStatusTodo && t.AssigneeID != nil
}

func (t *Task) CanBeCompleted() bool {
	return t.Status == TaskStatusInProgress || t.Status == TaskStatusReview || t.Status == TaskStatusTesting
}

func (t *Task) IsOverdue() bool {
	if t.DueDate == nil {
		return false
	}
	return time.Now().After(*t.DueDate) && t.Status != TaskStatusCompleted
}

func (t *Task) AssignTo(userID uuid.UUID) error {
	if !t.CanBeAssigned() {
		return ErrTaskNotAssignable
	}
	
	t.AssigneeID = &userID
	t.Status = TaskStatusTodo
	return nil
}

func (t *Task) Start() error {
	if !t.CanBeStarted() {
		return ErrInvalidStatus
	}
	
	t.Status = TaskStatusInProgress
	if t.StartDate == nil {
		now := time.Now()
		t.StartDate = &now
	}
	return nil
}

func (t *Task) Complete(actualHours float64) error {
	if !t.CanBeCompleted() {
		return ErrInvalidStatus
	}
	
	t.Status = TaskStatusCompleted
	t.ActualHours = actualHours
	now := time.Now()
	t.CompletedAt = &now
	return nil
}

func (t *Task) SetDueDate(dueDate time.Time) error {
	if dueDate.Before(time.Now()) {
		return ErrDeadlineInPast
	}
	
	t.DueDate = &dueDate
	return nil
}

func (t *Task) GetProgress() float64 {
	switch t.Status {
	case TaskStatusTodo:
		return 0
	case TaskStatusInProgress:
		return 25
	case TaskStatusReview:
		return 75
	case TaskStatusTesting:
		return 90
	case TaskStatusCompleted:
		return 100
	default:
		return 0
	}
}

func (t *Task) GetEffort() float64 {
	if t.EstimatedHours == nil {
		return t.ActualHours
	}
	return *t.EstimatedHours
}

func (t *Task) IsEstimateAccurate() bool {
	if t.EstimatedHours == nil || t.Status != TaskStatusCompleted {
		return true
	}
	
	variance := (t.ActualHours - *t.EstimatedHours) / *t.EstimatedHours
	return variance >= -0.2 && variance <= 0.2 // Within 20% variance
}

// Business logic methods for TimeEntry
func (te *TimeEntry) CalculateDuration() time.Duration {
	if te.EndTime == nil {
		return 0
	}
	return te.EndTime.Sub(te.StartTime)
}

func (te *TimeEntry) Stop() error {
	if te.EndTime != nil {
		return ErrInvalidTimeEntry
	}
	
	now := time.Now()
	te.EndTime = &now
	duration := int(te.CalculateDuration().Minutes())
	te.DurationMinutes = &duration
	
	return nil
}

func (te *TimeEntry) CalculateCost() float64 {
	if te.HourlyRate == nil || te.DurationMinutes == nil {
		return 0
	}
	
	hours := float64(*te.DurationMinutes) / 60.0
	return hours * *te.HourlyRate
}

func (te *TimeEntry) IsValid() bool {
	if te.EndTime != nil && te.EndTime.Before(te.StartTime) {
		return false
	}
	
	if te.DurationMinutes != nil && *te.DurationMinutes < 0 {
		return false
	}
	
	return true
}

// Utility methods
func (ur UserRole) IsValid() bool {
	switch ur {
	case UserRoleAdmin, UserRoleProjectManager, UserRoleTeamLead, UserRoleDeveloper, UserRoleViewer:
		return true
	default:
		return false
	}
}

func (ps ProjectStatus) IsValid() bool {
	switch ps {
	case ProjectStatusPlanning, ProjectStatusActive, ProjectStatusOnHold, ProjectStatusCompleted, ProjectStatusCancelled:
		return true
	default:
		return false
	}
}

func (ts TaskStatus) IsValid() bool {
	switch ts {
	case TaskStatusTodo, TaskStatusInProgress, TaskStatusReview, TaskStatusTesting, TaskStatusCompleted, TaskStatusCancelled:
		return true
	default:
		return false
	}
}

func (p Priority) IsValid() bool {
	switch p {
	case PriorityLow, PriorityMedium, PriorityHigh, PriorityCritical:
		return true
	default:
		return false
	}
}