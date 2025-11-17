package models

import (
	"time"
)

// Workflow represents a workflow definition
type Workflow struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	YAMLContent string    `json:"yaml_content"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// File represents an indexed file
type File struct {
	ID            string    `json:"id"`
	WorkflowID    string    `json:"workflow_id"`
	FilePath      string    `json:"file_path"`
	FileMD5       string    `json:"file_md5"`
	FileSize      int64     `json:"file_size"`
	LastScannedAt time.Time `json:"last_scanned_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Task represents a conversion task
type Task struct {
	ID           string     `json:"id"`
	WorkflowID   string     `json:"workflow_id"`
	FileID       string     `json:"file_id"`
	InputPath    string     `json:"input_path"`
	OutputPath   string     `json:"output_path"`
	Status       string     `json:"status"` // pending, running, completed, failed, cancelled
	LogText      string     `json:"log_text,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// TaskStep represents a step within a task
type TaskStep struct {
	ID          string     `json:"id"`
	TaskID      string     `json:"task_id"`
	Name        string     `json:"name"`
	Command     string     `json:"command"`
	Status      string     `json:"status"` // pending, running, completed, failed, skipped
	ExitCode    *int       `json:"exit_code,omitempty"`
	Stdout      string     `json:"stdout,omitempty"`
	Stderr      string     `json:"stderr,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// TaskStatus constants
const (
	TaskStatusPending   = "pending"
	TaskStatusRunning   = "running"
	TaskStatusCompleted = "completed"
	TaskStatusFailed    = "failed"
	TaskStatusCancelled = "cancelled"
)

// StepStatus constants
const (
	StepStatusPending   = "pending"
	StepStatusRunning   = "running"
	StepStatusCompleted = "completed"
	StepStatusFailed    = "failed"
	StepStatusSkipped   = "skipped"
)
