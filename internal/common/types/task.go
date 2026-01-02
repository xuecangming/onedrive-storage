package types

import "time"

// TaskStatus represents the status of an async task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// TaskType represents the type of task
type TaskType string

const (
	TaskTypeCopy     TaskType = "copy"
	TaskTypeMove     TaskType = "move"
	TaskTypeDelete   TaskType = "delete"
	TaskTypeSync     TaskType = "sync"
	TaskTypeUpload   TaskType = "upload"
	TaskTypeDownload TaskType = "download"
)

// Task represents an asynchronous background task
type Task struct {
	ID          string                 `json:"id"`
	Type        TaskType               `json:"type"`
	Status      TaskStatus             `json:"status"`
	Progress    int                    `json:"progress"` // 0-100
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
