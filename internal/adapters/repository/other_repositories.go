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

// AuthRepositoryImpl implements the AuthRepository interface
type AuthRepositoryImpl struct {
	db *sqlx.DB
}

// NewAuthRepository creates a new auth repository
func NewAuthRepository(db *sqlx.DB) ports.AuthRepository {
	return &AuthRepositoryImpl{db: db}
}

func (r *AuthRepositoryImpl) CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	query := `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`

	_, err := r.db.ExecContext(ctx, query, userID, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}

	return nil
}

func (r *AuthRepositoryImpl) GetRefreshToken(ctx context.Context, tokenHash string) (*ports.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, created_at, revoked_at
		FROM refresh_tokens 
		WHERE token_hash = $1`

	var token ports.RefreshToken
	err := r.db.GetContext(ctx, &token, query, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("get refresh token: %w", err)
	}

	return &token, nil
}

func (r *AuthRepositoryImpl) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	query := `
		UPDATE refresh_tokens 
		SET revoked_at = CURRENT_TIMESTAMP 
		WHERE token_hash = $1 AND revoked_at IS NULL`

	_, err := r.db.ExecContext(ctx, query, tokenHash)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	return nil
}

func (r *AuthRepositoryImpl) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE refresh_tokens 
		SET revoked_at = CURRENT_TIMESTAMP 
		WHERE user_id = $1 AND revoked_at IS NULL`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("revoke all user tokens: %w", err)
	}

	return nil
}

func (r *AuthRepositoryImpl) CleanupExpiredTokens(ctx context.Context) error {
	query := `DELETE FROM refresh_tokens WHERE expires_at < CURRENT_TIMESTAMP`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("cleanup expired tokens: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Cleaned up %d expired tokens\n", rowsAffected)

	return nil
}

// TimeEntryRepositoryImpl implements the TimeEntryRepository interface
type TimeEntryRepositoryImpl struct {
	db *sqlx.DB
}

// NewTimeEntryRepository creates a new time entry repository
func NewTimeEntryRepository(db *sqlx.DB) ports.TimeEntryRepository {
	return &TimeEntryRepositoryImpl{db: db}
}

func (r *TimeEntryRepositoryImpl) Create(ctx context.Context, entry *entities.TimeEntry) error {
	query := `
		INSERT INTO time_entries (user_id, task_id, project_id, start_time, end_time, 
			duration_minutes, description, entry_date, billable, hourly_rate)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query,
		entry.UserID, entry.TaskID, entry.ProjectID, entry.StartTime, entry.EndTime,
		entry.DurationMinutes, entry.Description, entry.EntryDate, entry.Billable, entry.HourlyRate,
	).Scan(&entry.ID, &entry.CreatedAt, &entry.UpdatedAt)

	if err != nil {
		return fmt.Errorf("create time entry: %w", err)
	}

	return nil
}

func (r *TimeEntryRepositoryImpl) GetByID(ctx context.Context, id int) (*entities.TimeEntry, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *TimeEntryRepositoryImpl) Update(ctx context.Context, entry *entities.TimeEntry) error {
	return fmt.Errorf("not implemented")
}

func (r *TimeEntryRepositoryImpl) Delete(ctx context.Context, id int) error {
	return fmt.Errorf("not implemented")
}

func (r *TimeEntryRepositoryImpl) List(ctx context.Context, filter ports.TimeEntryFilter) ([]*entities.TimeEntry, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *TimeEntryRepositoryImpl) GetUserEntries(ctx context.Context, userID uuid.UUID, filter ports.TimeEntryFilter) ([]*entities.TimeEntry, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *TimeEntryRepositoryImpl) GetProjectEntries(ctx context.Context, projectID int, filter ports.TimeEntryFilter) ([]*entities.TimeEntry, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *TimeEntryRepositoryImpl) GetTaskEntries(ctx context.Context, taskID int) ([]*entities.TimeEntry, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *TimeEntryRepositoryImpl) GetActiveEntry(ctx context.Context, userID uuid.UUID) (*entities.TimeEntry, error) {
	query := `
		SELECT id, user_id, task_id, project_id, start_time, end_time,
			duration_minutes, description, entry_date, billable, hourly_rate,
			created_at, updated_at
		FROM time_entries 
		WHERE user_id = $1 AND end_time IS NULL
		ORDER BY start_time DESC
		LIMIT 1`

	var entry entities.TimeEntry
	err := r.db.GetContext(ctx, &entry, query, userID)
	if err != nil {
		return nil, fmt.Errorf("no active time entry found")
	}

	return &entry, nil
}

func (r *TimeEntryRepositoryImpl) GetTotalHoursForPeriod(ctx context.Context, userID uuid.UUID, start, end time.Time) (float64, error) {
	return 0, fmt.Errorf("not implemented")
}

// ProjectRepositoryImpl - basic implementation
type ProjectRepositoryImpl struct {
	db *sqlx.DB
}

func NewProjectRepository(db *sqlx.DB) ports.ProjectRepository {
	return &ProjectRepositoryImpl{db: db}
}

func (r *ProjectRepositoryImpl) Create(ctx context.Context, project *entities.Project) error {
	return fmt.Errorf("not implemented")
}

func (r *ProjectRepositoryImpl) GetByID(ctx context.Context, id int) (*entities.Project, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *ProjectRepositoryImpl) GetByCode(ctx context.Context, code string) (*entities.Project, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *ProjectRepositoryImpl) Update(ctx context.Context, project *entities.Project) error {
	return fmt.Errorf("not implemented")
}

func (r *ProjectRepositoryImpl) Delete(ctx context.Context, id int) error {
	return fmt.Errorf("not implemented")
}

func (r *ProjectRepositoryImpl) List(ctx context.Context, filter ports.ProjectFilter) ([]*entities.Project, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *ProjectRepositoryImpl) Count(ctx context.Context, filter ports.ProjectFilter) (int64, error) {
	return 0, fmt.Errorf("not implemented")
}

func (r *ProjectRepositoryImpl) GetProjectMembers(ctx context.Context, projectID int) ([]entities.ProjectMember, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *ProjectRepositoryImpl) AddProjectMember(ctx context.Context, member *entities.ProjectMember) error {
	return fmt.Errorf("not implemented")
}

func (r *ProjectRepositoryImpl) UpdateProjectMember(ctx context.Context, member *entities.ProjectMember) error {
	return fmt.Errorf("not implemented")
}

func (r *ProjectRepositoryImpl) RemoveProjectMember(ctx context.Context, projectID int, userID uuid.UUID) error {
	return fmt.Errorf("not implemented")
}

func (r *ProjectRepositoryImpl) GetUserProjects(ctx context.Context, userID uuid.UUID) ([]*entities.Project, error) {
	return nil, fmt.Errorf("not implemented")
}

// TaskRepositoryImpl - basic implementation
type TaskRepositoryImpl struct {
	db *sqlx.DB
}

func NewTaskRepository(db *sqlx.DB) ports.TaskRepository {
	return &TaskRepositoryImpl{db: db}
}

func (r *TaskRepositoryImpl) Create(ctx context.Context, task *entities.Task) error {
	return fmt.Errorf("not implemented")
}

func (r *TaskRepositoryImpl) GetByID(ctx context.Context, id int) (*entities.Task, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *TaskRepositoryImpl) Update(ctx context.Context, task *entities.Task) error {
	return fmt.Errorf("not implemented")
}

func (r *TaskRepositoryImpl) Delete(ctx context.Context, id int) error {
	return fmt.Errorf("not implemented")
}

func (r *TaskRepositoryImpl) List(ctx context.Context, filter ports.TaskFilter) ([]*entities.Task, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *TaskRepositoryImpl) Count(ctx context.Context, filter ports.TaskFilter) (int64, error) {
	return 0, fmt.Errorf("not implemented")
}

func (r *TaskRepositoryImpl) GetProjectTasks(ctx context.Context, projectID int, filter ports.TaskFilter) ([]*entities.Task, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *TaskRepositoryImpl) GetUserTasks(ctx context.Context, userID uuid.UUID, filter ports.TaskFilter) ([]*entities.Task, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *TaskRepositoryImpl) GetSubtasks(ctx context.Context, parentTaskID int) ([]*entities.Task, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *TaskRepositoryImpl) GetTasksNearDeadline(ctx context.Context, days int) ([]*entities.Task, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *TaskRepositoryImpl) GetOverdueTasks(ctx context.Context) ([]*entities.Task, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *TaskRepositoryImpl) BulkUpdateStatus(ctx context.Context, taskIDs []int, status entities.TaskStatus) error {
	return fmt.Errorf("not implemented")
}
