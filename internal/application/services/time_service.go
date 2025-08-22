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

// StartTimeTrackingRequest represents starting time tracking
type StartTimeTrackingRequest struct {
	TaskID      *int    `json:"task_id"`
	ProjectID   int     `json:"project_id" validate:"required"`
	Description *string `json:"description" validate:"omitempty,max=500"`
}

// CreateTimeEntry creates a new time entry (simplified version)
func (s *TimeService) CreateTimeEntry(ctx context.Context, req CreateTimeEntryRequest, userID uuid.UUID) (*entities.TimeEntry, error) {
	// Create time entry entity
	entry := &entities.TimeEntry{
		UserID:      userID,
		TaskID:      req.TaskID,
		ProjectID:   req.ProjectID,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		Description: req.Description,
		EntryDate:   req.StartTime.Truncate(24 * time.Hour),
		Billable:    req.Billable,
		HourlyRate:  req.HourlyRate,
	}

	// Calculate duration if end time is provided
	if req.EndTime != nil {
		duration := int(req.EndTime.Sub(req.StartTime).Minutes())
		entry.DurationMinutes = &duration
	}

	// Save time entry
	err := s.timeRepo.Create(ctx, entry)
	if err != nil {
		return nil, fmt.Errorf("failed to create time entry: %w", err)
	}

	s.logger.LogUserAction(userID.String(), "time_entry_created", map[string]interface{}{
		"entry_id":   entry.ID,
		"project_id": entry.ProjectID,
	})

	return entry, nil
}

// StartTimeTracking starts time tracking for a user
func (s *TimeService) StartTimeTracking(ctx context.Context, req StartTimeTrackingRequest, userID uuid.UUID) (*entities.TimeEntry, error) {
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
	}

	err := s.timeRepo.Create(ctx, entry)
	if err != nil {
		return nil, fmt.Errorf("failed to start time tracking: %w", err)
	}

	s.logger.LogUserAction(userID.String(), "time_tracking_started", map[string]interface{}{
		"entry_id":   entry.ID,
		"project_id": entry.ProjectID,
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

// GetActiveTimeEntry gets the current user's active time tracking session
func (s *TimeService) GetActiveTimeEntry(ctx context.Context, userID uuid.UUID) (*entities.TimeEntry, error) {
	entry, err := s.timeRepo.GetActiveEntry(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("no active time tracking session")
	}

	return entry, nil
}

// ListTimeEntries lists time entries with filtering (simplified)
func (s *TimeService) ListTimeEntries(ctx context.Context, filter ports.TimeEntryFilter, userID uuid.UUID) ([]*entities.TimeEntry, int64, error) {
	filter.UserID = &userID

	entries, err := s.timeRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list time entries: %w", err)
	}

	count := int64(len(entries))
	return entries, count, nil
}
