// internal/domain/entities/entities_test.go
package entities

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUser_CanAssignTask(t *testing.T) {
	tests := []struct {
		name     string
		user     *User
		task     *Task
		expected bool
	}{
		{
			name: "admin can assign task",
			user: &User{
				Role:     UserRoleAdmin,
				IsActive: true,
			},
			task:     &Task{Status: TaskStatusTodo},
			expected: true,
		},
		{
			name: "project manager can assign task",
			user: &User{
				Role:     UserRoleProjectManager,
				IsActive: true,
			},
			task:     &Task{Status: TaskStatusTodo},
			expected: true,
		},
		{
			name: "developer cannot assign task",
			user: &User{
				Role:     UserRoleDeveloper,
				IsActive: true,
			},
			task:     &Task{Status: TaskStatusTodo},
			expected: false,
		},
		{
			name: "inactive user cannot assign task",
			user: &User{
				Role:     UserRoleAdmin,
				IsActive: false,
			},
			task:     &Task{Status: TaskStatusTodo},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.user.CanAssignTask(tt.task)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTask_AssignTo(t *testing.T) {
	userID := uuid.New()
	
	tests := []struct {
		name      string
		task      *Task
		userID    uuid.UUID
		expectErr bool
	}{
		{
			name: "can assign todo task",
			task: &Task{
				Status: TaskStatusTodo,
			},
			userID:    userID,
			expectErr: false,
		},
		{
			name: "cannot assign in-progress task",
			task: &Task{
				Status: TaskStatusInProgress,
			},
			userID:    userID,
			expectErr: true,
		},
		{
			name: "cannot assign completed task",
			task: &Task{
				Status: TaskStatusCompleted,
			},
			userID:    userID,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.task.AssignTo(tt.userID)
			
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, &tt.userID, tt.task.AssigneeID)
			}
		})
	}
}

func TestTask_SetDueDate(t *testing.T) {
	task := &Task{}
	
	t.Run("valid future date", func(t *testing.T) {
		futureDate := time.Now().Add(24 * time.Hour)
		err := task.SetDueDate(futureDate)
		assert.NoError(t, err)
		assert.Equal(t, &futureDate, task.DueDate)
	})
	
	t.Run("past date should fail", func(t *testing.T) {
		pastDate := time.Now().Add(-24 * time.Hour)
		err := task.SetDueDate(pastDate)
		assert.Error(t, err)
		assert.Equal(t, ErrDeadlineInPast, err)
	})
}

func TestProject_IsOverBudget(t *testing.T) {
	tests := []struct {
		name        string
		budget      *float64
		spentBudget float64
		expected    bool
	}{
		{
			name:        "under budget",
			budget:      floatPtr(1000.0),
			spentBudget: 500.0,
			expected:    false,
		},
		{
			name:        "over budget",
			budget:      floatPtr(1000.0),
			spentBudget: 1500.0,
			expected:    true,
		},
		{
			name:        "no budget set",
			budget:      nil,
			spentBudget: 1500.0,
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := &Project{
				Budget:      tt.budget,
				SpentBudget: tt.spentBudget,
			}
			result := project.IsOverBudget()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTimeEntry_CalculateDuration(t *testing.T) {
	startTime := time.Now()
	endTime := startTime.Add(2 * time.Hour)
	
	entry := &TimeEntry{
		StartTime: startTime,
		EndTime:   &endTime,
	}
	
	duration := entry.CalculateDuration()
	assert.Equal(t, 2*time.Hour, duration)
}

func TestTimeEntry_Stop(t *testing.T) {
	entry := &TimeEntry{
		StartTime: time.Now().Add(-30 * time.Minute),
	}
	
	err := entry.Stop()
	assert.NoError(t, err)
	assert.NotNil(t, entry.EndTime)
	assert.NotNil(t, entry.DurationMinutes)
	assert.True(t, *entry.DurationMinutes >= 29) // Allow for test execution time
}

// Helper functions
func floatPtr(f float64) *float64 {
	return &f
}

// =============================================================================
// internal/application/services/auth_service_test.go
package services

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/taskmaster/core/internal/domain/entities"
	"github.com/taskmaster/core/internal/infrastructure/config"
	"github.com/taskmaster/core/internal/infrastructure/logger"
	"github.com/taskmaster/core/internal/ports"
)

// Mock repositories
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.User), args.Error(1)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.User), args.Error(1)
}

// Implement other required methods with empty implementations for this test
func (m *MockUserRepository) Create(ctx context.Context, user *entities.User) error { return nil }
func (m *MockUserRepository) GetByUsername(ctx context.Context, username string) (*entities.User, error) { return nil, nil }
func (m *MockUserRepository) Update(ctx context.Context, user *entities.User) error { return nil }
func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockUserRepository) List(ctx context.Context, filter ports.UserFilter) ([]*entities.User, error) { return nil, nil }
func (m *MockUserRepository) Count(ctx context.Context, filter ports.UserFilter) (int64, error) { return 0, nil }
func (m *MockUserRepository) GetUserSkills(ctx context.Context, userID uuid.UUID) ([]entities.UserSkill, error) { return nil, nil }
func (m *MockUserRepository) AddUserSkill(ctx context.Context, skill *entities.UserSkill) error { return nil }
func (m *MockUserRepository) UpdateUserSkill(ctx context.Context, skill *entities.UserSkill) error { return nil }
func (m *MockUserRepository) RemoveUserSkill(ctx context.Context, userID uuid.UUID, skillName string) error { return nil }

type MockAuthRepository struct {
	mock.Mock
}

func (m *MockAuthRepository) CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	args := m.Called(ctx, userID, tokenHash, expiresAt)
	return args.Error(0)
}

func (m *MockAuthRepository) GetRefreshToken(ctx context.Context, tokenHash string) (*ports.RefreshToken, error) {
	args := m.Called(ctx, tokenHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.RefreshToken), args.Error(1)
}

func (m *MockAuthRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	args := m.Called(ctx, tokenHash)
	return args.Error(0)
}

func (m *MockAuthRepository) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockAuthRepository) CleanupExpiredTokens(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestAuthService_Login(t *testing.T) {
	// Setup
	userRepo := new(MockUserRepository)
	authRepo := new(MockAuthRepository)
	
	jwtConfig := config.JWTConfig{
		Secret:    "test-secret",
		ExpiresIn: 24 * time.Hour,
		Issuer:    "test",
	}
	
	logger, _ := logger.New(config.LoggerConfig{
		Level:  "error", // Reduce noise in tests
		Format: "json",
		Output: "stdout",
	})
	
	authService := NewAuthService(userRepo, authRepo, jwtConfig, logger)

	// Test data
	userID := uuid.New()
	email := "test@example.com"
	password := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	user := &entities.User{
		ID:           userID,
		Email:        email,
		PasswordHash: string(hashedPassword),
		Role:         entities.UserRoleDeveloper,
		IsActive:     true,
	}

	t.Run("successful login", func(t *testing.T) {
		// Mock expectations
		userRepo.On("GetByEmail", mock.Anything, email).Return(user, nil)
		authRepo.On("CreateRefreshToken", mock.Anything, userID, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Return(nil)

		// Execute
		req := LoginRequest{
			Email:    email,
			Password: password,
		}
		
		resp, err := authService.Login(context.Background(), req)

		// Assert
		require.NoError(t, err)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
		assert.Equal(t, user.Email, resp.User.Email)
		assert.Empty(t, resp.User.PasswordHash) // Should be removed

		// Verify JWT token
		token, err := jwt.Parse(resp.AccessToken, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtConfig.Secret), nil
		})
		assert.NoError(t, err)
		assert.True(t, token.Valid)

		userRepo.AssertExpectations(t)
		authRepo.AssertExpectations(t)
	})

	t.Run("invalid password", func(t *testing.T) {
		// Mock expectations
		userRepo.On("GetByEmail", mock.Anything, email).Return(user, nil)

		// Execute
		req := LoginRequest{
			Email:    email,
			Password: "wrongpassword",
		}
		
		resp, err := authService.Login(context.Background(), req)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid credentials")

		userRepo.AssertExpectations(t)
	})

	t.Run("inactive user", func(t *testing.T) {
		inactiveUser := *user
		inactiveUser.IsActive = false

		// Mock expectations
		userRepo.On("GetByEmail", mock.Anything, email).Return(&inactiveUser, nil)

		// Execute
		req := LoginRequest{
			Email:    email,
			Password: password,
		}
		
		resp, err := authService.Login(context.Background(), req)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "account is inactive")

		userRepo.AssertExpectations(t)
	})
}

func TestAuthService_ValidateToken(t *testing.T) {
	jwtConfig := config.JWTConfig{
		Secret:    "test-secret",
		ExpiresIn: 24 * time.Hour,
		Issuer:    "test",
	}
	
	logger, _ := logger.New(config.LoggerConfig{
		Level:  "error",
		Format: "json",
		Output: "stdout",
	})
	
	authService := NewAuthService(nil, nil, jwtConfig, logger)

	userID := uuid.New()
	email := "test@example.com"

	t.Run("valid token", func(t *testing.T) {
		// Create a valid token
		claims := &Claims{
			UserID: userID.String(),
			Email:  email,
			Role:   entities.UserRoleDeveloper,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				Issuer:    jwtConfig.Issuer,
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(jwtConfig.Secret))
		require.NoError(t, err)

		// Validate token
		parsedClaims, err := authService.ValidateToken(tokenString)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, userID.String(), parsedClaims.UserID)
		assert.Equal(t, email, parsedClaims.Email)
		assert.Equal(t, entities.UserRoleDeveloper, parsedClaims.Role)
	})

	t.Run("expired token", func(t *testing.T) {
		// Create an expired token
		claims := &Claims{
			UserID: userID.String(),
			Email:  email,
			Role:   entities.UserRoleDeveloper,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)), // Expired
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
				Issuer:    jwtConfig.Issuer,
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(jwtConfig.Secret))
		require.NoError(t, err)

		// Validate token
		parsedClaims, err := authService.ValidateToken(tokenString)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, parsedClaims)
	})

	t.Run("invalid signature", func(t *testing.T) {
		// Create token with wrong secret
		claims := &Claims{
			UserID: userID.String(),
			Email:  email,
			Role:   entities.UserRoleDeveloper,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				Issuer:    jwtConfig.Issuer,
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte("wrong-secret"))
		require.NoError(t, err)

		// Validate token
		parsedClaims, err := authService.ValidateToken(tokenString)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, parsedClaims)
	})
}

// =============================================================================
// scripts/test.sh
#!/bin/bash

set -e

echo "Running TaskMaster Core Tests..."

# Set test environment
export APP_ENV=test
export DATABASE_HOST=localhost
export DATABASE_PORT=5433
export DATABASE_NAME=taskmaster_test
export DATABASE_USER=postgres
export DATABASE_PASSWORD=postgres
export REDIS_HOST=localhost
export REDIS_PORT=6380
export JWT_SECRET=test-secret

# Start test dependencies
echo "Starting test dependencies..."
docker-compose -f docker-compose.test.yml up -d test-db test-redis

# Wait for services to be ready
echo "Waiting for services to be ready..."
sleep 10

# Run migrations
echo "Running migrations..."
go run cmd/api/main.go migrate up

# Run tests
echo "Running unit tests..."
go test -v -race -coverprofile=coverage.out ./...

# Run integration tests
echo "Running integration tests..."
go test -v -tags=integration ./...

# Generate coverage report
echo "Generating coverage report..."
go tool cover -html=coverage.out -o coverage.html

# Run linting
echo "Running linting..."
golangci-lint run

# Clean up
echo "Cleaning up..."
docker-compose -f docker-compose.test.yml down

echo "All tests completed successfully!"

# =============================================================================
# Makefile
.PHONY: build test clean run docker-build docker-run migrate lint

# Variables
APP_NAME := taskmaster
VERSION := v1.0.0
BUILD_TIME := $(shell date +%Y-%m-%dT%H:%M:%S%z)
GIT_COMMIT := $(shell git rev-parse --short HEAD)

# Build flags
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# Build the application
build:
	@echo "Building $(APP_NAME)..."
	@go build $(LDFLAGS) -o bin/$(APP_NAME) cmd/api/main.go

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(APP_NAME)-linux-amd64 cmd/api/main.go
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(APP_NAME)-darwin-amd64 cmd/api/main.go
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(APP_NAME)-windows-amd64.exe cmd/api/main.go

# Run tests
test:
	@echo "Running tests..."
	@./scripts/test.sh

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

# Run linting
lint:
	@echo "Running linting..."
	@golangci-lint run

# Run the application
run:
	@echo "Running $(APP_NAME)..."
	@go run cmd/api/main.go serve

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

# Database operations
migrate-up:
	@echo "Running migrations..."
	@go run cmd/api/main.go migrate up

migrate-down:
	@echo "Rolling back migrations..."
	@go run cmd/api/main.go migrate down

migrate-status:
	@echo "Migration status..."
	@go run cmd/api/main.go migrate version

# Docker operations
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(APP_NAME):$(VERSION) .
	@docker tag $(APP_NAME):$(VERSION) $(APP_NAME):latest

docker-run:
	@echo "Running with Docker Compose..."
	@docker-compose up -d

docker-stop:
	@echo "Stopping Docker containers..."
	@docker-compose down

docker-logs:
	@echo "Showing Docker logs..."
	@docker-compose logs -f api

# Development setup
dev-setup:
	@echo "Setting up development environment..."
	@go mod download
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/swaggo/swag/cmd/swag@latest
	@cp .env.example .env
	@echo "Development environment setup complete!"

# Generate API documentation
docs:
	@echo "Generating API documentation..."
	@swag init -g cmd/api/main.go -o ./docs

# Deploy to Kubernetes
k8s-deploy:
	@echo "Deploying to Kubernetes..."
	@kubectl apply -f deployments/kubernetes/

k8s-delete:
	@echo "Deleting from Kubernetes..."
	@kubectl delete -f deployments/kubernetes/

# Create user
create-admin:
	@echo "Creating admin user..."
	@go run cmd/api/main.go user create --email admin@taskmaster.dev --password admin123 --role admin --first-name Admin --last-name User

# Help
help:
	@echo "Available commands:"
	@echo "  build         - Build the application"
	@echo "  build-all     - Build for multiple platforms"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  lint          - Run linting"
	@echo "  run           - Run the application"
	@echo "  clean         - Clean build artifacts"
	@echo "  migrate-up    - Run database migrations"
	@echo "  migrate-down  - Rollback database migrations"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run with Docker Compose"
	@echo "  docker-stop   - Stop Docker containers"
	@echo "  dev-setup     - Setup development environment"
	@echo "  docs          - Generate API documentation"
	@echo "  k8s-deploy    - Deploy to Kubernetes"
	@echo "  create-admin  - Create admin user"
	@echo "  help          - Show this help message"