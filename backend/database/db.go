package database

import (
	_ "embed"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

//go:embed default-workflow.yaml
var defaultWorkflowYAML string

//go:embed default-plugins/jpeg-to-heic-converter.yaml
var defaultPluginJpegToHeic string

// DB wraps the GORM database connection
type DB struct {
	conn   *gorm.DB
	dbType string // "mysql" or "sqlite"
}

// New creates a new database connection and initializes schema
func New(dsn string) (*DB, error) {
	var gormDB *gorm.DB
	var dbType string
	var err error

	// Configure GORM
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		NowFunc: func() time.Time {
			return time.Now()
		},
	}

	// Detect database type and open connection
	if dsn == "" || strings.HasSuffix(dsn, ".db") || strings.Contains(dsn, "file:") {
		// SQLite with pure Go driver (modernc.org/sqlite)
		if dsn == "" {
			dsn = "./data/fileaction.db"
		}
		dbType = "sqlite"
		// Use DriverName option to specify the pure Go SQLite driver
		gormDB, err = gorm.Open(sqlite.Dialector{
			DriverName: "sqlite",
			DSN:        dsn,
		}, config)
	} else {
		// MySQL
		dbType = "mysql"
		gormDB, err = gorm.Open(mysql.Open(dsn), config)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Get underlying SQL database for connection pool configuration
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying database: %w", err)
	}

	// Configure connection pool
	if dbType == "sqlite" {
		sqlDB.SetMaxOpenConns(1) // SQLite works best with single writer
		sqlDB.SetMaxIdleConns(1)
	} else {
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetMaxIdleConns(10)
	}

	db := &DB{
		conn:   gormDB,
		dbType: dbType,
	}

	// Initialize schema
	if err := db.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Initialize default workflows
	if err := db.initDefaultWorkflows(); err != nil {
		return nil, fmt.Errorf("failed to initialize default workflows: %w", err)
	}

	// Initialize default plugins
	if err := db.initDefaultPlugins(); err != nil {
		return nil, fmt.Errorf("failed to initialize default plugins: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	sqlDB, err := db.conn.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// GetConn returns the underlying GORM database connection
func (db *DB) GetConn() *gorm.DB {
	return db.conn
}

// initSchema creates all necessary tables using GORM AutoMigrate
func (db *DB) initSchema() error {
	// AutoMigrate will create tables with appropriate types for each database
	return db.conn.AutoMigrate(
		&WorkflowModel{},
		&FileModel{},
		&TaskModel{},
		&TaskStepModel{},
		&PluginModel{},
		&PluginVersionModel{},
	)
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
	var count int64
	if err := db.conn.Model(&WorkflowModel{}).Where("name = ?", workflowData.Name).Count(&count).Error; err != nil {
		return err
	}

	// If workflow already exists, skip initialization
	if count > 0 {
		return nil
	}

	// Create default workflow
	workflow := &WorkflowModel{
		ID:          "default-jpeg-to-heic",
		Name:        workflowData.Name,
		Description: workflowData.Description,
		YAMLContent: defaultWorkflowYAML,
		Enabled:     false, // Default workflow starts disabled
	}

	return db.conn.Create(workflow).Error
}

// initDefaultPlugins creates default plugins if they don't exist
func (db *DB) initDefaultPlugins() error {
	pluginRepo := NewPluginRepo(db)

	// Define default plugins to install
	defaultPlugins := []struct {
		yamlContent string
		name        string
	}{
		{
			yamlContent: defaultPluginJpegToHeic,
			name:        "jpeg-to-heic-converter",
		},
	}

	for _, dp := range defaultPlugins {
		// Check if plugin already exists
		var count int64
		if err := db.conn.Model(&PluginModel{}).Where("name = ?", dp.name).Count(&count).Error; err != nil {
			return err
		}

		// If plugin already exists, skip
		if count > 0 {
			continue
		}

		// Parse YAML to get plugin metadata
		var pluginData struct {
			Name        string `yaml:"name"`
			Description string `yaml:"description"`
		}

		if err := yaml.Unmarshal([]byte(dp.yamlContent), &pluginData); err != nil {
			return fmt.Errorf("failed to parse default plugin %s: %w", dp.name, err)
		}

		// Create plugin
		_, _, err := pluginRepo.CreatePlugin(
			pluginData.Name,
			pluginData.Description,
			dp.yamlContent,
			"system",
		)
		if err != nil {
			return fmt.Errorf("failed to create default plugin %s: %w", dp.name, err)
		}
	}

	return nil
}

// GORM Models
type WorkflowModel struct {
	ID          string    `gorm:"primaryKey;type:varchar(36)"`
	Name        string    `gorm:"uniqueIndex;type:varchar(255);not null"`
	Description string    `gorm:"type:text"`
	YAMLContent string    `gorm:"type:text;not null"`
	Enabled     bool      `gorm:"default:true;index"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

func (WorkflowModel) TableName() string {
	return "workflows"
}

type FileModel struct {
	ID            string    `gorm:"primaryKey;type:varchar(36)"`
	WorkflowID    string    `gorm:"type:varchar(36);not null;index"`
	FilePath      string    `gorm:"type:varchar(1024);not null"`
	FileMD5       string    `gorm:"type:varchar(32);not null;index"`
	FileSize      int64     `gorm:"not null"`
	LastScannedAt time.Time `gorm:"autoCreateTime"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
}

func (FileModel) TableName() string {
	return "files"
}

type TaskModel struct {
	ID           string     `gorm:"primaryKey;type:varchar(36)"`
	WorkflowID   string     `gorm:"type:varchar(36);not null;index"`
	FileID       string     `gorm:"type:varchar(36);not null;index"`
	InputPath    string     `gorm:"type:varchar(1024);not null"`
	OutputPath   string     `gorm:"type:varchar(1024)"`
	Status       string     `gorm:"type:varchar(20);not null;default:'pending';index"`
	LogText      string     `gorm:"type:text"`
	ErrorMessage string     `gorm:"type:text"`
	StartedAt    *time.Time `gorm:"index"`
	CompletedAt  *time.Time
	CreatedAt    time.Time `gorm:"autoCreateTime;index"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}

func (TaskModel) TableName() string {
	return "tasks"
}

type TaskStepModel struct {
	ID          string `gorm:"primaryKey;type:varchar(36)"`
	TaskID      string `gorm:"type:varchar(36);not null;index"`
	Name        string `gorm:"type:varchar(255);not null"`
	Command     string `gorm:"type:text;not null"`
	Status      string `gorm:"type:varchar(20);not null;default:'pending';index"`
	ExitCode    *int   `gorm:"type:int"`
	Stdout      string `gorm:"type:text"`
	Stderr      string `gorm:"type:text"`
	StartedAt   *time.Time
	CompletedAt *time.Time
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

func (TaskStepModel) TableName() string {
	return "task_steps"
}
