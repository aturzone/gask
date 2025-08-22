// internal/application/services/time_service.go
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

// TimeService handles time tracking operations
type TimeService struct {
	timeRepo    ports.TimeEntryRepository
	taskRepo    ports.TaskRepository
	projectRepo ports.ProjectRepository
	userRepo    ports.UserRepository
	logger      *logger.Logger
}

// NewTimeService creates a new time service
func NewTimeService(
	timeRepo ports.TimeEntryRepository,
	taskRepo ports.TaskRepository,
	projectRepo ports.ProjectRepository,
	userRepo ports.UserRepository,
	logger *logger.Logger,
) *TimeService {
	return &TimeService{
		timeRepo:    timeRepo,
		taskRepo:    taskRepo,
		projectRepo: projectRepo,
		userRepo:    userRepo,
		logger:      logger,
	}
}

// CreateTimeEntryRequest represents a time entry creation request
type CreateTimeEntryRequest struct {
	TaskID          *int      `json:"task_id"`
	ProjectID       int       `json:"project_id" validate:"required"`
	StartTime       time.Time `json:"start_time" validate:"required"`
	EndTime         *time.Time `json:"end_time"`
	DurationMinutes *int      `json:"duration_minutes"`
	Description     *string   `json:"description" validate:"omitempty,max=500"`
	Billable        bool      `json:"billable"`
	HourlyRate      *float64  `json:"hourly_rate" validate:"omitempty,min=0"`
}

// UpdateTimeEntryRequest represents a time entry update request
type UpdateTimeEntryRequest struct {
	StartTime       *time.Time `json:"start_time"`
	EndTime         *time.Time `json:"end_time"`
	DurationMinutes *int       `json:"duration_minutes"`
	Description     *string    `json:"description" validate:"omitempty,max=500"`
	Billable        *bool      `json:"billable"`
	HourlyRate      *float64   `json:"hourly_rate" validate:"omitempty,min=0"`
}

// StartTimeTrackingRequest represents starting time tracking
type StartTimeTrackingRequest struct {
	TaskID      *int    `json:"task_id"`
	ProjectID   int     `json:"project_id" validate:"required"`
	Description *string `json:"description" validate:"omitempty,max=500"`
}

// TimeReport represents time tracking reports
type TimeReport struct {
	UserID       uuid.UUID              `json:"user_id"`
	Period       string                 `json:"period"`
	TotalHours   float64                `json:"total_hours"`
	BillableHours float64               `json:"billable_hours"`
	TotalCost    float64                `json:"total_cost"`
	ProjectBreakdown []ProjectTimeStats `json:"project_breakdown"`
	DailyBreakdown   []DailyTimeStats   `json:"daily_breakdown"`
}

// ProjectTimeStats represents time statistics per project
type ProjectTimeStats struct {
	ProjectID     int     `json:"project_id"`
	ProjectName   string  `json:"project_name"`
	TotalHours    float64 `json:"total_hours"`
	BillableHours float64 `json:"billable_hours"`
	TotalCost     float64 `json:"total_cost"`
}

// DailyTimeStats represents daily time statistics
type DailyTimeStats struct {
	Date          time.Time `json:"date"`
	TotalHours    float64   `json:"total_hours"`
	BillableHours float64   `json:"billable_hours"`
	TotalCost     float64   `json:"total_cost"`
}

// CreateTimeEntry creates a new time entry
func (s *TimeService) CreateTimeEntry(ctx context.Context, req CreateTimeEntryRequest, userID uuid.UUID) (*entities.TimeEntry, error) {
	// Verify project exists and user has access
	project, err := s.projectRepo.GetByID(ctx, req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	// Verify task exists and belongs to project if provided
	if req.TaskID != nil {
		task, err := s.taskRepo.GetByID(ctx, *req.TaskID)
		if err != nil {
			return nil, fmt.Errorf("task not found: %w", err)
		}
		if task.ProjectID != req.ProjectID {
			return nil, fmt.Errorf("task does not belong to the specified project")
		}
	}

	// Get user for hourly rate if not provided
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Calculate duration if end time is provided
	var durationMinutes *int
	if req.EndTime != nil {
		if req.EndTime.Before(req.StartTime) {
			return nil, fmt.Errorf("end time cannot be before start time")
		}
		duration := int(req.EndTime.Sub(req.StartTime).Minutes())
		durationMinutes = &duration
	} else if req.DurationMinutes != nil {
		durationMinutes = req.DurationMinutes
		endTime := req.StartTime.Add(time.Duration(*req.DurationMinutes) * time.Minute)
		req.EndTime = &endTime
	}

	// Use user's hourly rate if not provided
	hourlyRate := req.HourlyRate
	if hourlyRate == nil && user.HourlyRate != nil {
		hourlyRate = user.HourlyRate
	}

	// Create time entry entity
	entry := &entities.TimeEntry{
		UserID:          userID,
		TaskID:          req.TaskID,
		ProjectID:       req.ProjectID,
		StartTime:       req.StartTime,
		EndTime:         req.EndTime,
		DurationMinutes: durationMinutes,
		Description:     req.Description,
		EntryDate:       req.StartTime.Truncate(24 * time.Hour),
		Billable:        req.Billable,
		HourlyRate:      hourlyRate,
	}

	// Validate entry
	if !entry.IsValid() {
		return nil, fmt.Errorf("invalid time entry data")
	}

	// Save time entry
	err = s.timeRepo.Create(ctx, entry)
	if err != nil {
		return nil, fmt.Errorf("failed to create time entry: %w", err)
	}

	s.logger.LogUserAction(userID.String(), "time_entry_created", map[string]interface{}{
		"entry_id":   entry.ID,
		"project_id": entry.ProjectID,
		"task_id":    entry.TaskID,
		"duration":   durationMinutes,
	})

	return entry, nil
}

// StartTimeTracking starts time tracking for a user
func (s *TimeService) StartTimeTracking(ctx context.Context, req StartTimeTrackingRequest, userID uuid.UUID) (*entities.TimeEntry, error) {
	// Check if user already has an active time entry
	activeEntry, err := s.timeRepo.GetActiveEntry(ctx, userID)
	if err == nil && activeEntry != nil {
		return nil, fmt.Errorf("user already has an active time tracking session")
	}

	// Verify project exists
	_, err = s.projectRepo.GetByID(ctx, req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	// Verify task exists if provided
	if req.TaskID != nil {
		task, err := s.taskRepo.GetByID(ctx, *req.TaskID)
		if err != nil {
			return nil, fmt.Errorf("task not found: %w", err)
		}
		if task.ProjectID != req.ProjectID {
			return nil, fmt.Errorf("task does not belong to the specified project")
		}
	}

	// Get user for hourly rate
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	startTime := time.Now()
	
	// Create time entry without end time
	entry := &entities.TimeEntry{
		UserID:      userID,
		TaskID:      req.TaskID,
		ProjectID:   req.ProjectID,
		StartTime:   startTime,
		Description: req.Description,
		EntryDate:   startTime.Truncate(24 * time.Hour),
		Billable:    true,
		HourlyRate:  user.HourlyRate,
	}

	err = s.timeRepo.Create(ctx, entry)
	if err != nil {
		return nil, fmt.Errorf("failed to start time tracking: %w", err)
	}

	s.logger.LogUserAction(userID.String(), "time_tracking_started", map[string]interface{}{
		"entry_id":   entry.ID,
		"project_id": entry.ProjectID,
		"task_id":    entry.TaskID,
	})

	return entry, nil
}

// StopTimeTracking stops active time tracking for a user
func (s *TimeService) StopTimeTracking(ctx context.Context, userID uuid.UUID) (*entities.TimeEntry, error) {
	// Get active time entry
	entry, err := s.timeRepo.GetActiveEntry(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("no active time tracking session found")
	}

	// Stop the entry
	err = entry.Stop()
	if err != nil {
		return nil, fmt.Errorf("failed to stop time entry: %w", err)
	}

	// Update in database
	err = s.timeRepo.Update(ctx, entry)
	if err != nil {
		return nil, fmt.Errorf("failed to update time entry: %w", err)
	}

	s.logger.LogUserAction(userID.String(), "time_tracking_stopped", map[string]interface{}{
		"entry_id": entry.ID,
		"duration": entry.DurationMinutes,
	})

	return entry, nil
}

// GetTimeEntry retrieves a time entry by ID
func (s *TimeService) GetTimeEntry(ctx context.Context, entryID int, userID uuid.UUID) (*entities.TimeEntry, error) {
	entry, err := s.timeRepo.GetByID(ctx, entryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get time entry: %w", err)
	}

	// Verify user owns the entry or has permission to view it
	if entry.UserID != userID {
		// TODO: Check if user has admin/manager permissions
		return nil, fmt.Errorf("access denied")
	}

	return entry, nil
}

// UpdateTimeEntry updates a time entry
func (s *TimeService) UpdateTimeEntry(ctx context.Context, entryID int, req UpdateTimeEntryRequest, userID uuid.UUID) (*entities.TimeEntry, error) {
	entry, err := s.timeRepo.GetByID(ctx, entryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get time entry: %w", err)
	}

	// Verify user owns the entry
	if entry.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	// Update fields if provided
	if req.StartTime != nil {
		entry.StartTime = *req.StartTime
		entry.EntryDate = req.StartTime.Truncate(24 * time.Hour)
	}
	if req.EndTime != nil {
		entry.EndTime = req.EndTime
	}
	if req.DurationMinutes != nil {
		entry.DurationMinutes = req.DurationMinutes
	}
	if req.Description != nil {
		entry.Description = req.Description
	}
	if req.Billable != nil {
		entry.Billable = *req.Billable
	}
	if req.HourlyRate != nil {
		entry.HourlyRate = req.HourlyRate
	}

	// Recalculate duration if both start and end times are set
	if entry.EndTime != nil {
		duration := int(entry.EndTime.Sub(entry.StartTime).Minutes())
		entry.DurationMinutes = &duration
	}

	// Validate entry
	if !entry.IsValid() {
		return nil, fmt.Errorf("invalid time entry data")
	}

	err = s.timeRepo.Update(ctx, entry)
	if err != nil {
		return nil, fmt.Errorf("failed to update time entry: %w", err)
	}

	s.logger.LogUserAction(userID.String(), "time_entry_updated", map[string]interface{}{
		"entry_id": entry.ID,
	})

	return entry, nil
}

// DeleteTimeEntry deletes a time entry
func (s *TimeService) DeleteTimeEntry(ctx context.Context, entryID int, userID uuid.UUID) error {
	entry, err := s.timeRepo.GetByID(ctx, entryID)
	if err != nil {
		return fmt.Errorf("failed to get time entry: %w", err)
	}

	// Verify user owns the entry
	if entry.UserID != userID {
		return fmt.Errorf("access denied")
	}

	err = s.timeRepo.Delete(ctx, entryID)
	if err != nil {
		return fmt.Errorf("failed to delete time entry: %w", err)
	}

	s.logger.LogUserAction(userID.String(), "time_entry_deleted", map[string]interface{}{
		"entry_id": entryID,
	})

	return nil
}

// ListTimeEntries lists time entries with filtering
func (s *TimeService) ListTimeEntries(ctx context.Context, filter ports.TimeEntryFilter, userID uuid.UUID) ([]*entities.TimeEntry, int64, error) {
	// Ensure user can only see their own entries unless they have admin permissions
	filter.UserID = &userID

	entries, err := s.timeRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list time entries: %w", err)
	}

	count := int64(len(entries)) // Simplified count
	return entries, count, nil
}

// GetActiveTimeEntry gets the current user's active time tracking session
func (s *TimeService) GetActiveTimeEntry(ctx context.Context, userID uuid.UUID) (*entities.TimeEntry, error) {
	entry, err := s.timeRepo.GetActiveEntry(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("no active time tracking session")
	}

	return entry, nil
}

// GetTimeReport generates a time report for a user
func (s *TimeService) GetTimeReport(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) (*TimeReport, error) {
	filter := ports.TimeEntryFilter{
		UserID:    &userID,
		StartDate: &startDate,
		EndDate:   &endDate,
	}

	entries, err := s.timeRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get time entries: %w", err)
	}

	report := &TimeReport{
		UserID: userID,
		Period: fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")),
	}

	projectStats := make(map[int]*ProjectTimeStats)
	dailyStats := make(map[string]*DailyTimeStats)

	for _, entry := range entries {
		if entry.DurationMinutes == nil {
			continue
		}

		hours := float64(*entry.DurationMinutes) / 60.0
		cost := entry.CalculateCost()

		report.TotalHours += hours
		report.TotalCost += cost

		if entry.Billable {
			report.BillableHours += hours
		}

		// Project breakdown
		if _, exists := projectStats[entry.ProjectID]; !exists {
			// Get project name (simplified - in real implementation, you'd join or cache this)
			projectStats[entry.ProjectID] = &ProjectTimeStats{
				ProjectID:   entry.ProjectID,
				ProjectName: fmt.Sprintf("Project %d", entry.ProjectID), // Placeholder
			}
		}
		
		projectStats[entry.ProjectID].TotalHours += hours
		projectStats[entry.ProjectID].TotalCost += cost
		if entry.Billable {
			projectStats[entry.ProjectID].BillableHours += hours
		}

		// Daily breakdown
		dateKey := entry.EntryDate.Format("2006-01-02")
		if _, exists := dailyStats[dateKey]; !exists {
			dailyStats[dateKey] = &DailyTimeStats{
				Date: entry.EntryDate,
			}
		}
		
		dailyStats[dateKey].TotalHours += hours
		dailyStats[dateKey].TotalCost += cost
		if entry.Billable {
			dailyStats[dateKey].BillableHours += hours
		}
	}

	// Convert maps to slices
	for _, stats := range projectStats {
		report.ProjectBreakdown = append(report.ProjectBreakdown, *stats)
	}

	for _, stats := range dailyStats {
		report.DailyBreakdown = append(report.DailyBreakdown, *stats)
	}

	return report, nil
}

// =============================================================================
// internal/adapters/http/time_handler.go
package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/taskmaster/core/internal/application/services"
	"github.com/taskmaster/core/internal/infrastructure/logger"
	"github.com/taskmaster/core/internal/ports"
)

// TimeHandler handles time tracking requests
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
// @Summary Create a time entry
// @Description Create a new time tracking entry
// @Tags time-tracking
// @Accept json
// @Produce json
// @Param request body services.CreateTimeEntryRequest true "Time entry data"
// @Success 201 {object} entities.TimeEntry
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Security BearerAuth
// @Router /time-entries [post]
func (h *TimeHandler) CreateTimeEntry(c echo.Context) error {
	userID := getUserIDFromContext(c)

	var req services.CreateTimeEntryRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	entry, err := h.timeService.CreateTimeEntry(c.Request().Context(), req, userID)
	if err != nil {
		h.logger.Error("Create time entry failed", "error", err, "user_id", userID)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusCreated, entry)
}

// StartTimeTracking godoc
// @Summary Start time tracking
// @Description Start a new time tracking session
// @Tags time-tracking
// @Accept json
// @Produce json
// @Param request body services.StartTimeTrackingRequest true "Start time tracking data"
// @Success 201 {object} entities.TimeEntry
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Security BearerAuth
// @Router /time-entries/start [post]
func (h *TimeHandler) StartTimeTracking(c echo.Context) error {
	userID := getUserIDFromContext(c)

	var req services.StartTimeTrackingRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	entry, err := h.timeService.StartTimeTracking(c.Request().Context(), req, userID)
	if err != nil {
		h.logger.Error("Start time tracking failed", "error", err, "user_id", userID)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusCreated, entry)
}

// StopTimeTracking godoc
// @Summary Stop time tracking
// @Description Stop the current active time tracking session
// @Tags time-tracking
// @Produce json
// @Success 200 {object} entities.TimeEntry
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Security BearerAuth
// @Router /time-entries/stop [post]
func (h *TimeHandler) StopTimeTracking(c echo.Context) error {
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
// @Description Get the current user's active time tracking session
// @Tags time-tracking
// @Produce json
// @Success 200 {object} entities.TimeEntry
// @Failure 404 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Security BearerAuth
// @Router /time-entries/active [get]
func (h *TimeHandler) GetActiveTimeEntry(c echo.Context) error {
	userID := getUserIDFromContext(c)

	entry, err := h.timeService.GetActiveTimeEntry(c.Request().Context(), userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "No active time tracking session")
	}

	return c.JSON(http.StatusOK, entry)
}

// GetTimeEntry godoc
// @Summary Get time entry by ID
// @Description Get time entry information by ID
// @Tags time-tracking
// @Produce json
// @Param id path int true "Time Entry ID"
// @Success 200 {object} entities.TimeEntry
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /time-entries/{id} [get]
func (h *TimeHandler) GetTimeEntry(c echo.Context) error {
	userID := getUserIDFromContext(c)

	entryIDStr := c.Param("id")
	entryID, err := strconv.Atoi(entryIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid time entry ID")
	}

	entry, err := h.timeService.GetTimeEntry(c.Request().Context(), entryID, userID)
	if err != nil {
		h.logger.Error("Get time entry failed", "error", err, "entry_id", entryID, "user_id", userID)
		return echo.NewHTTPError(http.StatusNotFound, "Time entry not found")
	}

	return c.JSON(http.StatusOK, entry)
}

// UpdateTimeEntry godoc
// @Summary Update time entry
// @Description Update time entry information
// @Tags time-tracking
// @Accept json
// @Produce json
// @Param id path int true "Time Entry ID"
// @Param request body services.UpdateTimeEntryRequest true "Time entry update data"
// @Success 200 {object} entities.TimeEntry
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /time-entries/{id} [put]
func (h *TimeHandler) UpdateTimeEntry(c echo.Context) error {
	userID := getUserIDFromContext(c)

	entryIDStr := c.Param("id")
	entryID, err := strconv.Atoi(entryIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid time entry ID")
	}

	var req services.UpdateTimeEntryRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	entry, err := h.timeService.UpdateTimeEntry(c.Request().Context(), entryID, req, userID)
	if err != nil {
		h.logger.Error("Update time entry failed", "error", err, "entry_id", entryID, "user_id", userID)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, entry)
}

// DeleteTimeEntry godoc
// @Summary Delete time entry
// @Description Delete a time entry
// @Tags time-tracking
// @Param id path int true "Time Entry ID"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security BearerAuth
// @Router /time-entries/{id} [delete]
func (h *TimeHandler) DeleteTimeEntry(c echo.Context) error {
	userID := getUserIDFromContext(c)

	entryIDStr := c.Param("id")
	entryID, err := strconv.Atoi(entryIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid time entry ID")
	}

	err = h.timeService.DeleteTimeEntry(c.Request().Context(), entryID, userID)
	if err != nil {
		h.logger.Error("Delete time entry failed", "error", err, "entry_id", entryID, "user_id", userID)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, MessageResponse{Message: "Time entry deleted successfully"})
}

// ListTimeEntries godoc
// @Summary List time entries
// @Description Get a paginated list of time entries
// @Tags time-tracking
// @Produce json
// @Param project_id query int false "Filter by project ID"
// @Param task_id query int false "Filter by task ID"
// @Param start_date query string false "Start date (YYYY-MM-DD)"
// @Param end_date query string false "End date (YYYY-MM-DD)"
// @Param billable query bool false "Filter by billable status"
// @Param limit query int false "Number of results per page" default(20)
// @Param offset query int false "Number of results to skip" default(0)
// @Success 200 {object} PaginatedResponse[entities.TimeEntry]
// @Failure 400 {object} ErrorResponse
// @Security BearerAuth
// @Router /time-entries [get]
func (h *TimeHandler) ListTimeEntries(c echo.Context) error {
	userID := getUserIDFromContext(c)
	filter := ports.TimeEntryFilter{
		UserID: &userID,
	}

	// Parse query parameters
	if projectIDStr := c.QueryParam("project_id"); projectIDStr != "" {
		projectID, err := strconv.Atoi(projectIDStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid project_id parameter")
		}
		filter.ProjectID = &projectID
	}

	if taskIDStr := c.QueryParam("task_id"); taskIDStr != "" {
		taskID, err := strconv.Atoi(taskIDStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid task_id parameter")
		}
		filter.TaskID = &taskID
	}

	if startDateStr := c.QueryParam("start_date"); startDateStr != "" {
		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid start_date parameter (use YYYY-MM-DD format)")
		}
		filter.StartDate = &startDate
	}

	if endDateStr := c.QueryParam("end_date"); endDateStr != "" {
		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid end_date parameter (use YYYY-MM-DD format)")
		}
		filter.EndDate = &endDate
	}

	if billableStr := c.QueryParam("billable"); billableStr != "" {
		billable, err := strconv.ParseBool(billableStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid billable parameter")
		}
		filter.Billable = &billable
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

	filter.SortBy = "entry_date"
	filter.SortOrder = "desc"

	entries, total, err := h.timeService.ListTimeEntries(c.Request().Context(), filter, userID)
	if err != nil {
		h.logger.Error("List time entries failed", "error", err, "user_id", userID)
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

// GetTimeReport godoc
// @Summary Get time report
// @Description Generate a time tracking report for a date range
// @Tags time-tracking
// @Produce json
// @Param start_date query string true "Start date (YYYY-MM-DD)"
// @Param end_date query string true "End date (YYYY-MM-DD)"
// @Success 200 {object} services.TimeReport
// @Failure 400 {object} ErrorResponse
// @Security BearerAuth
// @Router /time-entries/report [get]
func (h *TimeHandler) GetTimeReport(c echo.Context) error {
	userID := getUserIDFromContext(c)

	startDateStr := c.QueryParam("start_date")
	endDateStr := c.QueryParam("end_date")

	if startDateStr == "" || endDateStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "start_date and end_date parameters are required")
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid start_date parameter (use YYYY-MM-DD format)")
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid end_date parameter (use YYYY-MM-DD format)")
	}

	if endDate.Before(startDate) {
		return echo.NewHTTPError(http.StatusBadRequest, "end_date cannot be before start_date")
	}

	report, err := h.timeService.GetTimeReport(c.Request().Context(), userID, startDate, endDate)
	if err != nil {
		h.logger.Error("Get time report failed", "error", err, "user_id", userID)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate time report")
	}

	return c.JSON(http.StatusOK, report)
}