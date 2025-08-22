// internal/adapters/repository/auth_repository.go
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
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

// internal/adapters/repository/project_repository.go
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/taskmaster/core/internal/domain/entities"
	"github.com/taskmaster/core/internal/ports"
)

// ProjectRepositoryImpl implements the ProjectRepository interface
type ProjectRepositoryImpl struct {
	db *sqlx.DB
}

// NewProjectRepository creates a new project repository
func NewProjectRepository(db *sqlx.DB) ports.ProjectRepository {
	return &ProjectRepositoryImpl{db: db}
}

func (r *ProjectRepositoryImpl) Create(ctx context.Context, project *entities.Project) error {
	query := `
		INSERT INTO projects (name, project_code, description, status, priority, 
			start_date, end_date, budget, currency_code, owner_id, client_name)
		VALUES (:name, :project_code, :description, :status, :priority,
			:start_date, :end_date, :budget, :currency_code, :owner_id, :client_name)
		RETURNING id, created_at, updated_at, version`

	stmt, err := r.db.PrepareNamedContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	err = stmt.GetContext(ctx, project, project)
	if err != nil {
		return fmt.Errorf("create project: %w", err)
	}

	return nil
}

func (r *ProjectRepositoryImpl) GetByID(ctx context.Context, id int) (*entities.Project, error) {
	query := `
		SELECT id, name, project_code, description, status, priority,
			start_date, end_date, budget, spent_budget, currency_code,
			owner_id, client_name, created_at, updated_at, version
		FROM projects 
		WHERE id = $1 AND deleted_at IS NULL`

	var project entities.Project
	err := r.db.GetContext(ctx, &project, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entities.ErrProjectNotFound
		}
		return nil, fmt.Errorf("get project by id: %w", err)
	}

	return &project, nil
}

func (r *ProjectRepositoryImpl) GetByCode(ctx context.Context, code string) (*entities.Project, error) {
	query := `
		SELECT id, name, project_code, description, status, priority,
			start_date, end_date, budget, spent_budget, currency_code,
			owner_id, client_name, created_at, updated_at, version
		FROM projects 
		WHERE project_code = $1 AND deleted_at IS NULL`

	var project entities.Project
	err := r.db.GetContext(ctx, &project, query, code)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entities.ErrProjectNotFound
		}
		return nil, fmt.Errorf("get project by code: %w", err)
	}

	return &project, nil
}

func (r *ProjectRepositoryImpl) Update(ctx context.Context, project *entities.Project) error {
	query := `
		UPDATE projects 
		SET name = :name, description = :description, status = :status, 
			priority = :priority, start_date = :start_date, end_date = :end_date,
			budget = :budget, currency_code = :currency_code, client_name = :client_name,
			version = version + 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = :id AND deleted_at IS NULL AND version = :version
		RETURNING updated_at, version`

	stmt, err := r.db.PrepareNamedContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	err = stmt.GetContext(ctx, project, project)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("project not found or version conflict")
		}
		return fmt.Errorf("update project: %w", err)
	}

	return nil
}

func (r *ProjectRepositoryImpl) Delete(ctx context.Context, id int) error {
	query := `UPDATE projects SET deleted_at = CURRENT_TIMESTAMP WHERE id = $1 AND deleted_at IS NULL`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return entities.ErrProjectNotFound
	}

	return nil
}

func (r *ProjectRepositoryImpl) List(ctx context.Context, filter ports.ProjectFilter) ([]*entities.Project, error) {
	query := `
		SELECT id, name, project_code, description, status, priority,
			start_date, end_date, budget, spent_budget, currency_code,
			owner_id, client_name, created_at, updated_at, version
		FROM projects 
		WHERE deleted_at IS NULL`

	args := []interface{}{}
	argIndex := 1

	// Apply filters
	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *filter.Status)
		argIndex++
	}

	if filter.OwnerID != nil {
		query += fmt.Sprintf(" AND owner_id = $%d", argIndex)
		args = append(args, *filter.OwnerID)
		argIndex++
	}

	if filter.Priority != nil {
		query += fmt.Sprintf(" AND priority = $%d", argIndex)
		args = append(args, *filter.Priority)
		argIndex++
	}

	if filter.Search != nil && *filter.Search != "" {
		query += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d OR project_code ILIKE $%d)", 
			argIndex, argIndex, argIndex)
		searchTerm := "%" + *filter.Search + "%"
		args = append(args, searchTerm)
		argIndex++
	}

	// Apply sorting
	if filter.SortBy != "" {
		order := "ASC"
		if filter.SortOrder == "desc" {
			order = "DESC"
		}
		query += fmt.Sprintf(" ORDER BY %s %s", filter.SortBy, order)
	} else {
		query += " ORDER BY created_at DESC"
	}

	// Apply pagination
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filter.Offset)
	}

	var projects []*entities.Project
	err := r.db.SelectContext(ctx, &projects, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}

	return projects, nil
}

func (r *ProjectRepositoryImpl) Count(ctx context.Context, filter ports.ProjectFilter) (int64, error) {
	query := `SELECT COUNT(*) FROM projects WHERE deleted_at IS NULL`

	args := []interface{}{}
	argIndex := 1

	// Apply same filters as List method
	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *filter.Status)
		argIndex++
	}

	if filter.OwnerID != nil {
		query += fmt.Sprintf(" AND owner_id = $%d", argIndex)
		args = append(args, *filter.OwnerID)
		argIndex++
	}

	if filter.Search != nil && *filter.Search != "" {
		query += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d OR project_code ILIKE $%d)", 
			argIndex, argIndex, argIndex)
		searchTerm := "%" + *filter.Search + "%"
		args = append(args, searchTerm)
	}

	var count int64
	err := r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		return 0, fmt.Errorf("count projects: %w", err)
	}

	return count, nil
}

func (r *ProjectRepositoryImpl) GetProjectMembers(ctx context.Context, projectID int) ([]entities.ProjectMember, error) {
	query := `
		SELECT id, project_id, user_id, role, allocation_percentage, joined_at, left_at
		FROM project_members 
		WHERE project_id = $1 AND left_at IS NULL
		ORDER BY joined_at ASC`

	var members []entities.ProjectMember
	err := r.db.SelectContext(ctx, &members, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("get project members: %w", err)
	}

	return members, nil
}

func (r *ProjectRepositoryImpl) AddProjectMember(ctx context.Context, member *entities.ProjectMember) error {
	query := `
		INSERT INTO project_members (project_id, user_id, role, allocation_percentage)
		VALUES (:project_id, :user_id, :role, :allocation_percentage)
		RETURNING id, joined_at`

	stmt, err := r.db.PrepareNamedContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	err = stmt.GetContext(ctx, member, member)
	if err != nil {
		return fmt.Errorf("add project member: %w", err)
	}

	return nil
}

func (r *ProjectRepositoryImpl) UpdateProjectMember(ctx context.Context, member *entities.ProjectMember) error {
	query := `
		UPDATE project_members 
		SET role = :role, allocation_percentage = :allocation_percentage
		WHERE id = :id`

	_, err := r.db.NamedExecContext(ctx, query, member)
	if err != nil {
		return fmt.Errorf("update project member: %w", err)
	}

	return nil
}

func (r *ProjectRepositoryImpl) RemoveProjectMember(ctx context.Context, projectID int, userID uuid.UUID) error {
	query := `
		UPDATE project_members 
		SET left_at = CURRENT_TIMESTAMP 
		WHERE project_id = $1 AND user_id = $2 AND left_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, projectID, userID)
	if err != nil {
		return fmt.Errorf("remove project member: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("member not found in project")
	}

	return nil
}

func (r *ProjectRepositoryImpl) GetUserProjects(ctx context.Context, userID uuid.UUID) ([]*entities.Project, error) {
	query := `
		SELECT DISTINCT p.id, p.name, p.project_code, p.description, p.status, p.priority,
			p.start_date, p.end_date, p.budget, p.spent_budget, p.currency_code,
			p.owner_id, p.client_name, p.created_at, p.updated_at, p.version
		FROM projects p
		INNER JOIN project_members pm ON p.id = pm.project_id
		WHERE pm.user_id = $1 AND pm.left_at IS NULL AND p.deleted_at IS NULL
		ORDER BY p.created_at DESC`

	var projects []*entities.Project
	err := r.db.SelectContext(ctx, &projects, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get user projects: %w", err)
	}

	return projects, nil
}

// internal/adapters/repository/cache_repository.go
package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/taskmaster/core/internal/ports"
)

// CacheRepositoryImpl implements the CacheRepository interface using Redis
type CacheRepositoryImpl struct {
	client *redis.Client
}

// NewCacheRepository creates a new cache repository
func NewCacheRepository(client *redis.Client) ports.CacheRepository {
	return &CacheRepositoryImpl{client: client}
}

func (r *CacheRepositoryImpl) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal value: %w", err)
	}

	err = r.client.Set(ctx, key, data, expiration).Err()
	if err != nil {
		return fmt.Errorf("set cache: %w", err)
	}

	return nil
}

func (r *CacheRepositoryImpl) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("key not found")
		}
		return fmt.Errorf("get cache: %w", err)
	}

	err = json.Unmarshal([]byte(data), dest)
	if err != nil {
		return fmt.Errorf("unmarshal value: %w", err)
	}

	return nil
}

func (r *CacheRepositoryImpl) Delete(ctx context.Context, key string) error {
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("delete cache: %w", err)
	}

	return nil
}

func (r *CacheRepositoryImpl) DeletePattern(ctx context.Context, pattern string) error {
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("get keys: %w", err)
	}

	if len(keys) > 0 {
		err = r.client.Del(ctx, keys...).Err()
		if err != nil {
			return fmt.Errorf("delete keys: %w", err)
		}
	}

	return nil
}

func (r *CacheRepositoryImpl) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("check exists: %w", err)
	}

	return count > 0, nil
}

func (r *CacheRepositoryImpl) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("marshal value: %w", err)
	}

	result, err := r.client.SetNX(ctx, key, data, expiration).Result()
	if err != nil {
		return false, fmt.Errorf("setnx cache: %w", err)
	}

	return result, nil
}

func (r *CacheRepositoryImpl) Increment(ctx context.Context, key string) (int64, error) {
	result, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("increment: %w", err)
	}

	return result, nil
}

func (r *CacheRepositoryImpl) Expire(ctx context.Context, key string, expiration time.Duration) error {
	err := r.client.Expire(ctx, key, expiration).Err()
	if err != nil {
		return fmt.Errorf("expire: %w", err)
	}

	return nil
}

// internal/adapters/repository/time_repository.go
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/taskmaster/core/internal/domain/entities"
	"github.com/taskmaster/core/internal/ports"
)

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
		VALUES (:user_id, :task_id, :project_id, :start_time, :end_time,
			:duration_minutes, :description, :entry_date, :billable, :hourly_rate)
		RETURNING id, created_at, updated_at`

	stmt, err := r.db.PrepareNamedContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	err = stmt.GetContext(ctx, entry, entry)
	if err != nil {
		return fmt.Errorf("create time entry: %w", err)
	}

	return nil
}

func (r *TimeEntryRepositoryImpl) GetByID(ctx context.Context, id int) (*entities.TimeEntry, error) {
	query := `
		SELECT id, user_id, task_id, project_id, start_time, end_time,
			duration_minutes, description, entry_date, billable, hourly_rate,
			created_at, updated_at
		FROM time_entries 
		WHERE id = $1`

	var entry entities.TimeEntry
	err := r.db.GetContext(ctx, &entry, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("time entry not found")
		}
		return nil, fmt.Errorf("get time entry by id: %w", err)
	}

	return &entry, nil
}

func (r *TimeEntryRepositoryImpl) Update(ctx context.Context, entry *entities.TimeEntry) error {
	query := `
		UPDATE time_entries 
		SET start_time = :start_time, end_time = :end_time, duration_minutes = :duration_minutes,
			description = :description, entry_date = :entry_date, billable = :billable,
			hourly_rate = :hourly_rate, updated_at = CURRENT_TIMESTAMP
		WHERE id = :id
		RETURNING updated_at`

	stmt, err := r.db.PrepareNamedContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	err = stmt.GetContext(ctx, &entry.UpdatedAt, entry)
	if err != nil {
		return fmt.Errorf("update time entry: %w", err)
	}

	return nil
}

func (r *TimeEntryRepositoryImpl) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM time_entries WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete time entry: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("time entry not found")
	}

	return nil
}

func (r *TimeEntryRepositoryImpl) List(ctx context.Context, filter ports.TimeEntryFilter) ([]*entities.TimeEntry, error) {
	query := `
		SELECT id, user_id, task_id, project_id, start_time, end_time,
			duration_minutes, description, entry_date, billable, hourly_rate,
			created_at, updated_at
		FROM time_entries 
		WHERE 1=1`

	args := []interface{}{}
	argIndex := 1

	// Apply filters
	if filter.UserID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, *filter.UserID)
		argIndex++
	}

	if filter.ProjectID != nil {
		query += fmt.Sprintf(" AND project_id = $%d", argIndex)
		args = append(args, *filter.ProjectID)
		argIndex++
	}

	if filter.TaskID != nil {
		query += fmt.Sprintf(" AND task_id = $%d", argIndex)
		args = append(args, *filter.TaskID)
		argIndex++
	}

	if filter.StartDate != nil {
		query += fmt.Sprintf(" AND entry_date >= $%d", argIndex)
		args = append(args, *filter.StartDate)
		argIndex++
	}

	if filter.EndDate != nil {
		query += fmt.Sprintf(" AND entry_date <= $%d", argIndex)
		args = append(args, *filter.EndDate)
		argIndex++
	}

	if filter.Billable != nil {
		query += fmt.Sprintf(" AND billable = $%d", argIndex)
		args = append(args, *filter.Billable)
		argIndex++
	}

	// Apply sorting
	if filter.SortBy != "" {
		order := "ASC"
		if filter.SortOrder == "desc" {
			order = "DESC"
		}
		query += fmt.Sprintf(" ORDER BY %s %s", filter.SortBy, order)
	} else {
		query += " ORDER BY entry_date DESC, start_time DESC"
	}

	// Apply pagination
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filter.Offset)
	}

	var entries []*entities.TimeEntry
	err := r.db.SelectContext(ctx, &entries, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list time entries: %w", err)
	}

	return entries, nil
}

func (r *TimeEntryRepositoryImpl) GetUserEntries(ctx context.Context, userID uuid.UUID, filter ports.TimeEntryFilter) ([]*entities.TimeEntry, error) {
	filter.UserID = &userID
	return r.List(ctx, filter)
}

func (r *TimeEntryRepositoryImpl) GetProjectEntries(ctx context.Context, projectID int, filter ports.TimeEntryFilter) ([]*entities.TimeEntry, error) {
	filter.ProjectID = &projectID
	return r.List(ctx, filter)
}

func (r *TimeEntryRepositoryImpl) GetTaskEntries(ctx context.Context, taskID int) ([]*entities.TimeEntry, error) {
	query := `
		SELECT id, user_id, task_id, project_id, start_time, end_time,
			duration_minutes, description, entry_date, billable, hourly_rate,
			created_at, updated_at
		FROM time_entries 
		WHERE task_id = $1
		ORDER BY entry_date DESC, start_time DESC`

	var entries []*entities.TimeEntry
	err := r.db.SelectContext(ctx, &entries, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("get task entries: %w", err)
	}

	return entries, nil
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
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no active time entry found")
		}
		return nil, fmt.Errorf("get active entry: %w", err)
	}

	return &entry, nil
}

func (r *TimeEntryRepositoryImpl) GetTotalHoursForPeriod(ctx context.Context, userID uuid.UUID, start, end time.Time) (float64, error) {
	query := `
		SELECT COALESCE(SUM(duration_minutes), 0) / 60.0 as total_hours
		FROM time_entries 
		WHERE user_id = $1 AND entry_date >= $2 AND entry_date <= $3
		AND duration_minutes IS NOT NULL`

	var totalHours float64
	err := r.db.GetContext(ctx, &totalHours, query, userID, start, end)
	if err != nil {
		return 0, fmt.Errorf("get total hours: %w", err)
	}

	return totalHours, nil
}