package database

import (
	"database/sql"
	_ "embed"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/yaml.v3"
)

//go:embed default-workflow.yaml
var defaultWorkflowYAML string

// DB wraps the database connection
type DB struct {
	conn *sql.DB
}

// New creates a new database connection and initializes schema
func New(dsn string) (*DB, error) {
	// Use provided DSN or default
	if dsn == "" {
		dsn = "fileaction:fileaction_pass@tcp(localhost:3306)/fileaction?charset=utf8mb4&parseTime=True&loc=Local"
	}

	// Open database connection
	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{conn: conn}

	// Initialize schema
	if err := db.initSchema(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Initialize default workflows
	if err := db.initDefaultWorkflows(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize default workflows: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// GetConn returns the underlying database connection
func (db *DB) GetConn() *sql.DB {
	return db.conn
}

// initSchema creates all necessary tables
func (db *DB) initSchema() error {
	schemas := []string{
		`CREATE TABLE IF NOT EXISTS workflows (
			id VARCHAR(36) PRIMARY KEY,
			name VARCHAR(255) NOT NULL UNIQUE,
			description TEXT,
			yaml_content TEXT NOT NULL,
			enabled BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_name (name),
			INDEX idx_enabled (enabled)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS files (
			id VARCHAR(36) PRIMARY KEY,
			workflow_id VARCHAR(36) NOT NULL,
			file_path VARCHAR(1024) NOT NULL,
			file_md5 VARCHAR(32) NOT NULL,
			file_size BIGINT NOT NULL,
			last_scanned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE KEY unique_workflow_file (workflow_id, file_path(255)),
			INDEX idx_workflow_id (workflow_id),
			INDEX idx_file_md5 (file_md5)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS tasks (
			id VARCHAR(36) PRIMARY KEY,
			workflow_id VARCHAR(36) NOT NULL,
			file_id VARCHAR(36) NOT NULL,
			input_path VARCHAR(1024) NOT NULL,
			output_path VARCHAR(1024),
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			log_text TEXT,
			error_message TEXT,
			started_at TIMESTAMP NULL,
			completed_at TIMESTAMP NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_workflow_id (workflow_id),
			INDEX idx_file_id (file_id),
			INDEX idx_status (status),
			INDEX idx_created_at (created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS task_steps (
			id VARCHAR(36) PRIMARY KEY,
			task_id VARCHAR(36) NOT NULL,
			name VARCHAR(255) NOT NULL,
			command TEXT NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			exit_code INT,
			stdout TEXT,
			stderr TEXT,
			started_at TIMESTAMP NULL,
			completed_at TIMESTAMP NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_task_id (task_id),
			INDEX idx_status (status)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}

	for _, schema := range schemas {
		if _, err := db.conn.Exec(schema); err != nil {
			return err
		}
	}

	return nil
}

// initDefaultWorkflows creates default workflows if they don't exist
func (db *DB) initDefaultWorkflows() error {
	// Parse YAML to get workflow metadata
	var workflowData struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}

	if err := yaml.Unmarshal([]byte(defaultWorkflowYAML), &workflowData); err != nil {
		return fmt.Errorf("failed to parse default workflow: %w", err)
	}

	// Check if workflow already exists
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM workflows WHERE name = ?", workflowData.Name).Scan(&count)
	if err != nil {
		return err
	}

	// If workflow already exists, skip initialization
	if count > 0 {
		return nil
	}

	// Create default workflow
	_, err = db.conn.Exec(
		"INSERT INTO workflows (id, name, description, yaml_content, enabled) VALUES (?, ?, ?, ?, ?)",
		"default-jpeg-to-heic",
		workflowData.Name,
		workflowData.Description,
		defaultWorkflowYAML,
		true,
	)

	return err
}
