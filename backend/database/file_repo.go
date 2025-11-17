package database

import (
	"database/sql"
	"fmt"
	"time"

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
	now := time.Now()
	file.CreatedAt = now
	file.UpdatedAt = now

	query := `
		INSERT INTO files (id, workflow_id, file_path, file_md5, file_size, last_scanned_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.conn.Exec(query,
		file.ID,
		file.WorkflowID,
		file.FilePath,
		file.FileMD5,
		file.FileSize,
		file.LastScannedAt,
		file.CreatedAt,
		file.UpdatedAt,
	)
	return err
}

// GetByWorkflowAndPath retrieves a file by workflow ID and path
func (r *FileRepo) GetByWorkflowAndPath(workflowID, filePath string) (*models.File, error) {
	query := `
		SELECT id, workflow_id, file_path, file_md5, file_size, last_scanned_at, created_at, updated_at
		FROM files
		WHERE workflow_id = ? AND file_path = ?
	`
	var file models.File
	err := r.db.conn.QueryRow(query, workflowID, filePath).Scan(
		&file.ID,
		&file.WorkflowID,
		&file.FilePath,
		&file.FileMD5,
		&file.FileSize,
		&file.LastScannedAt,
		&file.CreatedAt,
		&file.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &file, nil
}

// Update updates a file record
func (r *FileRepo) Update(file *models.File) error {
	file.UpdatedAt = time.Now()
	query := `
		UPDATE files
		SET file_md5 = ?, file_size = ?, last_scanned_at = ?, updated_at = ?
		WHERE id = ?
	`
	result, err := r.db.conn.Exec(query,
		file.FileMD5,
		file.FileSize,
		file.LastScannedAt,
		file.UpdatedAt,
		file.ID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("file not found")
	}
	return nil
}

// ListByWorkflow retrieves all files for a workflow
func (r *FileRepo) ListByWorkflow(workflowID string, limit, offset int) ([]*models.File, error) {
	query := `
		SELECT id, workflow_id, file_path, file_md5, file_size, last_scanned_at, created_at, updated_at
		FROM files
		WHERE workflow_id = ?
		ORDER BY file_path
		LIMIT ? OFFSET ?
	`
	rows, err := r.db.conn.Query(query, workflowID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*models.File
	for rows.Next() {
		var file models.File
		err := rows.Scan(
			&file.ID,
			&file.WorkflowID,
			&file.FilePath,
			&file.FileMD5,
			&file.FileSize,
			&file.LastScannedAt,
			&file.CreatedAt,
			&file.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, &file)
	}
	return files, rows.Err()
}

// CountByWorkflow counts files for a workflow
func (r *FileRepo) CountByWorkflow(workflowID string) (int, error) {
	query := `SELECT COUNT(*) FROM files WHERE workflow_id = ?`
	var count int
	err := r.db.conn.QueryRow(query, workflowID).Scan(&count)
	return count, err
}

// DeleteByWorkflow deletes all files for a workflow
func (r *FileRepo) DeleteByWorkflow(workflowID string) error {
	query := `DELETE FROM files WHERE workflow_id = ?`
	_, err := r.db.conn.Exec(query, workflowID)
	return err
}
