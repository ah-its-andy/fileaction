package database

import (
	"fmt"

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

	model := FromTask(task)
	if err := r.db.conn.Create(model).Error; err != nil {
		return err
	}

	*task = *model.ToTask()
	return nil
}

// GetByID retrieves a task by ID
func (r *TaskRepo) GetByID(id string) (*models.Task, error) {
	var model TaskModel
	if err := r.db.conn.Where("id = ?", id).First(&model).Error; err != nil {
		return nil, fmt.Errorf("task not found")
	}
	return model.ToTask(), nil
}

// List retrieves tasks with optional filters
func (r *TaskRepo) List(workflowID, status string, limit, offset int) ([]*models.Task, error) {
	query := r.db.conn.Model(&TaskModel{})

	if workflowID != "" {
		query = query.Where("workflow_id = ?", workflowID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var modelList []TaskModel
	err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&modelList).Error
	if err != nil {
		return nil, err
	}

	tasks := make([]*models.Task, len(modelList))
	for i, model := range modelList {
		tasks[i] = model.ToTask()
	}
	return tasks, nil
}

// Count counts tasks with optional filters
func (r *TaskRepo) Count(workflowID, status string) (int, error) {
	query := r.db.conn.Model(&TaskModel{})

	if workflowID != "" {
		query = query.Where("workflow_id = ?", workflowID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var count int64
	err := query.Count(&count).Error
	return int(count), err
}

// Update updates a task
func (r *TaskRepo) Update(task *models.Task) error {
	model := FromTask(task)
	result := r.db.conn.Save(model)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("task not found")
	}
	*task = *model.ToTask()
	return nil
}

// UpdateStatus updates only the status of a task
func (r *TaskRepo) UpdateStatus(id, status string) error {
	result := r.db.conn.Model(&TaskModel{}).Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("task not found")
	}
	return nil
}

// Delete deletes a task
func (r *TaskRepo) Delete(id string) error {
	result := r.db.conn.Delete(&TaskModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("task not found")
	}
	return nil
}

// DeleteByWorkflow deletes all tasks for a workflow
func (r *TaskRepo) DeleteByWorkflow(workflowID string) error {
	return r.db.conn.Delete(&TaskModel{}, "workflow_id = ?", workflowID).Error
}

// GetPendingTasks retrieves all pending tasks
func (r *TaskRepo) GetPendingTasks(limit int) ([]*models.Task, error) {
	var modelList []TaskModel
	err := r.db.conn.Where("status = ?", models.TaskStatusPending).
		Order("created_at").
		Limit(limit).
		Find(&modelList).Error
	if err != nil {
		return nil, err
	}

	tasks := make([]*models.Task, len(modelList))
	for i, model := range modelList {
		tasks[i] = model.ToTask()
	}
	return tasks, nil
}
