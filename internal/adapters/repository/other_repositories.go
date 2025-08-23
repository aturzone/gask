package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/taskmaster/core/internal/domain/entities"
	"github.com/taskmaster/core/internal/ports"
)

// ProjectRepository implements the project repository interface
type ProjectRepository struct {
	db *sqlx.DB
}

// NewProjectRepository creates a new project repository
func NewProjectRepository(db *sqlx.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// Create creates a new project
func (r *ProjectRepository) Create(ctx context.Context, project *entities.Project) (*entities.Project, error) {
	// Simplified implementation - in a real app, this would interact with the database
	project.ID = 1 // Mock ID assignment
	return project, nil
}

// GetByID retrieves a project by ID
func (r *ProjectRepository) GetByID(ctx context.Context, id int) (*entities.Project, error) {
	// Simplified implementation
	return &entities.Project{
		ID:          id,
		Name:        "Sample Project",
		Status:      entities.ProjectStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

// Update updates a project
func (r *ProjectRepository) Update(ctx context.Context, project *entities.Project) (*entities.Project, error) {
	project.UpdatedAt = time.Now()
	return project, nil
}

// Delete deletes a project
func (r *ProjectRepository) Delete(ctx context.Context, id int) error {
	return nil
}

// List retrieves projects with filtering and pagination
func (r *ProjectRepository) List(ctx context.Context, filter ports.ProjectFilter) ([]*entities.Project, int, error) {
	projects := []*entities.Project{
		{
			ID:        1,
			Name:      "Sample Project",
			Status:    entities.ProjectStatusActive,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	return projects, 1, nil
}

// GetUserProjects gets all projects associated with a user
func (r *ProjectRepository) GetUserProjects(ctx context.Context, userID uuid.UUID) ([]*entities.Project, error) {
	return []*entities.Project{}, nil
}

// TaskRepository implements the task repository interface
type TaskRepository struct {
	db *sqlx.DB
}

// NewTaskRepository creates a new task repository
func NewTaskRepository(db *sqlx.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

// Create creates a new task
func (r *TaskRepository) Create(ctx context.Context, task *entities.Task) (*entities.Task, error) {
	task.ID = 1 // Mock ID assignment
	return task, nil
}

// GetByID retrieves a task by ID
func (r *TaskRepository) GetByID(ctx context.Context, id int) (*entities.Task, error) {
	return &entities.Task{
		ID:        id,
		Title:     "Sample Task",
		Status:    entities.TaskStatusTodo,
		Priority:  entities.TaskPriorityMedium,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// Update updates a task
func (r *TaskRepository) Update(ctx context.Context, task *entities.Task) (*entities.Task, error) {
	task.UpdatedAt = time.Now()
	return task, nil
}

// Delete deletes a task
func (r *TaskRepository) Delete(ctx context.Context, id int) error {
	return nil
}

// List retrieves tasks with filtering and pagination
func (r *TaskRepository) List(ctx context.Context, filter ports.TaskFilter) ([]*entities.Task, int, error) {
	tasks := []*entities.Task{
		{
			ID:        1,
			Title:     "Sample Task",
			Status:    entities.TaskStatusTodo,
			Priority:  entities.TaskPriorityMedium,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	return tasks, 1, nil
}

// GetByProject retrieves tasks by project ID
func (r *TaskRepository) GetByProject(ctx context.Context, projectID int) ([]*entities.Task, error) {
	return []*entities.Task{}, nil
}

// GetByAssignee retrieves tasks by assignee ID
func (r *TaskRepository) GetByAssignee(ctx context.Context, userID uuid.UUID) ([]*entities.Task, error) {
	return []*entities.Task{}, nil
}

// GetOverdue retrieves overdue tasks
func (r *TaskRepository) GetOverdue(ctx context.Context) ([]*entities.Task, error) {
	return []*entities.Task{}, nil
}

// GetNearDeadline retrieves tasks with approaching deadlines
func (r *TaskRepository) GetNearDeadline(ctx context.Context, days int) ([]*entities.Task, error) {
	return []*entities.Task{}, nil
}

// UpdateStatus updates a task's status
func (r *TaskRepository) UpdateStatus(ctx context.Context, id int, status entities.TaskStatus) error {
	return nil
}

// Assign assigns a task to a user
func (r *TaskRepository) Assign(ctx context.Context, id int, assigneeID uuid.UUID) error {
	return nil
}

// TimeEntryRepository implements the time entry repository interface
type TimeEntryRepository struct {
	db *sqlx.DB
}

// NewTimeEntryRepository creates a new time entry repository
func NewTimeEntryRepository(db *sqlx.DB) *TimeEntryRepository {
	return &TimeEntryRepository{db: db}
}

// Create creates a new time entry
func (r *TimeEntryRepository) Create(ctx context.Context, entry *entities.TimeEntry) (*entities.TimeEntry, error) {
	return entry, nil
}

// GetByID retrieves a time entry by ID
func (r *TimeEntryRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.TimeEntry, error) {
	return &entities.TimeEntry{
		ID:        id,
		Hours:     8.0,
		StartTime: time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// Update updates a time entry
func (r *TimeEntryRepository) Update(ctx context.Context, entry *entities.TimeEntry) (*entities.TimeEntry, error) {
	entry.UpdatedAt = time.Now()
	return entry, nil
}

// Delete deletes a time entry
func (r *TimeEntryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

// List retrieves time entries with filtering and pagination
func (r *TimeEntryRepository) List(ctx context.Context, filter ports.TimeEntryFilter) ([]*entities.TimeEntry, int, error) {
	entries := []*entities.TimeEntry{
		{
			ID:        uuid.New(),
			Hours:     8.0,
			StartTime: time.Now(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	return entries, 1, nil
}

// GetByUser retrieves time entries by user ID
func (r *TimeEntryRepository) GetByUser(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]*entities.TimeEntry, error) {
	return []*entities.TimeEntry{}, nil
}

// GetByTask retrieves time entries by task ID
func (r *TimeEntryRepository) GetByTask(ctx context.Context, taskID int) ([]*entities.TimeEntry, error) {
	return []*entities.TimeEntry{}, nil
}

// GetByProject retrieves time entries by project ID
func (r *TimeEntryRepository) GetByProject(ctx context.Context, projectID int, from, to time.Time) ([]*entities.TimeEntry, error) {
	return []*entities.TimeEntry{}, nil
}

// GetActiveEntry retrieves the active time entry for a user
func (r *TimeEntryRepository) GetActiveEntry(ctx context.Context, userID uuid.UUID) (*entities.TimeEntry, error) {
	return nil, fmt.Errorf("no active entry found")
}

// GetTotalHours calculates total hours for a user in a time range
func (r *TimeEntryRepository) GetTotalHours(ctx context.Context, userID uuid.UUID, from, to time.Time) (float64, error) {
	return 0.0, nil
}

// AuthRepository implements the auth repository interface
type AuthRepository struct {
	db *sqlx.DB
}

// NewAuthRepository creates a new auth repository
func NewAuthRepository(db *sqlx.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

// CreateRefreshToken creates a new refresh token
func (r *AuthRepository) CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	query := `
		INSERT INTO refresh_tokens (id, user_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.ExecContext(ctx, query, uuid.New(), userID, tokenHash, expiresAt, time.Now())
	if err != nil {
		return fmt.Errorf("failed to create refresh token: %w", err)
	}

	return nil
}

// GetRefreshToken retrieves a refresh token by hash
func (r *AuthRepository) GetRefreshToken(ctx context.Context, tokenHash string) (*entities.RefreshToken, error) {
	query := `
		SELECT id, user_id, token, expires_at, created_at, revoked_at
		FROM refresh_tokens WHERE token = $1
	`

	var token entities.RefreshToken
	row := r.db.QueryRowContext(ctx, query, tokenHash)

	err := row.Scan(
		&token.ID,
		&token.UserID,
		&token.Token,
		&token.ExpiresAt,
		&token.CreatedAt,
		&token.RevokedAt,
	)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, fmt.Errorf("refresh token not found")
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	return &token, nil
}

// RevokeRefreshToken revokes a refresh token
func (r *AuthRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	query := `UPDATE refresh_tokens SET revoked_at = $1 WHERE token = $2`

	_, err := r.db.ExecContext(ctx, query, time.Now(), tokenHash)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	return nil
}

// RevokeAllUserTokens revokes all refresh tokens for a user
func (r *AuthRepository) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE refresh_tokens SET revoked_at = $1 WHERE user_id = $2 AND revoked_at IS NULL`

	_, err := r.db.ExecContext(ctx, query, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to revoke user tokens: %w", err)
	}

	return nil
}

// CleanupExpiredTokens removes expired refresh tokens
func (r *AuthRepository) CleanupExpiredTokens(ctx context.Context) error {
	query := `DELETE FROM refresh_tokens WHERE expires_at < $1`

	_, err := r.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to cleanup expired tokens: %w", err)
	}

	return nil
}
