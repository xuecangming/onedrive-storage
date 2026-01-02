package task

import (
	"time"

	"github.com/xuecangming/onedrive-storage/internal/common/types"
	"github.com/xuecangming/onedrive-storage/internal/common/utils"
	"github.com/xuecangming/onedrive-storage/internal/repository"
)

// Service handles task operations
type Service struct {
	repo *repository.TaskRepository
}

// NewService creates a new task service
func NewService(repo *repository.TaskRepository) *Service {
	return &Service{
		repo: repo,
	}
}

// CreateTask creates a new task
func (s *Service) CreateTask(taskType types.TaskType, metadata map[string]interface{}) (*types.Task, error) {
	now := time.Now()
	task := &types.Task{
		ID:        utils.GenerateID(),
		Type:      taskType,
		Status:    types.TaskStatusPending,
		Progress:  0,
		Metadata:  metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(task); err != nil {
		return nil, err
	}

	return task, nil
}

// GetTask retrieves a task
func (s *Service) GetTask(id string) (*types.Task, error) {
	return s.repo.Get(id)
}

// UpdateProgress updates task progress
func (s *Service) UpdateProgress(id string, progress int) error {
	task, err := s.repo.Get(id)
	if err != nil {
		return err
	}

	task.Progress = progress
	if progress > 0 && task.Status == types.TaskStatusPending {
		task.Status = types.TaskStatusRunning
	}
	
	return s.repo.Update(task)
}

// CompleteTask marks a task as completed
func (s *Service) CompleteTask(id string, result map[string]interface{}) error {
	task, err := s.repo.Get(id)
	if err != nil {
		return err
	}

	now := time.Now()
	task.Status = types.TaskStatusCompleted
	task.Progress = 100
	task.Result = result
	task.CompletedAt = &now
	
	return s.repo.Update(task)
}

// FailTask marks a task as failed
func (s *Service) FailTask(id string, errorMsg string) error {
	task, err := s.repo.Get(id)
	if err != nil {
		return err
	}

	now := time.Now()
	task.Status = types.TaskStatusFailed
	task.Error = errorMsg
	task.CompletedAt = &now
	
	return s.repo.Update(task)
}

// ListTasks lists all tasks
func (s *Service) ListTasks() ([]*types.Task, error) {
	return s.repo.List()
}
