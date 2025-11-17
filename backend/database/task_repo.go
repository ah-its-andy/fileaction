package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/andi/fileaction/backend/models"
	"github.com/google/uuid"
)

// TaskRepo handles task database operations
type TaskRepo struct {
	db *DB
}

// NewTaskRepo creates a new task repository
func NewTaskRepo(db *DB) *TaskRepo {
	return &TaskRepo{db: db}
}

// Create creates a new task
func (r *TaskRepo) Create(task *models.Task) error {
	if task.ID == "" {
		task.ID = uuid.New().String()
	}
	now := time.Now()
	task.CreatedAt = now
	task.UpdatedAt = now

	query := `
		INSERT INTO tasks (id, workflow_id, file_id, input_path, output_path, status, log_text, error_message, started_at, completed_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.conn.Exec(query,
		task.ID,
		task.WorkflowID,
		task.FileID,
		task.InputPath,
		task.OutputPath,
		task.Status,
		task.LogText,
		task.ErrorMessage,
		task.StartedAt,
		task.CompletedAt,
		task.CreatedAt,
		task.UpdatedAt,
	)
	return err
}

// GetByID retrieves a task by ID
func (r *TaskRepo) GetByID(id string) (*models.Task, error) {
	query := `
		SELECT id, workflow_id, file_id, input_path, output_path, status, log_text, error_message, started_at, completed_at, created_at, updated_at
		FROM tasks
		WHERE id = ?
	`
	var task models.Task
	err := r.db.conn.QueryRow(query, id).Scan(
		&task.ID,
		&task.WorkflowID,
		&task.FileID,
		&task.InputPath,
		&task.OutputPath,
		&task.Status,
		&task.LogText,
		&task.ErrorMessage,
		&task.StartedAt,
		&task.CompletedAt,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found")
	}
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// List retrieves tasks with optional filters
func (r *TaskRepo) List(workflowID, status string, limit, offset int) ([]*models.Task, error) {
	query := `
		SELECT id, workflow_id, file_id, input_path, output_path, status, log_text, error_message, started_at, completed_at, created_at, updated_at
		FROM tasks
		WHERE 1=1
	`
	args := []interface{}{}

	if workflowID != "" {
		query += " AND workflow_id = ?"
		args = append(args, workflowID)
	}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		var task models.Task
		err := rows.Scan(
			&task.ID,
			&task.WorkflowID,
			&task.FileID,
			&task.InputPath,
			&task.OutputPath,
			&task.Status,
			&task.LogText,
			&task.ErrorMessage,
			&task.StartedAt,
			&task.CompletedAt,
			&task.CreatedAt,
			&task.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, &task)
	}
	return tasks, rows.Err()
}

// Count counts tasks with optional filters
func (r *TaskRepo) Count(workflowID, status string) (int, error) {
	query := `SELECT COUNT(*) FROM tasks WHERE 1=1`
	args := []interface{}{}

	if workflowID != "" {
		query += " AND workflow_id = ?"
		args = append(args, workflowID)
	}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}

	var count int
	err := r.db.conn.QueryRow(query, args...).Scan(&count)
	return count, err
}

// Update updates a task
func (r *TaskRepo) Update(task *models.Task) error {
	task.UpdatedAt = time.Now()
	query := `
		UPDATE tasks
		SET status = ?, log_text = ?, error_message = ?, started_at = ?, completed_at = ?, updated_at = ?
		WHERE id = ?
	`
	result, err := r.db.conn.Exec(query,
		task.Status,
		task.LogText,
		task.ErrorMessage,
		task.StartedAt,
		task.CompletedAt,
		task.UpdatedAt,
		task.ID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("task not found")
	}
	return nil
}

// UpdateStatus updates only the status of a task
func (r *TaskRepo) UpdateStatus(id, status string) error {
	query := `UPDATE tasks SET status = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.conn.Exec(query, status, time.Now(), id)
	return err
}

// Delete deletes a task
func (r *TaskRepo) Delete(id string) error {
	query := `DELETE FROM tasks WHERE id = ?`
	result, err := r.db.conn.Exec(query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("task not found")
	}
	return nil
}

// DeleteByWorkflow deletes all tasks for a workflow
func (r *TaskRepo) DeleteByWorkflow(workflowID string) error {
	query := `DELETE FROM tasks WHERE workflow_id = ?`
	_, err := r.db.conn.Exec(query, workflowID)
	return err
}

// GetPendingTasks retrieves all pending tasks
func (r *TaskRepo) GetPendingTasks(limit int) ([]*models.Task, error) {
	query := `
		SELECT id, workflow_id, file_id, input_path, output_path, status, log_text, error_message, started_at, completed_at, created_at, updated_at
		FROM tasks
		WHERE status = ?
		ORDER BY created_at
		LIMIT ?
	`
	rows, err := r.db.conn.Query(query, models.TaskStatusPending, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		var task models.Task
		err := rows.Scan(
			&task.ID,
			&task.WorkflowID,
			&task.FileID,
			&task.InputPath,
			&task.OutputPath,
			&task.Status,
			&task.LogText,
			&task.ErrorMessage,
			&task.StartedAt,
			&task.CompletedAt,
			&task.CreatedAt,
			&task.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, &task)
	}
	return tasks, rows.Err()
}
