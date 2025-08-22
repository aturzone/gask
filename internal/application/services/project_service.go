package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/taskmaster/core/internal/domain/entities"
	"github.com/taskmaster/core/internal/infrastructure/logger"
	"github.com/taskmaster/core/internal/ports"
)

// ProjectService handles project-related operations
type ProjectService struct {
	projectRepo ports.ProjectRepository
	userRepo    ports.UserRepository
	logger      *logger.Logger
}

// NewProjectService creates a new project service
func NewProjectService(
	projectRepo ports.ProjectRepository,
	userRepo ports.UserRepository,
	logger *logger.Logger,
) *ProjectService {
	return &ProjectService{
		projectRepo: projectRepo,
		userRepo:    userRepo,
		logger:      logger,
	}
}

// CreateProjectRequest represents a project creation request
type CreateProjectRequest struct {
	Name        string            `json:"name" validate:"required,min=3,max=255"`
	ProjectCode string            `json:"project_code" validate:"required,min=2,max=50"`
	Description *string           `json:"description" validate:"omitempty,max=2000"`
	Priority    entities.Priority `json:"priority" validate:"required"`
}

// CreateProject creates a new project (simplified for now)
func (s *ProjectService) CreateProject(ctx context.Context, req CreateProjectRequest, createdBy uuid.UUID) (*entities.Project, error) {
	// Simplified implementation for now
	project := &entities.Project{
		Name:        req.Name,
		ProjectCode: req.ProjectCode,
		Description: req.Description,
		Priority:    req.Priority,
		OwnerID:     &createdBy,
	}

	err := s.projectRepo.Create(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	s.logger.LogUserAction(createdBy.String(), "project_created", map[string]interface{}{
		"project_code": project.ProjectCode,
		"name":         project.Name,
	})

	return project, nil
}

// GetProject retrieves a project by ID
func (s *ProjectService) GetProject(ctx context.Context, projectID int) (*entities.Project, error) {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return project, nil
}
