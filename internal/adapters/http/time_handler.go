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

// TimeHandler handles time tracking related requests
type TimeHandler struct {
	timeService *services.TimeService
	logger      *logger.Logger
}

// NewTimeHandler creates a new time handler
func NewTimeHandler(timeService *services.TimeService, logger *logger.Logger) *TimeHandler {
	return &TimeHandler{
		timeService: timeService,
		logger:      logger,
	}
}

// CreateTimeEntry godoc
// @Summary Create a new time entry
// @Description Create a new time tracking entry
// @Tags time-tracking
// @Accept json
// @Produce json
// @Param request body ports.CreateTimeEntryRequest true "Time entry data"
// @Success 201 {object} entities.TimeEntry
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Security BearerAuth
// @Router /time [post]
func (h *TimeHandler) CreateTimeEntry(c echo.Context) error {
	userID := getUserIDFromContext(c)

	var req ports.CreateTimeEntryRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	entry, err := h.timeService.CreateTimeEntry(c.Request().Context(), req)
	if err != nil {
		h.logger.Error("Create time entry failed", "error", err, "user_id", userID)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusCreated, entry)
}

// GetEntry godoc
// @Summary Get time entry by ID
// @Description Get time entry details by ID
// @Tags time-tracking
// @Produce json
// @Param id path string true "Time Entry ID"
// @Success 200 {object} entities.TimeEntry
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /time/entries/{id} [get]
func (h *TimeHandler) GetEntry(c echo.Context) error {
	entryIDStr := c.Param("id")
	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid time entry ID")
	}

	entry, err := h.timeService.GetTimeEntry(c.Request().Context(), entryID)
	if err != nil {
		h.logger.Error("Get time entry failed", "error", err, "entry_id", entryID)
		return echo.NewHTTPError(http.StatusNotFound, "Time entry not found")
	}

	return c.JSON(http.StatusOK, entry)
}

// UpdateEntry godoc
// @Summary Update time entry
// @Description Update time entry information
// @Tags time-tracking
// @Accept json
// @Produce json
// @Param id path string true "Time Entry ID"
// @Param request body ports.UpdateTimeEntryRequest true "Time entry update data"
// @Success 200 {object} entities.TimeEntry
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /time/entries/{id} [put]
func (h *TimeHandler) UpdateEntry(c echo.Context) error {
	entryIDStr := c.Param("id")
	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid time entry ID")
	}

	var req ports.UpdateTimeEntryRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	entry, err := h.timeService.UpdateTimeEntry(c.Request().Context(), entryID, req)
	if err != nil {
		h.logger.Error("Update time entry failed", "error", err, "entry_id", entryID)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, entry)
}

// DeleteEntry godoc
// @Summary Delete time entry
// @Description Delete time entry by ID
// @Tags time-tracking
// @Param id path string true "Time Entry ID"
// @Success 204 "Time entry deleted successfully"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /time/entries/{id} [delete]
func (h *TimeHandler) DeleteEntry(c echo.Context) error {
	entryIDStr := c.Param("id")
	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid time entry ID")
	}

	err = h.timeService.DeleteTimeEntry(c.Request().Context(), entryID)
	if err != nil {
		h.logger.Error("Delete time entry failed", "error", err, "entry_id", entryID)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

// ListEntries godoc
// @Summary List time entries
// @Description Get list of time entries with optional filtering
// @Tags time-tracking
// @Produce json
// @Param user_id query string false "Filter by user"
// @Param task_id query int false "Filter by task"
// @Param project_id query int false "Filter by project"
// @Param start_date query string false "Filter by start date (RFC3339 format)"
// @Param end_date query string false "Filter by end date (RFC3339 format)"
// @Param limit query int false "Number of items to return" default(20)
// @Param offset query int false "Number of items to skip" default(0)
// @Success 200 {object} PaginatedResponse[entities.TimeEntry]
// @Failure 400 {object} ErrorResponse
// @Security BearerAuth
// @Router /time/entries [get]
func (h *TimeHandler) ListEntries(c echo.Context) error {
	filter := ports.TimeEntryFilter{}

	// Parse query parameters
	if userIDStr := c.QueryParam("user_id"); userIDStr != "" {
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid user_id parameter")
		}
		filter.UserID = &userID
	}

	if taskIDStr := c.QueryParam("task_id"); taskIDStr != "" {
		taskID, err := strconv.Atoi(taskIDStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid task_id parameter")
		}
		filter.TaskID = &taskID
	}

	if projectIDStr := c.QueryParam("project_id"); projectIDStr != "" {
		projectID, err := strconv.Atoi(projectIDStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid project_id parameter")
		}
		filter.ProjectID = &projectID
	}

	if startDateStr := c.QueryParam("start_date"); startDateStr != "" {
		startDate, err := parseTime(startDateStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid start_date parameter")
		}
		filter.StartDate = startDate
	}

	if endDateStr := c.QueryParam("end_date"); endDateStr != "" {
		endDate, err := parseTime(endDateStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid end_date parameter")
		}
		filter.EndDate = endDate
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
		filter.SortBy = "start_time"
	}

	filter.SortOrder = c.QueryParam("sort_order")
	if filter.SortOrder == "" {
		filter.SortOrder = "desc"
	}

	entries, total, err := h.timeService.ListTimeEntries(c.Request().Context(), filter)
	if err != nil {
		h.logger.Error("List time entries failed", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve time entries")
	}

	response := PaginatedResponse[*entities.TimeEntry]{
		Data:   entries,
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}

	return c.JSON(http.StatusOK, response)
}

// StartTime godoc
// @Summary Start time tracking
// @Description Start tracking time for a task
// @Tags time-tracking
// @Accept json
// @Produce json
// @Param request body map[string]int true "Task ID"
// @Success 201 {object} entities.TimeEntry
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Security BearerAuth
// @Router /time/start [post]
func (h *TimeHandler) StartTime(c echo.Context) error {
	userID := getUserIDFromContext(c)

	var req map[string]int
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	taskID, ok := req["task_id"]
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "task_id is required")
	}

	entry, err := h.timeService.StartTimeTracking(c.Request().Context(), userID, taskID)
	if err != nil {
		h.logger.Error("Start time tracking failed", "error", err, "user_id", userID, "task_id", taskID)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusCreated, entry)
}

// StopTime godoc
// @Summary Stop time tracking
// @Description Stop the active time tracking for current user
// @Tags time-tracking
// @Produce json
// @Success 200 {object} entities.TimeEntry
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Security BearerAuth
// @Router /time/stop [post]
func (h *TimeHandler) StopTime(c echo.Context) error {
	userID := getUserIDFromContext(c)

	entry, err := h.timeService.StopTimeTracking(c.Request().Context(), userID)
	if err != nil {
		h.logger.Error("Stop time tracking failed", "error", err, "user_id", userID)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, entry)
}

// GetActiveTimeEntry godoc
// @Summary Get active time entry
// @Description Get the currently active time entry for the user
// @Tags time-tracking
// @Produce json
// @Success 200 {object} entities.TimeEntry
// @Failure 404 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Security BearerAuth
// @Router /time/active [get]
func (h *TimeHandler) GetActiveTimeEntry(c echo.Context) error {
	userID := getUserIDFromContext(c)

	entry, err := h.timeService.GetActiveTimeEntry(c.Request().Context(), userID)
	if err != nil {
		h.logger.Error("Get active time entry failed", "error", err, "user_id", userID)
		return echo.NewHTTPError(http.StatusNotFound, "No active time entry found")
	}

	return c.JSON(http.StatusOK, entry)
}

// GetTimeReport godoc
// @Summary Get time report
// @Description Generate time tracking report with various grouping options
// @Tags time-tracking
// @Accept json
// @Produce json
// @Param request body ports.TimeReportRequest true "Time report parameters"
// @Success 200 {object} ports.TimeReport
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Security BearerAuth
// @Router /time/reports [post]
func (h *TimeHandler) GetTimeReport(c echo.Context) error {
	var req ports.TimeReportRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	report, err := h.timeService.GetTimeReport(c.Request().Context(), req)
	if err != nil {
		h.logger.Error("Get time report failed", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate report")
	}

	return c.JSON(http.StatusOK, report)
}
