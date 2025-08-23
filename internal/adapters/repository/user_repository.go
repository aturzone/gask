package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/taskmaster/core/internal/domain/entities"
	"github.com/taskmaster/core/internal/ports"
)

// UserRepository implements the user repository interface
type UserRepository struct {
	db *sqlx.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *entities.User) (*entities.User, error) {
	query := `
		INSERT INTO users (id, email, username, password_hash, first_name, last_name, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, email, username, password_hash, first_name, last_name, role, is_active, last_login_at, created_at, updated_at
	`

	var createdUser entities.User
	row := r.db.QueryRowContext(ctx, query,
		user.ID,
		user.Email,
		user.Username,
		user.PasswordHash,
		user.FirstName,
		user.LastName,
		user.Role,
		user.IsActive,
		user.CreatedAt,
		user.UpdatedAt,
	)

	err := row.Scan(
		&createdUser.ID,
		&createdUser.Email,
		&createdUser.Username,
		&createdUser.PasswordHash,
		&createdUser.FirstName,
		&createdUser.LastName,
		&createdUser.Role,
		&createdUser.IsActive,
		&createdUser.LastLoginAt,
		&createdUser.CreatedAt,
		&createdUser.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &createdUser, nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	query := `
		SELECT id, email, username, password_hash, first_name, last_name, role, is_active, last_login_at, created_at, updated_at
		FROM users WHERE id = $1
	`

	var user entities.User
	row := r.db.QueryRowContext(ctx, query, id)

	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.Role,
		&user.IsActive,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	query := `
		SELECT id, email, username, password_hash, first_name, last_name, role, is_active, last_login_at, created_at, updated_at
		FROM users WHERE email = $1
	`

	var user entities.User
	row := r.db.QueryRowContext(ctx, query, email)

	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.Role,
		&user.IsActive,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetByUsername retrieves a user by username
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*entities.User, error) {
	query := `
		SELECT id, email, username, password_hash, first_name, last_name, role, is_active, last_login_at, created_at, updated_at
		FROM users WHERE username = $1
	`

	var user entities.User
	row := r.db.QueryRowContext(ctx, query, username)

	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.Role,
		&user.IsActive,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// Update updates a user
func (r *UserRepository) Update(ctx context.Context, user *entities.User) (*entities.User, error) {
	query := `
		UPDATE users 
		SET email = $2, username = $3, password_hash = $4, first_name = $5, last_name = $6, role = $7, is_active = $8, updated_at = $9
		WHERE id = $1
		RETURNING id, email, username, password_hash, first_name, last_name, role, is_active, last_login_at, created_at, updated_at
	`

	var updatedUser entities.User
	row := r.db.QueryRowContext(ctx, query,
		user.ID,
		user.Email,
		user.Username,
		user.PasswordHash,
		user.FirstName,
		user.LastName,
		user.Role,
		user.IsActive,
		user.UpdatedAt,
	)

	err := row.Scan(
		&updatedUser.ID,
		&updatedUser.Email,
		&updatedUser.Username,
		&updatedUser.PasswordHash,
		&updatedUser.FirstName,
		&updatedUser.LastName,
		&updatedUser.Role,
		&updatedUser.IsActive,
		&updatedUser.LastLoginAt,
		&updatedUser.CreatedAt,
		&updatedUser.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return &updatedUser, nil
}

// Delete deletes a user
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// List retrieves users with filtering and pagination
func (r *UserRepository) List(ctx context.Context, filter ports.UserFilter) ([]*entities.User, int, error) {
	// Build WHERE clause
	var conditions []string
	var args []interface{}
	argIndex := 1

	if filter.Role != nil {
		conditions = append(conditions, fmt.Sprintf("role = $%d", argIndex))
		args = append(args, *filter.Role)
		argIndex++
	}

	if filter.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *filter.IsActive)
		argIndex++
	}

	if filter.Search != nil && *filter.Search != "" {
		searchPattern := "%" + *filter.Search + "%"
		conditions = append(conditions, fmt.Sprintf("(username ILIKE $%d OR email ILIKE $%d OR first_name ILIKE $%d OR last_name ILIKE $%d)", argIndex, argIndex, argIndex, argIndex))
		args = append(args, searchPattern)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users %s", whereClause)
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
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
		SELECT id, email, username, password_hash, first_name, last_name, role, is_active, last_login_at, created_at, updated_at
		FROM users %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, sortOrder, argIndex, argIndex+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*entities.User
	for rows.Next() {
		var user entities.User
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Username,
			&user.PasswordHash,
			&user.FirstName,
			&user.LastName,
			&user.Role,
			&user.IsActive,
			&user.LastLoginAt,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("row iteration error: %w", err)
	}

	return users, total, nil
}

// UpdateLastLogin updates the last login time for a user
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID, loginTime time.Time) error {
	query := `UPDATE users SET last_login_at = $2 WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id, loginTime)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	return nil
}
