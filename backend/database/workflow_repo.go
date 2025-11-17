package database

import (
	"database/sql"
	"fmt"
	"time"

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
	now := time.Now()
	workflow.CreatedAt = now
	workflow.UpdatedAt = now

	query := `
		INSERT INTO workflows (id, name, description, yaml_content, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.conn.Exec(query,
		workflow.ID,
		workflow.Name,
		workflow.Description,
		workflow.YAMLContent,
		workflow.Enabled,
		workflow.CreatedAt,
		workflow.UpdatedAt,
	)
	return err
}

// GetByID retrieves a workflow by ID
func (r *WorkflowRepo) GetByID(id string) (*models.Workflow, error) {
	query := `
		SELECT id, name, description, yaml_content, enabled, created_at, updated_at
		FROM workflows
		WHERE id = ?
	`
	var workflow models.Workflow
	err := r.db.conn.QueryRow(query, id).Scan(
		&workflow.ID,
		&workflow.Name,
		&workflow.Description,
		&workflow.YAMLContent,
		&workflow.Enabled,
		&workflow.CreatedAt,
		&workflow.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("workflow not found")
	}
	if err != nil {
		return nil, err
	}
	return &workflow, nil
}

// GetByName retrieves a workflow by name
func (r *WorkflowRepo) GetByName(name string) (*models.Workflow, error) {
	query := `
		SELECT id, name, description, yaml_content, enabled, created_at, updated_at
		FROM workflows
		WHERE name = ?
	`
	var workflow models.Workflow
	err := r.db.conn.QueryRow(query, name).Scan(
		&workflow.ID,
		&workflow.Name,
		&workflow.Description,
		&workflow.YAMLContent,
		&workflow.Enabled,
		&workflow.CreatedAt,
		&workflow.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("workflow not found")
	}
	if err != nil {
		return nil, err
	}
	return &workflow, nil
}

// List retrieves all workflows
func (r *WorkflowRepo) List() ([]*models.Workflow, error) {
	query := `
		SELECT id, name, description, yaml_content, enabled, created_at, updated_at
		FROM workflows
		ORDER BY created_at DESC
	`
	rows, err := r.db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workflows []*models.Workflow
	for rows.Next() {
		var workflow models.Workflow
		err := rows.Scan(
			&workflow.ID,
			&workflow.Name,
			&workflow.Description,
			&workflow.YAMLContent,
			&workflow.Enabled,
			&workflow.CreatedAt,
			&workflow.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		workflows = append(workflows, &workflow)
	}
	return workflows, rows.Err()
}

// Update updates a workflow
func (r *WorkflowRepo) Update(workflow *models.Workflow) error {
	workflow.UpdatedAt = time.Now()
	query := `
		UPDATE workflows
		SET name = ?, description = ?, yaml_content = ?, enabled = ?, updated_at = ?
		WHERE id = ?
	`
	result, err := r.db.conn.Exec(query,
		workflow.Name,
		workflow.Description,
		workflow.YAMLContent,
		workflow.Enabled,
		workflow.UpdatedAt,
		workflow.ID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("workflow not found")
	}
	return nil
}

// Delete deletes a workflow
func (r *WorkflowRepo) Delete(id string) error {
	query := `DELETE FROM workflows WHERE id = ?`
	result, err := r.db.conn.Exec(query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("workflow not found")
	}
	return nil
}
