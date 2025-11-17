package database

import (
	"fmt"
	"time"

	"github.com/andi/fileaction/backend/models"
	"github.com/google/uuid"
)

// TaskStepRepo handles task step database operations
type TaskStepRepo struct {
	db *DB
}

// NewTaskStepRepo creates a new task step repository
func NewTaskStepRepo(db *DB) *TaskStepRepo {
	return &TaskStepRepo{db: db}
}

// Create creates a new task step
func (r *TaskStepRepo) Create(step *models.TaskStep) error {
	if step.ID == "" {
		step.ID = uuid.New().String()
	}
	now := time.Now()
	step.CreatedAt = now
	step.UpdatedAt = now

	query := `
		INSERT INTO task_steps (id, task_id, name, command, status, exit_code, stdout, stderr, started_at, completed_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.conn.Exec(query,
		step.ID,
		step.TaskID,
		step.Name,
		step.Command,
		step.Status,
		step.ExitCode,
		step.Stdout,
		step.Stderr,
		step.StartedAt,
		step.CompletedAt,
		step.CreatedAt,
		step.UpdatedAt,
	)
	return err
}

// GetByTaskID retrieves all steps for a task
func (r *TaskStepRepo) GetByTaskID(taskID string) ([]*models.TaskStep, error) {
	query := `
		SELECT id, task_id, name, command, status, exit_code, stdout, stderr, started_at, completed_at, created_at, updated_at
		FROM task_steps
		WHERE task_id = ?
		ORDER BY created_at
	`
	rows, err := r.db.conn.Query(query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []*models.TaskStep
	for rows.Next() {
		var step models.TaskStep
		err := rows.Scan(
			&step.ID,
			&step.TaskID,
			&step.Name,
			&step.Command,
			&step.Status,
			&step.ExitCode,
			&step.Stdout,
			&step.Stderr,
			&step.StartedAt,
			&step.CompletedAt,
			&step.CreatedAt,
			&step.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		steps = append(steps, &step)
	}
	return steps, rows.Err()
}

// Update updates a task step
func (r *TaskStepRepo) Update(step *models.TaskStep) error {
	step.UpdatedAt = time.Now()
	query := `
		UPDATE task_steps
		SET status = ?, exit_code = ?, stdout = ?, stderr = ?, started_at = ?, completed_at = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := r.db.conn.Exec(query,
		step.Status,
		step.ExitCode,
		step.Stdout,
		step.Stderr,
		step.StartedAt,
		step.CompletedAt,
		step.UpdatedAt,
		step.ID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("task step not found")
	}
	return nil
}

// DeleteByTaskID deletes all steps for a task
func (r *TaskStepRepo) DeleteByTaskID(taskID string) error {
	query := `DELETE FROM task_steps WHERE task_id = ?`
	_, err := r.db.conn.Exec(query, taskID)
	return err
}
