package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
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
	query := `
		INSERT INTO projects (name, description, status, start_date, end_date, manager_id, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		project.Name,
		project.Description,
		project.Status,
		project.StartDate,
		project.EndDate,
		project.ManagerID,
		project.CreatedBy,
		project.CreatedAt,
		project.UpdatedAt,
	).Scan(&project.ID, &project.CreatedAt, &project.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	return project, nil
}

// GetByID retrieves a project by ID
func (r *ProjectRepository) GetByID(ctx context.Context, id int) (*entities.Project, error) {
	query := `
		SELECT id, name, description, status, start_date, end_date, manager_id, created_by, created_at, updated_at
		FROM projects WHERE id = $1
	`

	var project entities.Project
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&project.ID,
		&project.Name,
		&project.Description,
		&project.Status,
		&project.StartDate,
		&project.EndDate,
		&project.ManagerID,
		&project.CreatedBy,
		&project.CreatedAt,
		&project.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("project not found")
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return &project, nil
}

// Update updates a project
func (r *ProjectRepository) Update(ctx context.Context, project *entities.Project) (*entities.Project, error) {
	query := `
		UPDATE projects 
		SET name = $2, description = $3, status = $4, start_date = $5, end_date = $6, manager_id = $7, updated_at = $8
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		project.ID,
		project.Name,
		project.Description,
		project.Status,
		project.StartDate,
		project.EndDate,
		project.ManagerID,
		project.UpdatedAt,
	).Scan(&project.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	return project, nil
}

// Delete deletes a project
func (r *ProjectRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM projects WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("project not found")
	}

	return nil
}

// List retrieves projects with filtering and pagination
func (r *ProjectRepository) List(ctx context.Context, filter ports.ProjectFilter) ([]*entities.Project, int, error) {
	// Build WHERE clause
	var conditions []string
	var args []interface{}
	argIndex := 1

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *filter.Status)
		argIndex++
	}

	if filter.ManagerID != nil {
		conditions = append(conditions, fmt.Sprintf("manager_id = $%d", argIndex))
		args = append(args, *filter.ManagerID)
		argIndex++
	}

	if filter.Search != nil && *filter.Search != "" {
		searchPattern := "%" + *filter.Search + "%"
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", argIndex, argIndex))
		args = append(args, searchPattern)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM projects %s", whereClause)
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count projects: %w", err)
	}

	// Build main query
	orderBy := "created_at"
	if filter.SortBy != "" {
		orderBy = filter.SortBy
	}

	sortOrder := "DESC"
	if filter.SortOrder != "" {
		sortOrder = strings.ToUpper(filter.SortOrder)
	}

	query := fmt.Sprintf(`
		SELECT id, name, description, status, start_date, end_date, manager_id, created_by, created_at, updated_at
		FROM projects %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, sortOrder, argIndex, argIndex+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	var projects []*entities.Project
	for rows.Next() {
		var project entities.Project
		err := rows.Scan(
			&project.ID,
			&project.Name,
			&project.Description,
			&project.Status,
			&project.StartDate,
			&project.EndDate,
			&project.ManagerID,
			&project.CreatedBy,
			&project.CreatedAt,
			&project.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan project: %w", err)
		}
		projects = append(projects, &project)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("row iteration error: %w", err)
	}

	return projects, total, nil
}

// GetUserProjects gets all projects associated with a user
func (r *ProjectRepository) GetUserProjects(ctx context.Context, userID uuid.UUID) ([]*entities.Project, error) {
	query := `
		SELECT p.id, p.name, p.description, p.status, p.start_date, p.end_date, p.manager_id, p.created_by, p.created_at, p.updated_at
		FROM projects p
		LEFT JOIN project_members pm ON p.id = pm.project_id
		WHERE p.manager_id = $1 OR pm.user_id = $1
		ORDER BY p.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user projects: %w", err)
	}
	defer rows.Close()

	var projects []*entities.Project
	for rows.Next() {
		var project entities.Project
		err := rows.Scan(
			&project.ID,
			&project.Name,
			&project.Description,
			&project.Status,
			&project.StartDate,
			&project.EndDate,
			&project.ManagerID,
			&project.CreatedBy,
			&project.CreatedAt,
			&project.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}
		projects = append(projects, &project)
	}

	return projects, nil
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
	query := `
		INSERT INTO tasks (title, description, status, priority, project_id, assignee_id, created_by, due_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		task.Title,
		task.Description,
		task.Status,
		task.Priority,
		task.ProjectID,
		task.AssigneeID,
		task.CreatedBy,
		task.DueDate,
		task.CreatedAt,
		task.UpdatedAt,
	).Scan(&task.ID, &task.CreatedAt, &task.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	return task, nil
}

// GetByID retrieves a task by ID
func (r *TaskRepository) GetByID(ctx context.Context, id int) (*entities.Task, error) {
	query := `
		SELECT id, title, description, status, priority, project_id, assignee_id, created_by, due_date, created_at, updated_at
		FROM tasks WHERE id = $1
	`

	var task entities.Task
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&task.Status,
		&task.Priority,
		&task.ProjectID,
		&task.AssigneeID,
		&task.CreatedBy,
		&task.DueDate,
		&task.CreatedAt,
		&task.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task not found")
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	return &task, nil
}

// Update updates a task
func (r *TaskRepository) Update(ctx context.Context, task *entities.Task) (*entities.Task, error) {
	query := `
		UPDATE tasks 
		SET title = $2, description = $3, status = $4, priority = $5, assignee_id = $6, due_date = $7, updated_at = $8
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		task.ID,
		task.Title,
		task.Description,
		task.Status,
		task.Priority,
		task.AssigneeID,
		task.DueDate,
		task.UpdatedAt,
	).Scan(&task.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	return task, nil
}

// Delete deletes a task
func (r *TaskRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM tasks WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task not found")
	}

	return nil
}

// List retrieves tasks with filtering and pagination
func (r *TaskRepository) List(ctx context.Context, filter ports.TaskFilter) ([]*entities.Task, int, error) {
	// Build WHERE clause
	var conditions []string
	var args []interface{}
	argIndex := 1

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *filter.Status)
		argIndex++
	}

	if filter.Priority != nil {
		conditions = append(conditions, fmt.Sprintf("priority = $%d", argIndex))
		args = append(args, *filter.Priority)
		argIndex++
	}

	if filter.ProjectID != nil {
		conditions = append(conditions, fmt.Sprintf("project_id = $%d", argIndex))
		args = append(args, *filter.ProjectID)
		argIndex++
	}

	if filter.AssigneeID != nil {
		conditions = append(conditions, fmt.Sprintf("assignee_id = $%d", argIndex))
		args = append(args, *filter.AssigneeID)
		argIndex++
	}

	if filter.Search != nil && *filter.Search != "" {
		searchPattern := "%" + *filter.Search + "%"
		conditions = append(conditions, fmt.Sprintf("(title ILIKE $%d OR description ILIKE $%d)", argIndex, argIndex))
		args = append(args, searchPattern)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tasks %s", whereClause)
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	// Build main query
	orderBy := "created_at"
	if filter.SortBy != "" {
		orderBy = filter.SortBy
	}

	sortOrder := "DESC"
	if filter.SortOrder != "" {
		sortOrder = strings.ToUpper(filter.SortOrder)
	}

	query := fmt.Sprintf(`
		SELECT id, title, description, status, priority, project_id, assignee_id, created_by, due_date, created_at, updated_at
		FROM tasks %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, sortOrder, argIndex, argIndex+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*entities.Task
	for rows.Next() {
		var task entities.Task
		err := rows.Scan(
			&task.ID,
			&task.Title,
			&task.Description,
			&task.Status,
			&task.Priority,
			&task.ProjectID,
			&task.AssigneeID,
			&task.CreatedBy,
			&task.DueDate,
			&task.CreatedAt,
			&task.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, &task)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("row iteration error: %w", err)
	}

	return tasks, total, nil
}

// GetByProject retrieves tasks by project ID
func (r *TaskRepository) GetByProject(ctx context.Context, projectID int) ([]*entities.Task, error) {
	query := `
		SELECT id, title, description, status, priority, project_id, assignee_id, created_by, due_date, created_at, updated_at
		FROM tasks WHERE project_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks by project: %w", err)
	}
	defer rows.Close()

	var tasks []*entities.Task
	for rows.Next() {
		var task entities.Task
		err := rows.Scan(
			&task.ID,
			&task.Title,
			&task.Description,
			&task.Status,
			&task.Priority,
			&task.ProjectID,
			&task.AssigneeID,
			&task.CreatedBy,
			&task.DueDate,
			&task.CreatedAt,
			&task.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, &task)
	}

	return tasks, nil
}

// GetByAssignee retrieves tasks by assignee ID
func (r *TaskRepository) GetByAssignee(ctx context.Context, userID uuid.UUID) ([]*entities.Task, error) {
	query := `
		SELECT id, title, description, status, priority, project_id, assignee_id, created_by, due_date, created_at, updated_at
		FROM tasks WHERE assignee_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks by assignee: %w", err)
	}
	defer rows.Close()

	var tasks []*entities.Task
	for rows.Next() {
		var task entities.Task
		err := rows.Scan(
			&task.ID,
			&task.Title,
			&task.Description,
			&task.Status,
			&task.Priority,
			&task.ProjectID,
			&task.AssigneeID,
			&task.CreatedBy,
			&task.DueDate,
			&task.CreatedAt,
			&task.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, &task)
	}

	return tasks, nil
}

// GetOverdue retrieves overdue tasks
func (r *TaskRepository) GetOverdue(ctx context.Context) ([]*entities.Task, error) {
	query := `
		SELECT id, title, description, status, priority, project_id, assignee_id, created_by, due_date, created_at, updated_at
		FROM tasks 
		WHERE due_date < NOW() AND status NOT IN ('done', 'cancelled')
		ORDER BY due_date ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get overdue tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*entities.Task
	for rows.Next() {
		var task entities.Task
		err := rows.Scan(
			&task.ID,
			&task.Title,
			&task.Description,
			&task.Status,
			&task.Priority,
			&task.ProjectID,
			&task.AssigneeID,
			&task.CreatedBy,
			&task.DueDate,
			&task.CreatedAt,
			&task.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, &task)
	}

	return tasks, nil
}

// GetNearDeadline retrieves tasks with approaching deadlines
func (r *TaskRepository) GetNearDeadline(ctx context.Context, days int) ([]*entities.Task, error) {
	query := `
		SELECT id, title, description, status, priority, project_id, assignee_id, created_by, due_date, created_at, updated_at
		FROM tasks 
		WHERE due_date BETWEEN NOW() AND NOW() + INTERVAL '%d days' 
		AND status NOT IN ('done', 'cancelled')
		ORDER BY due_date ASC
	`

	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(query, days))
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks near deadline: %w", err)
	}
	defer rows.Close()

	var tasks []*entities.Task
	for rows.Next() {
		var task entities.Task
		err := rows.Scan(
			&task.ID,
			&task.Title,
			&task.Description,
			&task.Status,
			&task.Priority,
			&task.ProjectID,
			&task.AssigneeID,
			&task.CreatedBy,
			&task.DueDate,
			&task.CreatedAt,
			&task.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, &task)
	}

	return tasks, nil
}

// UpdateStatus updates a task's status
func (r *TaskRepository) UpdateStatus(ctx context.Context, id int, status entities.TaskStatus) error {
	query := `UPDATE tasks SET status = $2, updated_at = NOW() WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task not found")
	}

	return nil
}

// Assign assigns a task to a user
func (r *TaskRepository) Assign(ctx context.Context, id int, assigneeID uuid.UUID) error {
	query := `UPDATE tasks SET assignee_id = $2, updated_at = NOW() WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id, assigneeID)
	if err != nil {
		return fmt.Errorf("failed to assign task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task not found")
	}

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
	query := `
		INSERT INTO time_entries (id, task_id, user_id, description, hours, start_time, end_time, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		entry.ID,
		entry.TaskID,
		entry.UserID,
		entry.Description,
		entry.Hours,
		entry.StartTime,
		entry.EndTime,
		entry.CreatedAt,
		entry.UpdatedAt,
	).Scan(&entry.CreatedAt, &entry.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create time entry: %w", err)
	}

	return entry, nil
}

// GetByID retrieves a time entry by ID
func (r *TimeEntryRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.TimeEntry, error) {
	query := `
		SELECT id, task_id, user_id, description, hours, start_time, end_time, created_at, updated_at
		FROM time_entries WHERE id = $1
	`

	var entry entities.TimeEntry
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&entry.ID,
		&entry.TaskID,
		&entry.UserID,
		&entry.Description,
		&entry.Hours,
		&entry.StartTime,
		&entry.EndTime,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("time entry not found")
		}
		return nil, fmt.Errorf("failed to get time entry: %w", err)
	}

	return &entry, nil
}

// Update updates a time entry
func (r *TimeEntryRepository) Update(ctx context.Context, entry *entities.TimeEntry) (*entities.TimeEntry, error) {
	query := `
		UPDATE time_entries 
		SET description = $2, hours = $3, start_time = $4, end_time = $5, updated_at = $6
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		entry.ID,
		entry.Description,
		entry.Hours,
		entry.StartTime,
		entry.EndTime,
		entry.UpdatedAt,
	).Scan(&entry.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to update time entry: %w", err)
	}

	return entry, nil
}

// Delete deletes a time entry
func (r *TimeEntryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM time_entries WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete time entry: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("time entry not found")
	}

	return nil
}

// List retrieves time entries with filtering and pagination
func (r *TimeEntryRepository) List(ctx context.Context, filter ports.TimeEntryFilter) ([]*entities.TimeEntry, int, error) {
	// Build WHERE clause
	var conditions []string
	var args []interface{}
	argIndex := 1

	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIndex))
		args = append(args, *filter.UserID)
		argIndex++
	}

	if filter.TaskID != nil {
		conditions = append(conditions, fmt.Sprintf("task_id = $%d", argIndex))
		args = append(args, *filter.TaskID)
		argIndex++
	}

	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("start_time >= $%d", argIndex))
		args = append(args, *filter.StartDate)
		argIndex++
	}

	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("start_time <= $%d", argIndex))
		args = append(args, *filter.EndDate)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM time_entries %s", whereClause)
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count time entries: %w", err)
	}

	// Build main query
	orderBy := "start_time"
	if filter.SortBy != "" {
		orderBy = filter.SortBy
	}

	sortOrder := "DESC"
	if filter.SortOrder != "" {
		sortOrder = strings.ToUpper(filter.SortOrder)
	}

	query := fmt.Sprintf(`
		SELECT id, task_id, user_id, description, hours, start_time, end_time, created_at, updated_at
		FROM time_entries %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, sortOrder, argIndex, argIndex+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list time entries: %w", err)
	}
	defer rows.Close()

	var entries []*entities.TimeEntry
	for rows.Next() {
		var entry entities.TimeEntry
		err := rows.Scan(
			&entry.ID,
			&entry.TaskID,
			&entry.UserID,
			&entry.Description,
			&entry.Hours,
			&entry.StartTime,
			&entry.EndTime,
			&entry.CreatedAt,
			&entry.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan time entry: %w", err)
		}
		entries = append(entries, &entry)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("row iteration error: %w", err)
	}

	return entries, total, nil
}

// GetByUser retrieves time entries by user ID
func (r *TimeEntryRepository) GetByUser(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]*entities.TimeEntry, error) {
	query := `
		SELECT id, task_id, user_id, description, hours, start_time, end_time, created_at, updated_at
		FROM time_entries 
		WHERE user_id = $1 AND start_time BETWEEN $2 AND $3
		ORDER BY start_time DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get time entries by user: %w", err)
	}
	defer rows.Close()

	var entries []*entities.TimeEntry
	for rows.Next() {
		var entry entities.TimeEntry
		err := rows.Scan(
			&entry.ID,
			&entry.TaskID,
			&entry.UserID,
			&entry.Description,
			&entry.Hours,
			&entry.StartTime,
			&entry.EndTime,
			&entry.CreatedAt,
			&entry.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan time entry: %w", err)
		}
		entries = append(entries, &entry)
	}

	return entries, nil
}

// GetByTask retrieves time entries by task ID
func (r *TimeEntryRepository) GetByTask(ctx context.Context, taskID int) ([]*entities.TimeEntry, error) {
	query := `
		SELECT id, task_id, user_id, description, hours, start_time, end_time, created_at, updated_at
		FROM time_entries WHERE task_id = $1
		ORDER BY start_time DESC
	`

	rows, err := r.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get time entries by task: %w", err)
	}
	defer rows.Close()

	var entries []*entities.TimeEntry
	for rows.Next() {
		var entry entities.TimeEntry
		err := rows.Scan(
			&entry.ID,
			&entry.TaskID,
			&entry.UserID,
			&entry.Description,
			&entry.Hours,
			&entry.StartTime,
			&entry.EndTime,
			&entry.CreatedAt,
			&entry.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan time entry: %w", err)
		}
		entries = append(entries, &entry)
	}

	return entries, nil
}

// GetByProject retrieves time entries by project ID
func (r *TimeEntryRepository) GetByProject(ctx context.Context, projectID int, from, to time.Time) ([]*entities.TimeEntry, error) {
	query := `
		SELECT te.id, te.task_id, te.user_id, te.description, te.hours, te.start_time, te.end_time, te.created_at, te.updated_at
		FROM time_entries te
		JOIN tasks t ON te.task_id = t.id
		WHERE t.project_id = $1 AND te.start_time BETWEEN $2 AND $3
		ORDER BY te.start_time DESC
	`

	rows, err := r.db.QueryContext(ctx, query, projectID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get time entries by project: %w", err)
	}
	defer rows.Close()

	var entries []*entities.TimeEntry
	for rows.Next() {
		var entry entities.TimeEntry
		err := rows.Scan(
			&entry.ID,
			&entry.TaskID,
			&entry.UserID,
			&entry.Description,
			&entry.Hours,
			&entry.StartTime,
			&entry.EndTime,
			&entry.CreatedAt,
			&entry.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan time entry: %w", err)
		}
		entries = append(entries, &entry)
	}

	return entries, nil
}

// GetActiveEntry retrieves the active time entry for a user
func (r *TimeEntryRepository) GetActiveEntry(ctx context.Context, userID uuid.UUID) (*entities.TimeEntry, error) {
	query := `
		SELECT id, task_id, user_id, description, hours, start_time, end_time, created_at, updated_at
		FROM time_entries 
		WHERE user_id = $1 AND end_time IS NULL
		ORDER BY start_time DESC
		LIMIT 1
	`

	var entry entities.TimeEntry
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&entry.ID,
		&entry.TaskID,
		&entry.UserID,
		&entry.Description,
		&entry.Hours,
		&entry.StartTime,
		&entry.EndTime,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no active entry found")
		}
		return nil, fmt.Errorf("failed to get active entry: %w", err)
	}

	return &entry, nil
}

// GetTotalHours calculates total hours for a user in a time range
func (r *TimeEntryRepository) GetTotalHours(ctx context.Context, userID uuid.UUID, from, to time.Time) (float64, error) {
	query := `
		SELECT COALESCE(SUM(hours), 0) as total_hours
		FROM time_entries 
		WHERE user_id = $1 AND start_time BETWEEN $2 AND $3
	`

	var totalHours float64
	err := r.db.QueryRowContext(ctx, query, userID, from, to).Scan(&totalHours)
	if err != nil {
		return 0, fmt.Errorf("failed to get total hours: %w", err)
	}

	return totalHours, nil
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
		if err == sql.ErrNoRows {
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
