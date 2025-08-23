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
func NewTimeService(timeRepo ports.TimeEntryRepository, taskRepo ports.TaskRepository, projectRepo ports.ProjectRepository, userRepo ports.UserRepository, logger *logger.Logger) *TimeService {
	return &TimeService{
		timeRepo:    timeRepo,
		taskRepo:    taskRepo,
		projectRepo: projectRepo,
		userRepo:    userRepo,
		logger:      logger,
	}
}

// CreateTimeEntry creates a new time entry
func (s *TimeService) CreateTimeEntry(ctx context.Context, req ports.CreateTimeEntryRequest) (*entities.TimeEntry, error) {
	// Verify task exists
	_, err := s.taskRepo.GetByID(ctx, req.TaskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	// Create time entry entity
	entry := &entities.TimeEntry{
		ID:          uuid.New(),
		TaskID:      req.TaskID,
		Description: req.Description,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Calculate hours if end time is provided
	if req.Hours != nil {
		entry.Hours = *req.Hours
	} else if req.EndTime != nil {
		entry.Hours = req.EndTime.Sub(req.StartTime).Hours()
	}

	// Get user from context (would need to be passed in real implementation)
	// For now, using a placeholder UUID
	entry.UserID = uuid.New() // This should come from the authenticated user context

	createdEntry, err := s.timeRepo.Create(ctx, entry)
	if err != nil {
		return nil, fmt.Errorf("failed to create time entry: %w", err)
	}

	s.logger.Info("Time entry created successfully", "entry_id", createdEntry.ID, "task_id", createdEntry.TaskID)

	return createdEntry, nil
}

// GetTimeEntry retrieves a time entry by ID
func (s *TimeService) GetTimeEntry(ctx context.Context, id uuid.UUID) (*entities.TimeEntry, error) {
	entry, err := s.timeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("time entry not found: %w", err)
	}

	return entry, nil
}

// UpdateTimeEntry updates a time entry's information
func (s *TimeService) UpdateTimeEntry(ctx context.Context, id uuid.UUID, req ports.UpdateTimeEntryRequest) (*entities.TimeEntry, error) {
	// Get existing time entry
	existingEntry, err := s.timeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("time entry not found: %w", err)
	}

	// Update fields
	if req.Description != nil {
		existingEntry.Description = req.Description
	}
	if req.StartTime != nil {
		existingEntry.StartTime = *req.StartTime
	}
	if req.EndTime != nil {
		existingEntry.EndTime = req.EndTime
	}
	if req.Hours != nil {
		existingEntry.Hours = *req.Hours
	}

	// Recalculate hours if start/end time changed
	if req.StartTime != nil || req.EndTime != nil {
		if existingEntry.EndTime != nil {
			existingEntry.Hours = existingEntry.EndTime.Sub(existingEntry.StartTime).Hours()
		}
	}

	existingEntry.UpdatedAt = time.Now()

	updatedEntry, err := s.timeRepo.Update(ctx, existingEntry)
	if err != nil {
		return nil, fmt.Errorf("failed to update time entry: %w", err)
	}

	s.logger.Info("Time entry updated successfully", "entry_id", updatedEntry.ID)

	return updatedEntry, nil
}

// DeleteTimeEntry deletes a time entry
func (s *TimeService) DeleteTimeEntry(ctx context.Context, id uuid.UUID) error {
	// Check if time entry exists
	_, err := s.timeRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("time entry not found: %w", err)
	}

	err = s.timeRepo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete time entry: %w", err)
	}

	s.logger.Info("Time entry deleted successfully", "entry_id", id)

	return nil
}

// ListTimeEntries retrieves time entries with filtering and pagination
func (s *TimeService) ListTimeEntries(ctx context.Context, filter ports.TimeEntryFilter) ([]*entities.TimeEntry, int, error) {
	entries, total, err := s.timeRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list time entries: %w", err)
	}

	return entries, total, nil
}

// StartTimeTracking starts time tracking for a task
func (s *TimeService) StartTimeTracking(ctx context.Context, userID uuid.UUID, taskID int) (*entities.TimeEntry, error) {
	// Check if user already has an active time entry
	activeEntry, err := s.timeRepo.GetActiveEntry(ctx, userID)
	if err == nil && activeEntry != nil {
		return nil, fmt.Errorf("user already has an active time entry")
	}

	// Verify task exists
	_, err = s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	// Create new time entry
	entry := &entities.TimeEntry{
		ID:        uuid.New(),
		TaskID:    taskID,
		UserID:    userID,
		StartTime: time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	createdEntry, err := s.timeRepo.Create(ctx, entry)
	if err != nil {
		return nil, fmt.Errorf("failed to start time tracking: %w", err)
	}

	s.logger.Info("Time tracking started", "entry_id", createdEntry.ID, "user_id", userID, "task_id", taskID)

	return createdEntry, nil
}

// StopTimeTracking stops the active time tracking for a user
func (s *TimeService) StopTimeTracking(ctx context.Context, userID uuid.UUID) (*entities.TimeEntry, error) {
	// Get active time entry
	activeEntry, err := s.timeRepo.GetActiveEntry(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("no active time entry found: %w", err)
	}

	// Stop the timer
	now := time.Now()
	activeEntry.EndTime = &now
	activeEntry.Hours = now.Sub(activeEntry.StartTime).Hours()
	activeEntry.UpdatedAt = now

	updatedEntry, err := s.timeRepo.Update(ctx, activeEntry)
	if err != nil {
		return nil, fmt.Errorf("failed to stop time tracking: %w", err)
	}

	s.logger.Info("Time tracking stopped", "entry_id", updatedEntry.ID, "user_id", userID, "hours", updatedEntry.Hours)

	return updatedEntry, nil
}

// GetActiveTimeEntry gets the active time entry for a user
func (s *TimeService) GetActiveTimeEntry(ctx context.Context, userID uuid.UUID) (*entities.TimeEntry, error) {
	entry, err := s.timeRepo.GetActiveEntry(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("no active time entry found: %w", err)
	}

	return entry, nil
}

// GetTimeReport generates a time report based on the provided criteria
func (s *TimeService) GetTimeReport(ctx context.Context, req ports.TimeReportRequest) (*ports.TimeReport, error) {
	filter := ports.TimeEntryFilter{
		UserID:    req.UserID,
		ProjectID: req.ProjectID,
		StartDate: &req.StartDate,
		EndDate:   &req.EndDate,
		Limit:     1000, // Set a high limit for reports
	}

	entries, _, err := s.timeRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get time entries for report: %w", err)
	}

	// Calculate total hours
	totalHours := 0.0
	for _, entry := range entries {
		totalHours += entry.Hours
	}

	// Group entries based on GroupBy parameter
	reportEntries := make([]ports.TimeReportEntry, 0)
	summary := make(map[string]interface{})

	switch req.GroupBy {
	case "user":
		// Group by user
		userHours := make(map[string]float64)
		for _, entry := range entries {
			userHours[entry.UserID.String()] += entry.Hours
		}
		for userID, hours := range userHours {
			reportEntries = append(reportEntries, ports.TimeReportEntry{
				Label:      userID, // In a real implementation, this would be the user's name
				Hours:      hours,
				Percentage: (hours / totalHours) * 100,
			})
		}
	case "project":
		// Group by project (would need project info from entries)
		projectHours := make(map[string]float64)
		for _, entry := range entries {
			projectKey := fmt.Sprintf("project_%d", entry.Task.ProjectID) // Assuming task is loaded
			projectHours[projectKey] += entry.Hours
		}
		for projectID, hours := range projectHours {
			reportEntries = append(reportEntries, ports.TimeReportEntry{
				Label:      projectID,
				Hours:      hours,
				Percentage: (hours / totalHours) * 100,
			})
		}
	case "task":
		// Group by task
		taskHours := make(map[string]float64)
		for _, entry := range entries {
			taskKey := fmt.Sprintf("task_%d", entry.TaskID)
			taskHours[taskKey] += entry.Hours
		}
		for taskID, hours := range taskHours {
			reportEntries = append(reportEntries, ports.TimeReportEntry{
				Label:      taskID,
				Hours:      hours,
				Percentage: (hours / totalHours) * 100,
			})
		}
	default:
		// Default grouping by day
		dayHours := make(map[string]float64)
		for _, entry := range entries {
			day := entry.StartTime.Format("2006-01-02")
			dayHours[day] += entry.Hours
		}
		for day, hours := range dayHours {
			reportEntries = append(reportEntries, ports.TimeReportEntry{
				Label:      day,
				Hours:      hours,
				Percentage: (hours / totalHours) * 100,
			})
		}
	}

	summary["total_entries"] = len(entries)
	summary["average_hours_per_day"] = totalHours / float64(req.EndDate.Sub(req.StartDate).Hours()/24)

	report := &ports.TimeReport{
		TotalHours: totalHours,
		Entries:    reportEntries,
		Summary:    summary,
	}

	return report, nil
}
