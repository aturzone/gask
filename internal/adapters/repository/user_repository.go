package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
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
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING created_at, updated_at`

	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	err := r.db.QueryRowContext(ctx, query,
		user.ID, user.Email, user.Username, user.PasswordHash,
		user.FirstName, user.LastName, user.Role, user.IsActive,
		user.WorkingHoursStart, user.WorkingHoursEnd, user.WorkingDays,
		user.Timezone, user.HourlyRate,
	).Scan(&user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

func (r *UserRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	query := `
		SELECT id, email, username, password_hash, first_name, last_name, role, 
			is_active, working_hours_start, working_hours_end, working_days, timezone, 
			hourly_rate, created_at, updated_at, deleted_at
		FROM users 
		WHERE id = $1 AND deleted_at IS NULL`

	var user entities.User
	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entities.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}

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
		SET email = $2, username = $3, first_name = $4, last_name = $5, role = $6, 
			is_active = $7, working_hours_start = $8, working_hours_end = $9,
			working_days = $10, timezone = $11, hourly_rate = $12, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at`

	err := r.db.QueryRowContext(ctx, query,
		user.ID, user.Email, user.Username, user.FirstName, user.LastName, user.Role,
		user.IsActive, user.WorkingHoursStart, user.WorkingHoursEnd, user.WorkingDays,
		user.Timezone, user.HourlyRate,
	).Scan(&user.UpdatedAt)

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
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT 20`

	var users []*entities.User
	err := r.db.SelectContext(ctx, &users, query)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	return users, nil
}

func (r *UserRepositoryImpl) Count(ctx context.Context, filter ports.UserFilter) (int64, error) {
	query := `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`

	var count int64
	err := r.db.GetContext(ctx, &count, query)
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
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`

	err := r.db.QueryRowContext(ctx, query, 
		skill.UserID, skill.SkillName, skill.ProficiencyLevel, 
		skill.YearsOfExperience, skill.IsCertified,
	).Scan(&skill.ID, &skill.CreatedAt)

	if err != nil {
		return fmt.Errorf("add user skill: %w", err)
	}

	return nil
}

func (r *UserRepositoryImpl) UpdateUserSkill(ctx context.Context, skill *entities.UserSkill) error {
	query := `
		UPDATE user_skills 
		SET proficiency_level = $2, years_of_experience = $3, is_certified = $4
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, 
		skill.ID, skill.ProficiencyLevel, skill.YearsOfExperience, skill.IsCertified)
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
