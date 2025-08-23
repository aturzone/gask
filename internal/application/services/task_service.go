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

// TaskService handles task-related operations
type TaskService struct {
	taskRepo    ports.TaskRepository
	projectRepo ports.ProjectRepository
	userRepo    ports.UserRepository
	logger      *logger.Logger
}

// NewTaskService creates a new task service
func NewTaskService(taskRepo ports.TaskRepository, projectRepo ports.ProjectRepository, userRepo ports.UserRepository, logger *logger.Logger) *TaskService {
	return &TaskService{
		taskRepo:    taskRepo,
		projectRepo: projectRepo,
		userRepo:    userRepo,
		logger:      logger,
	}
}

// CreateTask creates a new task
func (s *TaskService) CreateTask(ctx context.Context, req ports.CreateTaskRequest) (*entities.Task, error) {
	// Verify project exists
	_, err := s.projectRepo.GetByID(ctx, req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	// Verify assignee exists if provided
	if req.AssigneeID != nil {
		_, err := s.userRepo.GetByID(ctx, *req.AssigneeID)
		if err != nil {
			return nil, fmt.Errorf("assignee not found: %w", err)
		}
	}

	// Create task entity
	task := &entities.Task{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		Priority:    req.Priority,
		ProjectID:   req.ProjectID,
		AssigneeID:  req.AssigneeID,
		DueDate:     req.DueDate,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Get created by from context (would need to be passed in real implementation)
	// For now, using a placeholder UUID
	task.CreatedBy = uuid.New() // This should come from the authenticated user context

	createdTask, err := s.taskRepo.Create(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	s.logger.Info("Task created successfully", "task_id", createdTask.ID, "title", createdTask.Title)

	return createdTask, nil
}

// GetTask retrieves a task by ID
func (s *TaskService) GetTask(ctx context.Context, id int) (*entities.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	return task, nil
}

// UpdateTask updates a task's information
func (s *TaskService) UpdateTask(ctx context.Context, id int, req ports.UpdateTaskRequest) (*entities.Task, error) {
	// Get existing task
	existingTask, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	// Verify assignee exists if provided
	if req.AssigneeID != nil {
		_, err := s.userRepo.GetByID(ctx, *req.AssigneeID)
		if err != nil {
			return nil, fmt.Errorf("assignee not found: %w", err)
		}
	}

	// Update fields
	if req.Title != nil {
		existingTask.Title = *req.Title
	}
	if req.Description != nil {
		existingTask.Description = req.Description
	}
	if req.Status != nil {
		existingTask.Status = *req.Status
	}
	if req.Priority != nil {
		existingTask.Priority = *req.Priority
	}
	if req.AssigneeID != nil {
		existingTask.AssigneeID = req.AssigneeID
	}
	if req.DueDate != nil {
		existingTask.DueDate = req.DueDate
	}

	existingTask.UpdatedAt = time.Now()

	updatedTask, err := s.taskRepo.Update(ctx, existingTask)
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	s.logger.Info("Task updated successfully", "task_id", updatedTask.ID, "title", updatedTask.Title)

	return updatedTask, nil
}

// DeleteTask deletes a task
func (s *TaskService) DeleteTask(ctx context.Context, id int) error {
	// Check if task exists
	_, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	err = s.taskRepo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	s.logger.Info("Task deleted successfully", "task_id", id)

	return nil
}

// ListTasks retrieves tasks with filtering and pagination
func (s *TaskService) ListTasks(ctx context.Context, filter ports.TaskFilter) ([]*entities.Task, int, error) {
	tasks, total, err := s.taskRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tasks: %w", err)
	}

	return tasks, total, nil
}

// AssignTask assigns a task to a user
func (s *TaskService) AssignTask(ctx context.Context, taskID int, assigneeID uuid.UUID) (*entities.Task, error) {
	// Verify assignee exists
	_, err := s.userRepo.GetByID(ctx, assigneeID)
	if err != nil {
		return nil, fmt.Errorf("assignee not found: %w", err)
	}

	err = s.taskRepo.Assign(ctx, taskID, assigneeID)
	if err != nil {
		return nil, fmt.Errorf("failed to assign task: %w", err)
	}

	// Get updated task
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated task: %w", err)
	}

	s.logger.Info("Task assigned successfully", "task_id", taskID, "assignee_id", assigneeID)

	return task, nil
}

// UpdateTaskStatus updates a task's status
func (s *TaskService) UpdateTaskStatus(ctx context.Context, taskID int, status entities.TaskStatus) (*entities.Task, error) {
	err := s.taskRepo.UpdateStatus(ctx, taskID, status)
	if err != nil {
		return nil, fmt.Errorf("failed to update task status: %w", err)
	}

	// Get updated task
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated task: %w", err)
	}

	s.logger.Info("Task status updated successfully", "task_id", taskID, "status", status)

	return task, nil
}

// GetTasksNearDeadline gets tasks with deadlines approaching within specified days
func (s *TaskService) GetTasksNearDeadline(ctx context.Context, days int) ([]*entities.Task, error) {
	tasks, err := s.taskRepo.GetNearDeadline(ctx, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks near deadline: %w", err)
	}

	return tasks, nil
}

// GetOverdueTasks gets all overdue tasks
func (s *TaskService) GetOverdueTasks(ctx context.Context) ([]*entities.Task, error) {
	tasks, err := s.taskRepo.GetOverdue(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get overdue tasks: %w", err)
	}

	return tasks, nil
}

// GetUserTasks gets all tasks assigned to a user
func (s *TaskService) GetUserTasks(ctx context.Context, userID uuid.UUID) ([]*entities.Task, error) {
	tasks, err := s.taskRepo.GetByAssignee(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user tasks: %w", err)
	}

	return tasks, nil
}
