package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/taskmaster/core/internal/domain/entities"
	"github.com/taskmaster/core/internal/ports"
)

// UserRepositoryImpl implements the UserRepository interface
type UserRepositoryImpl struct {
	db *sqlx.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sqlx.DB) ports.UserRepository {
	return &UserRepositoryImpl{db: db}
}

func (r *UserRepositoryImpl) Create(ctx context.Context, user *entities.User) error {
	query := `
		INSERT INTO users (id, email, username, password_hash, first_name, last_name, role, 
			is_active, working_hours_start, working_hours_end, working_days, timezone, hourly_rate)
		VALUES (:id, :email, :username, :password_hash, :first_name, :last_name, :role,
			:is_active, :working_hours_start, :working_hours_end, :working_days, :timezone, :hourly_rate)
		RETURNING created_at, updated_at`

	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	stmt, err := r.db.PrepareNamedContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	err = stmt.GetContext(ctx, user, user)
	if err != nil {
		return fmt.Errorf("execute query: %w", err)
	}

	return nil
}

func (r *UserRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	query := `
		SELECT u.*, 
			COALESCE(
				json_agg(
					json_build_object(
						'id', us.id,
						'user_id', us.user_id,
						'skill_name', us.skill_name,
						'proficiency_level', us.proficiency_level,
						'years_of_experience', us.years_of_experience,
						'is_certified', us.is_certified,
						'created_at', us.created_at
					)
				) FILTER (WHERE us.id IS NOT NULL), 
				'[]'
			) as skills
		FROM users u
		LEFT JOIN user_skills us ON u.id = us.user_id
		WHERE u.id = $1 AND u.deleted_at IS NULL
		GROUP BY u.id`

	var user entities.User
	var skillsJSON string

	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entities.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	// Parse skills JSON if needed (simplified for this example)
	// In a real implementation, you'd properly unmarshal the JSON

	return &user, nil
}

func (r *UserRepositoryImpl) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	query := `
		SELECT id, email, username, password_hash, first_name, last_name, role, 
			is_active, working_hours_start, working_hours_end, working_days, timezone, 
			hourly_rate, created_at, updated_at, deleted_at
		FROM users 
		WHERE email = $1 AND deleted_at IS NULL`

	var user entities.User
	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entities.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	return &user, nil
}

func (r *UserRepositoryImpl) GetByUsername(ctx context.Context, username string) (*entities.User, error) {
	query := `
		SELECT id, email, username, password_hash, first_name, last_name, role, 
			is_active, working_hours_start, working_hours_end, working_days, timezone, 
			hourly_rate, created_at, updated_at, deleted_at
		FROM users 
		WHERE username = $1 AND deleted_at IS NULL`

	var user entities.User
	err := r.db.GetContext(ctx, &user, query, username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entities.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by username: %w", err)
	}

	return &user, nil
}

func (r *UserRepositoryImpl) Update(ctx context.Context, user *entities.User) error {
	query := `
		UPDATE users 
		SET email = :email, username = :username, first_name = :first_name, 
			last_name = :last_name, role = :role, is_active = :is_active,
			working_hours_start = :working_hours_start, working_hours_end = :working_hours_end,
			working_days = :working_days, timezone = :timezone, hourly_rate = :hourly_rate,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = :id AND deleted_at IS NULL
		RETURNING updated_at`

	stmt, err := r.db.PrepareNamedContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	err = stmt.GetContext(ctx, &user.UpdatedAt, user)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	return nil
}

func (r *UserRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET deleted_at = CURRENT_TIMESTAMP WHERE id = $1 AND deleted_at IS NULL`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return entities.ErrUserNotFound
	}

	return nil
}

func (r *UserRepositoryImpl) List(ctx context.Context, filter ports.UserFilter) ([]*entities.User, error) {
	query := `
		SELECT id, email, username, first_name, last_name, role, is_active,
			working_hours_start, working_hours_end, working_days, timezone, 
			hourly_rate, created_at, updated_at
		FROM users 
		WHERE deleted_at IS NULL`

	args := []interface{}{}
	argIndex := 1

	// Apply filters
	if filter.Role != nil {
		query += fmt.Sprintf(" AND role = $%d", argIndex)
		args = append(args, *filter.Role)
		argIndex++
	}

	if filter.IsActive != nil {
		query += fmt.Sprintf(" AND is_active = $%d", argIndex)
		args = append(args, *filter.IsActive)
		argIndex++
	}

	if filter.Search != nil && *filter.Search != "" {
		query += fmt.Sprintf(" AND (email ILIKE $%d OR username ILIKE $%d OR first_name ILIKE $%d OR last_name ILIKE $%d)", 
			argIndex, argIndex, argIndex, argIndex)
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

	var users []*entities.User
	err := r.db.SelectContext(ctx, &users, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	return users, nil
}

func (r *UserRepositoryImpl) Count(ctx context.Context, filter ports.UserFilter) (int64, error) {
	query := `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`

	args := []interface{}{}
	argIndex := 1

	// Apply same filters as List method
	if filter.Role != nil {
		query += fmt.Sprintf(" AND role = $%d", argIndex)
		args = append(args, *filter.Role)
		argIndex++
	}

	if filter.IsActive != nil {
		query += fmt.Sprintf(" AND is_active = $%d", argIndex)
		args = append(args, *filter.IsActive)
		argIndex++
	}

	if filter.Search != nil && *filter.Search != "" {
		query += fmt.Sprintf(" AND (email ILIKE $%d OR username ILIKE $%d OR first_name ILIKE $%d OR last_name ILIKE $%d)", 
			argIndex, argIndex, argIndex, argIndex)
		searchTerm := "%" + *filter.Search + "%"
		args = append(args, searchTerm)
	}

	var count int64
	err := r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}

	return count, nil
}

func (r *UserRepositoryImpl) GetUserSkills(ctx context.Context, userID uuid.UUID) ([]entities.UserSkill, error) {
	query := `
		SELECT id, user_id, skill_name, proficiency_level, years_of_experience, 
			is_certified, created_at
		FROM user_skills 
		WHERE user_id = $1
		ORDER BY skill_name`

	var skills []entities.UserSkill
	err := r.db.SelectContext(ctx, &skills, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get user skills: %w", err)
	}

	return skills, nil
}

func (r *UserRepositoryImpl) AddUserSkill(ctx context.Context, skill *entities.UserSkill) error {
	query := `
		INSERT INTO user_skills (user_id, skill_name, proficiency_level, years_of_experience, is_certified)
		VALUES (:user_id, :skill_name, :proficiency_level, :years_of_experience, :is_certified)
		RETURNING id, created_at`

	stmt, err := r.db.PrepareNamedContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	err = stmt.GetContext(ctx, skill, skill)
	if err != nil {
		return fmt.Errorf("add user skill: %w", err)
	}

	return nil
}

func (r *UserRepositoryImpl) UpdateUserSkill(ctx context.Context, skill *entities.UserSkill) error {
	query := `
		UPDATE user_skills 
		SET proficiency_level = :proficiency_level, years_of_experience = :years_of_experience, 
			is_certified = :is_certified
		WHERE id = :id`

	_, err := r.db.NamedExecContext(ctx, query, skill)
	if err != nil {
		return fmt.Errorf("update user skill: %w", err)
	}

	return nil
}

func (r *UserRepositoryImpl) RemoveUserSkill(ctx context.Context, userID uuid.UUID, skillName string) error {
	query := `DELETE FROM user_skills WHERE user_id = $1 AND skill_name = $2`
	
	result, err := r.db.ExecContext(ctx, query, userID, skillName)
	if err != nil {
		return fmt.Errorf("remove user skill: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("skill not found")
	}

	return nil
}

// TaskRepositoryImpl implements the TaskRepository interface
type TaskRepositoryImpl struct {
	db *sqlx.DB
}

// NewTaskRepository creates a new task repository
func NewTaskRepository(db *sqlx.DB) ports.TaskRepository {
	return &TaskRepositoryImpl{db: db}
}

func (r *TaskRepositoryImpl) Create(ctx context.Context, task *entities.Task) error {
	query := `
		INSERT INTO tasks (project_id, parent_task_id, title, description, status, priority, 
			assignee_id, reporter_id, estimated_hours, start_date, due_date, tags)
		VALUES (:project_id, :parent_task_id, :title, :description, :status, :priority,
			:assignee_id, :reporter_id, :estimated_hours, :start_date, :due_date, :tags)
		RETURNING id, created_at, updated_at, version`

	stmt, err := r.db.PrepareNamedContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	err = stmt.GetContext(ctx, task, task)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}

	return nil
}

func (r *TaskRepositoryImpl) GetByID(ctx context.Context, id int) (*entities.Task, error) {
	query := `
		SELECT id, project_id, parent_task_id, title, description, status, priority,
			assignee_id, reporter_id, estimated_hours, actual_hours, start_date, due_date,
			completed_at, tags, created_at, updated_at, version
		FROM tasks 
		WHERE id = $1 AND deleted_at IS NULL`

	var task entities.Task
	err := r.db.GetContext(ctx, &task, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entities.ErrTaskNotFound
		}
		return nil, fmt.Errorf("get task by id: %w", err)
	}

	return &task, nil
}

func (r *TaskRepositoryImpl) Update(ctx context.Context, task *entities.Task) error {
	query := `
		UPDATE tasks 
		SET title = :title, description = :description, status = :status, priority = :priority,
			assignee_id = :assignee_id, estimated_hours = :estimated_hours, 
			start_date = :start_date, due_date = :due_date, completed_at = :completed_at,
			tags = :tags, version = version + 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = :id AND deleted_at IS NULL AND version = :version
		RETURNING updated_at, version`

	stmt, err := r.db.PrepareNamedContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	err = stmt.GetContext(ctx, task, task)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("task not found or version conflict")
		}
		return fmt.Errorf("update task: %w", err)
	}

	return nil
}

func (r *TaskRepositoryImpl) Delete(ctx context.Context, id int) error {
	query := `UPDATE tasks SET deleted_at = CURRENT_TIMESTAMP WHERE id = $1 AND deleted_at IS NULL`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return entities.ErrTaskNotFound
	}

	return nil
}

func (r *TaskRepositoryImpl) List(ctx context.Context, filter ports.TaskFilter) ([]*entities.Task, error) {
	query := `
		SELECT id, project_id, parent_task_id, title, description, status, priority,
			assignee_id, reporter_id, estimated_hours, actual_hours, start_date, due_date,
			completed_at, tags, created_at, updated_at, version
		FROM tasks 
		WHERE deleted_at IS NULL`

	args := []interface{}{}
	argIndex := 1

	// Apply filters
	if filter.ProjectID != nil {
		query += fmt.Sprintf(" AND project_id = $%d", argIndex)
		args = append(args, *filter.ProjectID)
		argIndex++
	}

	if filter.AssigneeID != nil {
		query += fmt.Sprintf(" AND assignee_id = $%d", argIndex)
		args = append(args, *filter.AssigneeID)
		argIndex++
	}

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *filter.Status)
		argIndex++
	}

	if filter.Priority != nil {
		query += fmt.Sprintf(" AND priority = $%d", argIndex)
		args = append(args, *filter.Priority)
		argIndex++
	}

	if filter.DueBefore != nil {
		query += fmt.Sprintf(" AND due_date <= $%d", argIndex)
		args = append(args, *filter.DueBefore)
		argIndex++
	}

	if filter.DueAfter != nil {
		query += fmt.Sprintf(" AND due_date >= $%d", argIndex)
		args = append(args, *filter.DueAfter)
		argIndex++
	}

	if len(filter.Tags) > 0 {
		query += fmt.Sprintf(" AND tags && $%d", argIndex)
		args = append(args, pq.Array(filter.Tags))
		argIndex++
	}

	if filter.Search != nil && *filter.Search != "" {
		query += fmt.Sprintf(" AND (title ILIKE $%d OR description ILIKE $%d)", argIndex, argIndex)
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

	var tasks []*entities.Task
	err := r.db.SelectContext(ctx, &tasks, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	return tasks, nil
}

func (r *TaskRepositoryImpl) Count(ctx context.Context, filter ports.TaskFilter) (int64, error) {
	query := `SELECT COUNT(*) FROM tasks WHERE deleted_at IS NULL`

	args := []interface{}{}
	argIndex := 1

	// Apply same filters as List method (simplified)
	if filter.ProjectID != nil {
		query += fmt.Sprintf(" AND project_id = $%d", argIndex)
		args = append(args, *filter.ProjectID)
		argIndex++
	}

	if filter.AssigneeID != nil {
		query += fmt.Sprintf(" AND assignee_id = $%d", argIndex)
		args = append(args, *filter.AssigneeID)
		argIndex++
	}

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *filter.Status)
	}

	var count int64
	err := r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		return 0, fmt.Errorf("count tasks: %w", err)
	}

	return count, nil
}

func (r *TaskRepositoryImpl) GetProjectTasks(ctx context.Context, projectID int, filter ports.TaskFilter) ([]*entities.Task, error) {
	filter.ProjectID = &projectID
	return r.List(ctx, filter)
}

func (r *TaskRepositoryImpl) GetUserTasks(ctx context.Context, userID uuid.UUID, filter ports.TaskFilter) ([]*entities.Task, error) {
	filter.AssigneeID = &userID
	return r.List(ctx, filter)
}

func (r *TaskRepositoryImpl) GetSubtasks(ctx context.Context, parentTaskID int) ([]*entities.Task, error) {
	query := `
		SELECT id, project_id, parent_task_id, title, description, status, priority,
			assignee_id, reporter_id, estimated_hours, actual_hours, start_date, due_date,
			completed_at, tags, created_at, updated_at, version
		FROM tasks 
		WHERE parent_task_id = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC`

	var tasks []*entities.Task
	err := r.db.SelectContext(ctx, &tasks, query, parentTaskID)
	if err != nil {
		return nil, fmt.Errorf("get subtasks: %w", err)
	}

	return tasks, nil
}

func (r *TaskRepositoryImpl) GetTasksNearDeadline(ctx context.Context, days int) ([]*entities.Task, error) {
	query := `
		SELECT id, project_id, parent_task_id, title, description, status, priority,
			assignee_id, reporter_id, estimated_hours, actual_hours, start_date, due_date,
			completed_at, tags, created_at, updated_at, version
		FROM tasks 
		WHERE deleted_at IS NULL 
			AND due_date IS NOT NULL 
			AND due_date <= CURRENT_DATE + INTERVAL '%d days'
			AND due_date >= CURRENT_DATE
			AND status NOT IN ('completed', 'cancelled')
		ORDER BY due_date ASC`

	var tasks []*entities.Task
	err := r.db.SelectContext(ctx, &tasks, fmt.Sprintf(query, days))
	if err != nil {
		return nil, fmt.Errorf("get tasks near deadline: %w", err)
	}

	return tasks, nil
}

func (r *TaskRepositoryImpl) GetOverdueTasks(ctx context.Context) ([]*entities.Task, error) {
	query := `
		SELECT id, project_id, parent_task_id, title, description, status, priority,
			assignee_id, reporter_id, estimated_hours, actual_hours, start_date, due_date,
			completed_at, tags, created_at, updated_at, version
		FROM tasks 
		WHERE deleted_at IS NULL 
			AND due_date IS NOT NULL 
			AND due_date < CURRENT_DATE
			AND status NOT IN ('completed', 'cancelled')
		ORDER BY due_date ASC`

	var tasks []*entities.Task
	err := r.db.SelectContext(ctx, &tasks, query)
	if err != nil {
		return nil, fmt.Errorf("get overdue tasks: %w", err)
	}

	return tasks, nil
}

func (r *TaskRepositoryImpl) BulkUpdateStatus(ctx context.Context, taskIDs []int, status entities.TaskStatus) error {
	if len(taskIDs) == 0 {
		return nil
	}

	query := `
		UPDATE tasks 
		SET status = $1, updated_at = CURRENT_TIMESTAMP, version = version + 1
		WHERE id = ANY($2) AND deleted_at IS NULL`

	_, err := r.db.ExecContext(ctx, query, status, pq.Array(taskIDs))
	if err != nil {
		return fmt.Errorf("bulk update task status: %w", err)
	}

	return nil
}