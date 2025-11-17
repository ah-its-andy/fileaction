package database

import (
	"fmt"

	"github.com/andi/fileaction/backend/models"
	"github.com/google/uuid"
)

// WorkflowRepo handles workflow database operations
type WorkflowRepo struct {
	db *DB
}

// NewWorkflowRepo creates a new workflow repository
func NewWorkflowRepo(db *DB) *WorkflowRepo {
	return &WorkflowRepo{db: db}
}

// Create creates a new workflow
func (r *WorkflowRepo) Create(workflow *models.Workflow) error {
	if workflow.ID == "" {
		workflow.ID = uuid.New().String()
	}

	model := FromWorkflow(workflow)
	if err := r.db.conn.Create(model).Error; err != nil {
		return err
	}

	*workflow = *model.ToWorkflow()
	return nil
}

// GetByID retrieves a workflow by ID
func (r *WorkflowRepo) GetByID(id string) (*models.Workflow, error) {
	var model WorkflowModel
	if err := r.db.conn.Where("id = ?", id).First(&model).Error; err != nil {
		return nil, fmt.Errorf("workflow not found")
	}
	return model.ToWorkflow(), nil
}

// GetByName retrieves a workflow by name
func (r *WorkflowRepo) GetByName(name string) (*models.Workflow, error) {
	var model WorkflowModel
	if err := r.db.conn.Where("name = ?", name).First(&model).Error; err != nil {
		return nil, fmt.Errorf("workflow not found")
	}
	return model.ToWorkflow(), nil
}

// List retrieves all workflows
func (r *WorkflowRepo) List() ([]*models.Workflow, error) {
	var modelList []WorkflowModel
	if err := r.db.conn.Order("created_at DESC").Find(&modelList).Error; err != nil {
		return nil, err
	}

	workflows := make([]*models.Workflow, len(modelList))
	for i, model := range modelList {
		workflows[i] = model.ToWorkflow()
	}
	return workflows, nil
}

// Update updates a workflow
func (r *WorkflowRepo) Update(workflow *models.Workflow) error {
	model := FromWorkflow(workflow)
	if err := r.db.conn.Save(model).Error; err != nil {
		return err
	}
	*workflow = *model.ToWorkflow()
	return nil
}

// Delete deletes a workflow
func (r *WorkflowRepo) Delete(id string) error {
	result := r.db.conn.Delete(&WorkflowModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("workflow not found")
	}
	return nil
}
