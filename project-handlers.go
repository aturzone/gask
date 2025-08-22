// internal/adapters/http/project_handler.go
package http

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/taskmaster/core/internal/application/services"
	"github.com/taskmaster/core/internal/domain/entities"
	"github.com/taskmaster/core/internal/infrastructure/logger"
	"github.com/taskmaster/core/internal/ports"
)

// ProjectHandler handles project-related requests
type ProjectHandler struct {
	projectService *services.ProjectService
	logger         *logger.Logger
}

// NewProjectHandler creates a new project handler
func NewProjectHandler(projectService *services.ProjectService, logger *logger.Logger) *ProjectHandler {
	return &ProjectHandler{
		projectService: projectService,
		logger:         logger,
	}
}

// CreateProject godoc
// @Summary Create a new project
// @Description Create a new project with the provided details
// @Tags projects
// @Accept json
// @Produce json
// @Param request body services.CreateProjectRequest true "Project data"
// @Success 201 {object} entities.Project
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Security BearerAuth
// @Router /projects [post]
func (h *ProjectHandler) CreateProject(c echo.Context) error {
	userID := getUserIDFromContext(c)

	var req services.CreateProjectRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	project, err := h.projectService.CreateProject(c.Request().Context(), req, userID)
	if err != nil {
		h.logger.Error("Create project failed", "error", err, "user_id", userID)
		if contains(err.Error(), "already exists") {
			return echo.NewHTTPError(http.StatusConflict, err.Error())
		}
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusCreated, project)
}

// GetProject godoc
// @Summary Get project by ID
// @Description Get project information by project ID
// @Tags projects
// @Produce json
// @Param id path int true "Project ID"
// @Success 200 {object} entities.Project
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /projects/{id} [get]
func (h *ProjectHandler) GetProject(c echo.Context) error {
	projectIDStr := c.Param("id")
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid project ID")
	}

	project, err := h.projectService.GetProject(c.Request().Context(), projectID)
	if err != nil {
		h.logger.Error("Get project failed", "error", err, "project_id", projectID)
		return echo.NewHTTPError(http.StatusNotFound, "Project not found")
	}

	return c.JSON(http.StatusOK, project)
}

// UpdateProject godoc
// @Summary Update project
// @Description Update project information
// @Tags projects
// @Accept json
// @Produce json
// @Param id path int true "Project ID"
// @Param request body services.UpdateProjectRequest true "Project update data"
// @Success 200 {object} entities.Project
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /projects/{id} [put]
func (h *ProjectHandler) UpdateProject(c echo.Context) error {
	userID := getUserIDFromContext(c)

	projectIDStr := c.Param("id")
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid project ID")
	}

	var req services.UpdateProjectRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	project, err := h.projectService.UpdateProject(c.Request().Context(), projectID, req, userID)
	if err != nil {
		h.logger.Error("Update project failed", "error", err, "project_id", projectID, "user_id", userID)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, project)
}

// DeleteProject godoc
// @Summary Delete project
// @Description Soft delete a project
// @Tags projects
// @Param id path int true "Project ID"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /projects/{id} [delete]
func (h *ProjectHandler) DeleteProject(c echo.Context) error {
	userID := getUserIDFromContext(c)

	projectIDStr := c.Param("id")
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid project ID")
	}

	err = h.projectService.DeleteProject(c.Request().Context(), projectID, userID)
	if err != nil {
		h.logger.Error("Delete project failed", "error", err, "project_id", projectID, "user_id", userID)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, MessageResponse{Message: "Project deleted successfully"})
}

// ListProjects godoc
// @Summary List projects
// @Description Get a paginated list of projects with optional filtering
// @Tags projects
// @Produce json
// @Param status query string false "Filter by status"
// @Param owner_id query string false "Filter by owner ID"
// @Param priority query string false "Filter by priority"
// @Param search query string false "Search in name, description, or code"
// @Param limit query int false "Number of results per page" default(20)
// @Param offset query int false "Number of results to skip" default(0)
// @Param sort_by query string false "Sort field" default(created_at)
// @Param sort_order query string false "Sort order (asc/desc)" default(desc)
// @Success 200 {object} PaginatedResponse[entities.Project]
// @Failure 400 {object} ErrorResponse
// @Security BearerAuth
// @Router /projects [get]
func (h *ProjectHandler) ListProjects(c echo.Context) error {
	filter := ports.ProjectFilter{}

	// Parse query parameters
	if status := c.QueryParam("status"); status != "" {
		projectStatus := entities.ProjectStatus(status)
		filter.Status = &projectStatus
	}

	if ownerIDStr := c.QueryParam("owner_id"); ownerIDStr != "" {
		ownerID, err := uuid.Parse(ownerIDStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid owner_id parameter")
		}
		filter.OwnerID = &ownerID
	}

	if priority := c.QueryParam("priority"); priority != "" {
		projectPriority := entities.Priority(priority)
		filter.Priority = &projectPriority
	}

	if search := c.QueryParam("search"); search != "" {
		filter.Search = &search
	}

	if limitStr := c.QueryParam("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid limit parameter")
		}
		filter.Limit = limit
	} else {
		filter.Limit = 20
	}

	if offsetStr := c.QueryParam("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid offset parameter")
		}
		filter.Offset = offset
	}

	filter.SortBy = c.QueryParam("sort_by")
	if filter.SortBy == "" {
		filter.SortBy = "created_at"
	}

	filter.SortOrder = c.QueryParam("sort_order")
	if filter.SortOrder == "" {
		filter.SortOrder = "desc"
	}

	projects, total, err := h.projectService.ListProjects(c.Request().Context(), filter)
	if err != nil {
		h.logger.Error("List projects failed", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve projects")
	}

	response := PaginatedResponse[*entities.Project]{
		Data:   projects,
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}

	return c.JSON(http.StatusOK, response)
}

// GetMyProjects godoc
// @Summary Get current user's projects
// @Description Get projects where the current user is a member
// @Tags projects
// @Produce json
// @Success 200 {array} entities.Project
// @Failure 401 {object} ErrorResponse
// @Security BearerAuth
// @Router /projects/me [get]
func (h *ProjectHandler) GetMyProjects(c echo.Context) error {
	userID := getUserIDFromContext(c)

	projects, err := h.projectService.GetUserProjects(c.Request().Context(), userID)
	if err != nil {
		h.logger.Error("Get user projects failed", "error", err, "user_id", userID)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve projects")
	}

	return c.JSON(http.StatusOK, projects)
}

// AddProjectMember godoc
// @Summary Add member to project
// @Description Add a team member to a project
// @Tags projects
// @Accept json
// @Produce json
// @Param id path int true "Project ID"
// @Param request body services.AddProjectMemberRequest true "Member data"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /projects/{id}/members [post]
func (h *ProjectHandler) AddProjectMember(c echo.Context) error {
	userID := getUserIDFromContext(c)

	projectIDStr := c.Param("id")
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid project ID")
	}

	var req services.AddProjectMemberRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	err = h.projectService.AddProjectMember(c.Request().Context(), projectID, req, userID)
	if err != nil {
		h.logger.Error("Add project member failed", "error", err, "project_id", projectID)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, MessageResponse{Message: "Member added successfully"})
}

// RemoveProjectMember godoc
// @Summary Remove member from project
// @Description Remove a team member from a project
// @Tags projects
// @Param id path int true "Project ID"
// @Param user_id path string true "User ID"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /projects/{id}/members/{user_id} [delete]
func (h *ProjectHandler) RemoveProjectMember(c echo.Context) error {
	currentUserID := getUserIDFromContext(c)

	projectIDStr := c.Param("id")
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid project ID")
	}

	userIDStr := c.Param("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid user ID")
	}

	err = h.projectService.RemoveProjectMember(c.Request().Context(), projectID, userID, currentUserID)
	if err != nil {
		h.logger.Error("Remove project member failed", "error", err, "project_id", projectID, "user_id", userID)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, MessageResponse{Message: "Member removed successfully"})
}

// GetProjectStats godoc
// @Summary Get project statistics
// @Description Get overall project statistics
// @Tags projects
// @Produce json
// @Param owner_id query string false "Filter by owner ID"
// @Success 200 {object} services.ProjectStats
// @Failure 400 {object} ErrorResponse
// @Security BearerAuth
// @Router /projects/stats [get]
func (h *ProjectHandler) GetProjectStats(c echo.Context) error {
	var ownerID *uuid.UUID

	if ownerIDStr := c.QueryParam("owner_id"); ownerIDStr != "" {
		id, err := uuid.Parse(ownerIDStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid owner_id parameter")
		}
		ownerID = &id
	}

	stats, err := h.projectService.GetProjectStats(c.Request().Context(), ownerID)
	if err != nil {
		h.logger.Error("Get project stats failed", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve project statistics")
	}

	return c.JSON(http.StatusOK, stats)
}

// ActivateProject godoc
// @Summary Activate project
// @Description Change project status to active
// @Tags projects
// @Accept json
// @Produce json
// @Param id path int true "Project ID"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /projects/{id}/activate [post]
func (h *ProjectHandler) ActivateProject(c echo.Context) error {
	userID := getUserIDFromContext(c)

	projectIDStr := c.Param("id")
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid project ID")
	}

	err = h.projectService.ActivateProject(c.Request().Context(), projectID, userID)
	if err != nil {
		h.logger.Error("Activate project failed", "error", err, "project_id", projectID)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, MessageResponse{Message: "Project activated successfully"})
}

// CompleteProject godoc
// @Summary Complete project
// @Description Change project status to completed
// @Tags projects
// @Accept json
// @Produce json
// @Param id path int true "Project ID"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /projects/{id}/complete [post]
func (h *ProjectHandler) CompleteProject(c echo.Context) error {
	userID := getUserIDFromContext(c)

	projectIDStr := c.Param("id")
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid project ID")
	}

	err = h.projectService.CompleteProject(c.Request().Context(), projectID, userID)
	if err != nil {
		h.logger.Error("Complete project failed", "error", err, "project_id", projectID)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, MessageResponse{Message: "Project completed successfully"})
}

// GetProjectTasks godoc
// @Summary Get project tasks
// @Description Get all tasks for a specific project
// @Tags projects
// @Produce json
// @Param id path int true "Project ID"
// @Param status query string false "Filter by task status"
// @Param assignee_id query string false "Filter by assignee ID"
// @Param limit query int false "Number of results per page" default(20)
// @Param offset query int false "Number of results to skip" default(0)
// @Success 200 {object} PaginatedResponse[entities.Task]
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /projects/{id}/tasks [get]
func (h *ProjectHandler) GetProjectTasks(c echo.Context) error {
	projectIDStr := c.Param("id")
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid project ID")
	}

	// Verify project exists
	_, err = h.projectService.GetProject(c.Request().Context(), projectID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Project not found")
	}

	filter := ports.TaskFilter{
		ProjectID: &projectID,
	}

	// Parse additional filters
	if status := c.QueryParam("status"); status != "" {
		taskStatus := entities.TaskStatus(status)
		filter.Status = &taskStatus
	}

	if assigneeIDStr := c.QueryParam("assignee_id"); assigneeIDStr != "" {
		assigneeID, err := uuid.Parse(assigneeIDStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid assignee_id parameter")
		}
		filter.AssigneeID = &assigneeID
	}

	if limitStr := c.QueryParam("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid limit parameter")
		}
		filter.Limit = limit
	} else {
		filter.Limit = 20
	}

	if offsetStr := c.QueryParam("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid offset parameter")
		}
		filter.Offset = offset
	}

	filter.SortBy = "created_at"
	filter.SortOrder = "desc"

	// This would use the task service to get tasks
	// For now, we'll return an empty response structure
	response := PaginatedResponse[*entities.Task]{
		Data:   []*entities.Task{},
		Total:  0,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}

	return c.JSON(http.StatusOK, response)
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		 containsHelper(s, substr))))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}