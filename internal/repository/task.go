package repository

import (
	"sync"
	"time"

	"github.com/xuecangming/onedrive-storage/internal/common/errors"
	"github.com/xuecangming/onedrive-storage/internal/common/types"
)

// TaskRepository handles task storage
type TaskRepository struct {
	tasks map[string]*types.Task
	mu    sync.RWMutex
}

// NewTaskRepository creates a new task repository
func NewTaskRepository() *TaskRepository {
	return &TaskRepository{
		tasks: make(map[string]*types.Task),
	}
}

// Create creates a new task
func (r *TaskRepository) Create(task *types.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[task.ID]; exists {
		return errors.NewConflictError("task already exists")
	}

	r.tasks[task.ID] = task
	return nil
}

// Get retrieves a task by ID
func (r *TaskRepository) Get(id string) (*types.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, exists := r.tasks[id]
	if !exists {
		return nil, errors.NewNotFoundError("task not found")
	}

	return task, nil
}

// Update updates a task
func (r *TaskRepository) Update(task *types.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[task.ID]; !exists {
		return errors.NewNotFoundError("task not found")
	}

	task.UpdatedAt = time.Now()
	r.tasks[task.ID] = task
	return nil
}

// List returns all tasks (with optional filtering, omitted for now)
func (r *TaskRepository) List() ([]*types.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tasks := make([]*types.Task, 0, len(r.tasks))
	for _, t := range r.tasks {
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// Delete deletes a task
func (r *TaskRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.tasks, id)
	return nil
}

// FindByMetadata finds a task by metadata key-value pair
func (r *TaskRepository) FindByMetadata(key string, value interface{}) (*types.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, task := range r.tasks {
		if task.Metadata != nil {
			if v, ok := task.Metadata[key]; ok && v == value {
				return task, nil
			}
		}
	}
	return nil, errors.NewNotFoundError("task not found")
}
