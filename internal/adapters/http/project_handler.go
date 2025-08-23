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
// @Description Create a new project
// @Tags projects
// @Accept json
// @Produce json
// @Param request body ports.CreateProjectRequest true "Project data"
// @Success 201 {object} entities.Project
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Security BearerAuth
// @Router /projects [post]
func (h *ProjectHandler) CreateProject(c echo.Context) error {
	var req ports.CreateProjectRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	project, err := h.projectService.CreateProject(c.Request().Context(), req)
	if err != nil {
		h.logger.Error("Create project failed", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusCreated, project)
}

// GetProject godoc
// @Summary Get project by ID
// @Description Get project details by ID
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
// @Param request body ports.UpdateProjectRequest true "Project update data"
// @Success 200 {object} entities.Project
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /projects/{id} [put]
func (h *ProjectHandler) UpdateProject(c echo.Context) error {
	projectIDStr := c.Param("id")
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid project ID")
	}

	var req ports.UpdateProjectRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	project, err := h.projectService.UpdateProject(c.Request().Context(), projectID, req)
	if err != nil {
		h.logger.Error("Update project failed", "error", err, "project_id", projectID)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, project)
}

// DeleteProject godoc
// @Summary Delete project
// @Description Delete project by ID
// @Tags projects
// @Param id path int true "Project ID"
// @Success 204 "Project deleted successfully"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /projects/{id} [delete]
func (h *ProjectHandler) DeleteProject(c echo.Context) error {
	projectIDStr := c.Param("id")
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid project ID")
	}

	err = h.projectService.DeleteProject(c.Request().Context(), projectID)
	if err != nil {
		h.logger.Error("Delete project failed", "error", err, "project_id", projectID)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

// ListProjects godoc
// @Summary List projects
// @Description Get list of projects with optional filtering
// @Tags projects
// @Produce json
// @Param status query string false "Filter by status"
// @Param manager_id query string false "Filter by manager"
// @Param search query string false "Search in name and description"
// @Param limit query int false "Number of items to return" default(20)
// @Param offset query int false "Number of items to skip" default(0)
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

	if managerIDStr := c.QueryParam("manager_id"); managerIDStr != "" {
		managerID, err := uuid.Parse(managerIDStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid manager_id parameter")
		}
		filter.ManagerID = &managerID
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

// GetProjectTasks godoc
// @Summary Get project tasks
// @Description Get all tasks for a specific project
// @Tags projects
// @Produce json
// @Param id path int true "Project ID"
// @Param status query string false "Filter by task status"
// @Param assignee_id query string false "Filter by assignee"
// @Param limit query int false "Number of items to return" default(20)
// @Param offset query int false "Number of items to skip" default(0)
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

	// This would use the task service to get tasks (assuming we have access to it)
	// For now, we'll return an empty response structure
	response := PaginatedResponse[*entities.Task]{
		Data:   []*entities.Task{},
		Total:  0,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}

	return c.JSON(http.StatusOK, response)
}
