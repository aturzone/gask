package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/taskmaster/core/internal/domain/entities"
	"github.com/taskmaster/core/internal/infrastructure/logger"
	"github.com/taskmaster/core/internal/ports"
)

// UserService handles user-related operations
type UserService struct {
	userRepo ports.UserRepository
	logger   *logger.Logger
}

// NewUserService creates a new user service
func NewUserService(userRepo ports.UserRepository, logger *logger.Logger) *UserService {
	return &UserService{
		userRepo: userRepo,
		logger:   logger,
	}
}

// CreateUser creates a new user
func (s *UserService) CreateUser(ctx context.Context, req ports.CreateUserRequest) (*entities.User, error) {
	// Check if user already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("user with email %s already exists", req.Email)
	}

	existingUser, err = s.userRepo.GetByUsername(ctx, req.Username)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("user with username %s already exists", req.Username)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user entity
	user := &entities.User{
		ID:           uuid.New(),
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Role:         req.Role,
		IsActive:     req.IsActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	createdUser, err := s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.Info("User created successfully", "user_id", createdUser.ID, "email", createdUser.Email)

	// Remove password hash from response
	createdUser.PasswordHash = ""

	return createdUser, nil
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Remove password hash from response
	user.PasswordHash = ""

	return user, nil
}

// UpdateUser updates a user's information
func (s *UserService) UpdateUser(ctx context.Context, id uuid.UUID, req ports.UpdateUserRequest) (*entities.User, error) {
	// Get existing user
	existingUser, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Check if email is being changed and if it's already taken
	if req.Email != nil && *req.Email != existingUser.Email {
		emailUser, err := s.userRepo.GetByEmail(ctx, *req.Email)
		if err == nil && emailUser != nil && emailUser.ID != id {
			return nil, fmt.Errorf("email %s is already taken", *req.Email)
		}
	}

	// Check if username is being changed and if it's already taken
	if req.Username != nil && *req.Username != existingUser.Username {
		usernameUser, err := s.userRepo.GetByUsername(ctx, *req.Username)
		if err == nil && usernameUser != nil && usernameUser.ID != id {
			return nil, fmt.Errorf("username %s is already taken", *req.Username)
		}
	}

	// Update fields
	if req.Email != nil {
		existingUser.Email = *req.Email
	}
	if req.Username != nil {
		existingUser.Username = *req.Username
	}
	if req.FirstName != nil {
		existingUser.FirstName = req.FirstName
	}
	if req.LastName != nil {
		existingUser.LastName = req.LastName
	}
	if req.Role != nil {
		existingUser.Role = *req.Role
	}
	if req.IsActive != nil {
		existingUser.IsActive = *req.IsActive
	}

	existingUser.UpdatedAt = time.Now()

	updatedUser, err := s.userRepo.Update(ctx, existingUser)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	s.logger.Info("User updated successfully", "user_id", updatedUser.ID, "email", updatedUser.Email)

	// Remove password hash from response
	updatedUser.PasswordHash = ""

	return updatedUser, nil
}

// DeleteUser deletes a user
func (s *UserService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	// Check if user exists
	_, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	err = s.userRepo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	s.logger.Info("User deleted successfully", "user_id", id)

	return nil
}

// ListUsers retrieves users with filtering and pagination
func (s *UserService) ListUsers(ctx context.Context, filter ports.UserFilter) ([]*entities.User, int, error) {
	users, total, err := s.userRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	// Remove password hashes from all users
	for _, user := range users {
		user.PasswordHash = ""
	}

	return users, total, nil
}

// ChangePassword changes a user's password
func (s *UserService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	// Get user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Verify old password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword))
	if err != nil {
		return fmt.Errorf("invalid old password")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update password
	user.PasswordHash = string(hashedPassword)
	user.UpdatedAt = time.Now()

	_, err = s.userRepo.Update(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	s.logger.Info("Password changed successfully", "user_id", userID)

	return nil
}

// GetUserProfile retrieves a user's profile (same as GetUser but with explicit purpose)
func (s *UserService) GetUserProfile(ctx context.Context, userID uuid.UUID) (*entities.User, error) {
	return s.GetUser(ctx, userID)
}
