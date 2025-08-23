package services

import (
	"context"
	"fmt"
	"time"

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
func NewProjectService(projectRepo ports.ProjectRepository, userRepo ports.UserRepository, logger *logger.Logger) *ProjectService {
	return &ProjectService{
		projectRepo: projectRepo,
		userRepo:    userRepo,
		logger:      logger,
	}
}

// CreateProject creates a new project
func (s *ProjectService) CreateProject(ctx context.Context, req ports.CreateProjectRequest) (*entities.Project, error) {
	// Verify manager exists
	_, err := s.userRepo.GetByID(ctx, req.ManagerID)
	if err != nil {
		return nil, fmt.Errorf("manager not found: %w", err)
	}

	// Create project entity
	project := &entities.Project{
		Name:        req.Name,
		Description: req.Description,
		Status:      req.Status,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		ManagerID:   req.ManagerID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Get created by from context (would need to be passed in real implementation)
	// For now, using the manager as creator
	project.CreatedBy = req.ManagerID

	createdProject, err := s.projectRepo.Create(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	s.logger.Info("Project created successfully", "project_id", createdProject.ID, "name", createdProject.Name)

	return createdProject, nil
}

// GetProject retrieves a project by ID
func (s *ProjectService) GetProject(ctx context.Context, id int) (*entities.Project, error) {
	project, err := s.projectRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	return project, nil
}

// UpdateProject updates a project's information
func (s *ProjectService) UpdateProject(ctx context.Context, id int, req ports.UpdateProjectRequest) (*entities.Project, error) {
	// Get existing project
	existingProject, err := s.projectRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	// Verify manager exists if being updated
	if req.ManagerID != nil {
		_, err := s.userRepo.GetByID(ctx, *req.ManagerID)
		if err != nil {
			return nil, fmt.Errorf("manager not found: %w", err)
		}
	}

	// Update fields
	if req.Name != nil {
		existingProject.Name = *req.Name
	}
	if req.Description != nil {
		existingProject.Description = req.Description
	}
	if req.Status != nil {
		existingProject.Status = *req.Status
	}
	if req.StartDate != nil {
		existingProject.StartDate = req.StartDate
	}
	if req.EndDate != nil {
		existingProject.EndDate = req.EndDate
	}
	if req.ManagerID != nil {
		existingProject.ManagerID = *req.ManagerID
	}

	existingProject.UpdatedAt = time.Now()

	updatedProject, err := s.projectRepo.Update(ctx, existingProject)
	if err != nil {
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	s.logger.Info("Project updated successfully", "project_id", updatedProject.ID, "name", updatedProject.Name)

	return updatedProject, nil
}

// DeleteProject deletes a project
func (s *ProjectService) DeleteProject(ctx context.Context, id int) error {
	// Check if project exists
	_, err := s.projectRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("project not found: %w", err)
	}

	err = s.projectRepo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	s.logger.Info("Project deleted successfully", "project_id", id)

	return nil
}

// ListProjects retrieves projects with filtering and pagination
func (s *ProjectService) ListProjects(ctx context.Context, filter ports.ProjectFilter) ([]*entities.Project, int, error) {
	projects, total, err := s.projectRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list projects: %w", err)
	}

	return projects, total, nil
}

// GetUserProjects gets all projects associated with a user
func (s *ProjectService) GetUserProjects(ctx context.Context, userID uuid.UUID) ([]*entities.Project, error) {
	projects, err := s.projectRepo.GetUserProjects(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user projects: %w", err)
	}

	return projects, nil
}
