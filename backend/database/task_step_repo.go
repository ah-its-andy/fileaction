package database

import (
	"fmt"

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

	model := FromTaskStep(step)
	if err := r.db.conn.Create(model).Error; err != nil {
		return err
	}

	*step = *model.ToTaskStep()
	return nil
}

// GetByTaskID retrieves all steps for a task
func (r *TaskStepRepo) GetByTaskID(taskID string) ([]*models.TaskStep, error) {
	var modelList []TaskStepModel
	err := r.db.conn.Where("task_id = ?", taskID).
		Order("created_at").
		Find(&modelList).Error
	if err != nil {
		return nil, err
	}

	steps := make([]*models.TaskStep, len(modelList))
	for i, model := range modelList {
		steps[i] = model.ToTaskStep()
	}
	return steps, nil
}

// Update updates a task step
func (r *TaskStepRepo) Update(step *models.TaskStep) error {
	model := FromTaskStep(step)
	result := r.db.conn.Save(model)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("task step not found")
	}
	*step = *model.ToTaskStep()
	return nil
}

// DeleteByTaskID deletes all steps for a task
func (r *TaskStepRepo) DeleteByTaskID(taskID string) error {
	return r.db.conn.Delete(&TaskStepModel{}, "task_id = ?", taskID).Error
}
