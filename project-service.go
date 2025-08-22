// internal/application/services/project_service.go
package services

import (
	"context"
	"fmt"
	"strings"
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
	Name        string                    `json:"name" validate:"required,min=3,max=255"`
	ProjectCode string                    `json:"project_code" validate:"required,min=2,max=50"`
	Description *string                   `json:"description" validate:"omitempty,max=2000"`
	Priority    entities.Priority         `json:"priority" validate:"required"`
	StartDate   *time.Time                `json:"start_date"`
	EndDate     *time.Time                `json:"end_date"`
	Budget      *float64                  `json:"budget" validate:"omitempty,min=0"`
	CurrencyCode string                   `json:"currency_code" validate:"required,len=3"`
	ClientName  *string                   `json:"client_name" validate:"omitempty,max=255"`
}

// UpdateProjectRequest represents a project update request
type UpdateProjectRequest struct {
	Name        *string           `json:"name" validate:"omitempty,min=3,max=255"`
	Description *string           `json:"description" validate:"omitempty,max=2000"`
	Status      *entities.ProjectStatus `json:"status"`
	Priority    *entities.Priority `json:"priority"`
	StartDate   *time.Time        `json:"start_date"`
	EndDate     *time.Time        `json:"end_date"`
	Budget      *float64          `json:"budget" validate:"omitempty,min=0"`
	CurrencyCode *string          `json:"currency_code" validate:"omitempty,len=3"`
	ClientName  *string           `json:"client_name" validate:"omitempty,max=255"`
}

// AddProjectMemberRequest represents adding a member to a project
type AddProjectMemberRequest struct {
	UserID               uuid.UUID `json:"user_id" validate:"required"`
	Role                 string    `json:"role" validate:"required"`
	AllocationPercentage float64   `json:"allocation_percentage" validate:"required,min=0,max=100"`
}

// ProjectStats represents project statistics
type ProjectStats struct {
	TotalProjects     int64   `json:"total_projects"`
	ActiveProjects    int64   `json:"active_projects"`
	CompletedProjects int64   `json:"completed_projects"`
	TotalBudget       float64 `json:"total_budget"`
	SpentBudget       float64 `json:"spent_budget"`
	BudgetUtilization float64 `json:"budget_utilization"`
	OverdueProjects   int64   `json:"overdue_projects"`
}

// CreateProject creates a new project
func (s *ProjectService) CreateProject(ctx context.Context, req CreateProjectRequest, createdBy uuid.UUID) (*entities.Project, error) {
	// Validate project code uniqueness
	existingProject, err := s.projectRepo.GetByCode(ctx, req.ProjectCode)
	if err == nil && existingProject != nil {
		return nil, fmt.Errorf("project with code %s already exists", req.ProjectCode)
	}

	// Validate dates
	if req.StartDate != nil && req.EndDate != nil && req.EndDate.Before(*req.StartDate) {
		return nil, fmt.Errorf("end date cannot be before start date")
	}

	// Create project entity
	project := &entities.Project{
		Name:         req.Name,
		ProjectCode:  strings.ToUpper(req.ProjectCode),
		Description:  req.Description,
		Status:       entities.ProjectStatusPlanning,
		Priority:     req.Priority,
		StartDate:    req.StartDate,
		EndDate:      req.EndDate,
		Budget:       req.Budget,
		SpentBudget:  0,
		CurrencyCode: strings.ToUpper(req.CurrencyCode),
		OwnerID:      &createdBy,
		ClientName:   req.ClientName,
	}

	// Save project
	err = s.projectRepo.Create(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	// Add creator as project manager
	member := &entities.ProjectMember{
		ProjectID:            project.ID,
		UserID:               createdBy,
		Role:                 "project_manager",
		AllocationPercentage: 100.0,
	}

	err = s.projectRepo.AddProjectMember(ctx, member)
	if err != nil {
		s.logger.Error("Failed to add creator as project member", "error", err, "project_id", project.ID)
		// Don't fail the project creation for this
	}

	s.logger.LogUserAction(createdBy.String(), "project_created", map[string]interface{}{
		"project_id":   project.ID,
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

	// Load team members
	members, err := s.projectRepo.GetProjectMembers(ctx, projectID)
	if err != nil {
		s.logger.Error("Failed to load project members", "error", err, "project_id", projectID)
		// Don't fail the request, just log the error
	} else {
		project.TeamMembers = members
	}

	return project, nil
}

// UpdateProject updates a project
func (s *ProjectService) UpdateProject(ctx context.Context, projectID int, req UpdateProjectRequest, updatedBy uuid.UUID) (*entities.Project, error) {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// Update fields if provided
	if req.Name != nil {
		project.Name = *req.Name
	}
	if req.Description != nil {
		project.Description = req.Description
	}
	if req.Status != nil {
		project.Status = *req.Status
	}
	if req.Priority != nil {
		project.Priority = *req.Priority
	}
	if req.StartDate != nil {
		project.StartDate = req.StartDate
	}
	if req.EndDate != nil {
		project.EndDate = req.EndDate
	}
	if req.Budget != nil {
		project.Budget = req.Budget
	}
	if req.CurrencyCode != nil {
		project.CurrencyCode = strings.ToUpper(*req.CurrencyCode)
	}
	if req.ClientName != nil {
		project.ClientName = req.ClientName
	}

	// Validate dates
	if project.StartDate != nil && project.EndDate != nil && project.EndDate.Before(*project.StartDate) {
		return nil, fmt.Errorf("end date cannot be before start date")
	}

	err = s.projectRepo.Update(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	s.logger.LogUserAction(updatedBy.String(), "project_updated", map[string]interface{}{
		"project_id": project.ID,
	})

	return project, nil
}

// DeleteProject soft deletes a project
func (s *ProjectService) DeleteProject(ctx context.Context, projectID int, deletedBy uuid.UUID) error {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	// Check if project can be deleted (no active tasks, etc.)
	if project.Status == entities.ProjectStatusActive {
		return fmt.Errorf("cannot delete active project")
	}

	err = s.projectRepo.Delete(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	s.logger.LogUserAction(deletedBy.String(), "project_deleted", map[string]interface{}{
		"project_id":   project.ID,
		"project_code": project.ProjectCode,
	})

	return nil
}

// ListProjects lists projects with filtering
func (s *ProjectService) ListProjects(ctx context.Context, filter ports.ProjectFilter) ([]*entities.Project, int64, error) {
	projects, err := s.projectRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list projects: %w", err)
	}

	count, err := s.projectRepo.Count(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count projects: %w", err)
	}

	return projects, count, nil
}

// GetUserProjects gets projects for a specific user
func (s *ProjectService) GetUserProjects(ctx context.Context, userID uuid.UUID) ([]*entities.Project, error) {
	projects, err := s.projectRepo.GetUserProjects(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user projects: %w", err)
	}

	return projects, nil
}

// AddProjectMember adds a member to a project
func (s *ProjectService) AddProjectMember(ctx context.Context, projectID int, req AddProjectMemberRequest, addedBy uuid.UUID) error {
	// Verify project exists
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return fmt.Errorf("project not found: %w", err)
	}

	// Verify user exists
	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	if !user.IsActive {
		return fmt.Errorf("cannot add inactive user to project")
	}

	// Check if user is already a member
	members, err := s.projectRepo.GetProjectMembers(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to get project members: %w", err)
	}

	for _, member := range members {
		if member.UserID == req.UserID {
			return fmt.Errorf("user is already a member of this project")
		}
	}

	// Create member entity
	member := &entities.ProjectMember{
		ProjectID:            projectID,
		UserID:               req.UserID,
		Role:                 req.Role,
		AllocationPercentage: req.AllocationPercentage,
	}

	err = s.projectRepo.AddProjectMember(ctx, member)
	if err != nil {
		return fmt.Errorf("failed to add project member: %w", err)
	}

	s.logger.LogUserAction(addedBy.String(), "project_member_added", map[string]interface{}{
		"project_id": projectID,
		"user_id":    req.UserID.String(),
		"role":       req.Role,
	})

	return nil
}

// RemoveProjectMember removes a member from a project
func (s *ProjectService) RemoveProjectMember(ctx context.Context, projectID int, userID uuid.UUID, removedBy uuid.UUID) error {
	// Verify project exists
	_, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return fmt.Errorf("project not found: %w", err)
	}

	err = s.projectRepo.RemoveProjectMember(ctx, projectID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove project member: %w", err)
	}

	s.logger.LogUserAction(removedBy.String(), "project_member_removed", map[string]interface{}{
		"project_id": projectID,
		"user_id":    userID.String(),
	})

	return nil
}

// GetProjectStats returns project statistics
func (s *ProjectService) GetProjectStats(ctx context.Context, ownerID *uuid.UUID) (*ProjectStats, error) {
	filter := ports.ProjectFilter{
		OwnerID: ownerID,
		Limit:   0, // Get all projects
	}

	projects, err := s.projectRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects for stats: %w", err)
	}

	stats := &ProjectStats{}
	totalBudget := 0.0
	spentBudget := 0.0

	for _, project := range projects {
		stats.TotalProjects++

		switch project.Status {
		case entities.ProjectStatusActive:
			stats.ActiveProjects++
		case entities.ProjectStatusCompleted:
			stats.CompletedProjects++
		}

		if project.Budget != nil {
			totalBudget += *project.Budget
		}
		spentBudget += project.SpentBudget

		if project.IsOverdue() {
			stats.OverdueProjects++
		}
	}

	stats.TotalBudget = totalBudget
	stats.SpentBudget = spentBudget

	if totalBudget > 0 {
		stats.BudgetUtilization = (spentBudget / totalBudget) * 100
	}

	return stats, nil
}

// UpdateProjectBudget updates project spent budget
func (s *ProjectService) UpdateProjectBudget(ctx context.Context, projectID int, amount float64, updatedBy uuid.UUID) error {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	err = project.AddBudgetExpense(amount)
	if err != nil {
		return fmt.Errorf("failed to add budget expense: %w", err)
	}

	err = s.projectRepo.Update(ctx, project)
	if err != nil {
		return fmt.Errorf("failed to update project budget: %w", err)
	}

	s.logger.LogUserAction(updatedBy.String(), "project_budget_updated", map[string]interface{}{
		"project_id": project.ID,
		"amount":     amount,
		"new_spent":  project.SpentBudget,
	})

	return nil
}

// ActivateProject changes project status to active
func (s *ProjectService) ActivateProject(ctx context.Context, projectID int, activatedBy uuid.UUID) error {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	if project.Status != entities.ProjectStatusPlanning {
		return fmt.Errorf("only planning projects can be activated")
	}

	project.Status = entities.ProjectStatusActive

	err = s.projectRepo.Update(ctx, project)
	if err != nil {
		return fmt.Errorf("failed to activate project: %w", err)
	}

	s.logger.LogUserAction(activatedBy.String(), "project_activated", map[string]interface{}{
		"project_id": project.ID,
	})

	return nil
}

// CompleteProject changes project status to completed
func (s *ProjectService) CompleteProject(ctx context.Context, projectID int, completedBy uuid.UUID) error {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	if project.Status != entities.ProjectStatusActive {
		return fmt.Errorf("only active projects can be completed")
	}

	project.Status = entities.ProjectStatusCompleted

	err = s.projectRepo.Update(ctx, project)
	if err != nil {
		return fmt.Errorf("failed to complete project: %w", err)
	}

	s.logger.LogUserAction(completedBy.String(), "project_completed", map[string]interface{}{
		"project_id": project.ID,
	})

	return nil
}