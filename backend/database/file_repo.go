package database

import (
	"fmt"

	"github.com/andi/fileaction/backend/models"
	"github.com/google/uuid"
)

// FileRepo handles file database operations
type FileRepo struct {
	db *DB
}

// NewFileRepo creates a new file repository
func NewFileRepo(db *DB) *FileRepo {
	return &FileRepo{db: db}
}

// Create creates a new file record
func (r *FileRepo) Create(file *models.File) error {
	if file.ID == "" {
		file.ID = uuid.New().String()
	}

	model := FromFile(file)
	if err := r.db.conn.Create(model).Error; err != nil {
		return err
	}

	*file = *model.ToFile()
	return nil
}

// GetByWorkflowAndPath retrieves a file by workflow ID and path
func (r *FileRepo) GetByWorkflowAndPath(workflowID, filePath string) (*models.File, error) {
	var model FileModel
	err := r.db.conn.Where("workflow_id = ? AND file_path = ?", workflowID, filePath).First(&model).Error
	if err != nil {
		return nil, nil
	}
	return model.ToFile(), nil
}

// Update updates a file record
func (r *FileRepo) Update(file *models.File) error {
	model := FromFile(file)
	result := r.db.conn.Save(model)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("file not found")
	}
	*file = *model.ToFile()
	return nil
}

// ListByWorkflow retrieves all files for a workflow
func (r *FileRepo) ListByWorkflow(workflowID string, limit, offset int) ([]*models.File, error) {
	var modelList []FileModel
	err := r.db.conn.Where("workflow_id = ?", workflowID).
		Order("file_path").
		Limit(limit).
		Offset(offset).
		Find(&modelList).Error
	if err != nil {
		return nil, err
	}

	files := make([]*models.File, len(modelList))
	for i, model := range modelList {
		files[i] = model.ToFile()
	}
	return files, nil
}

// CountByWorkflow counts files for a workflow
func (r *FileRepo) CountByWorkflow(workflowID string) (int, error) {
	var count int64
	err := r.db.conn.Model(&FileModel{}).Where("workflow_id = ?", workflowID).Count(&count).Error
	return int(count), err
}

// DeleteByWorkflow deletes all files for a workflow
func (r *FileRepo) DeleteByWorkflow(workflowID string) error {
	return r.db.conn.Delete(&FileModel{}, "workflow_id = ?", workflowID).Error
}
