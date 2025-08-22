package http

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/taskmaster/core/internal/application/services"
	"github.com/taskmaster/core/internal/infrastructure/logger"
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

// Placeholder methods for compilation - will implement later
func (h *ProjectHandler) UpdateProject(c echo.Context) error {
	return echo.NewHTTPError(http.StatusNotImplemented, "Not implemented yet")
}

func (h *ProjectHandler) DeleteProject(c echo.Context) error {
	return echo.NewHTTPError(http.StatusNotImplemented, "Not implemented yet")
}

func (h *ProjectHandler) ListProjects(c echo.Context) error {
	return echo.NewHTTPError(http.StatusNotImplemented, "Not implemented yet")
}

func (h *ProjectHandler) GetMyProjects(c echo.Context) error {
	return echo.NewHTTPError(http.StatusNotImplemented, "Not implemented yet")
}

func (h *ProjectHandler) AddProjectMember(c echo.Context) error {
	return echo.NewHTTPError(http.StatusNotImplemented, "Not implemented yet")
}

func (h *ProjectHandler) RemoveProjectMember(c echo.Context) error {
	return echo.NewHTTPError(http.StatusNotImplemented, "Not implemented yet")
}

func (h *ProjectHandler) GetProjectStats(c echo.Context) error {
	return echo.NewHTTPError(http.StatusNotImplemented, "Not implemented yet")
}

func (h *ProjectHandler) ActivateProject(c echo.Context) error {
	return echo.NewHTTPError(http.StatusNotImplemented, "Not implemented yet")
}

func (h *ProjectHandler) CompleteProject(c echo.Context) error {
	return echo.NewHTTPError(http.StatusNotImplemented, "Not implemented yet")
}

func (h *ProjectHandler) GetProjectTasks(c echo.Context) error {
	return echo.NewHTTPError(http.StatusNotImplemented, "Not implemented yet")
}
