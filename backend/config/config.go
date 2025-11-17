package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server struct {
		Host         string        `yaml:"host"`
		Port         int           `yaml:"port"`
		ReadTimeout  time.Duration `yaml:"read_timeout"`
		WriteTimeout time.Duration `yaml:"write_timeout"`
	} `yaml:"server"`

	Database struct {
		Path string `yaml:"path"`
	} `yaml:"database"`

	Logging struct {
		Dir    string `yaml:"dir"`
		AppLog string `yaml:"app_log"`
		Level  string `yaml:"level"`
	} `yaml:"logging"`

	Execution struct {
		DefaultConcurrency int           `yaml:"default_concurrency"`
		MaxConcurrency     int           `yaml:"max_concurrency"`
		TaskTimeout        time.Duration `yaml:"task_timeout"`
		StepTimeout        time.Duration `yaml:"step_timeout"`
	} `yaml:"execution"`

	Polling struct {
		Interval time.Duration `yaml:"interval"`
	} `yaml:"polling"`

	Scheduler struct {
		Enabled bool   `yaml:"enabled"`
		Cron    string `yaml:"cron"`
	} `yaml:"scheduler"`
}

// Load loads configuration from a YAML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Set defaults if not specified
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Database.Path == "" {
		cfg.Database.Path = "./data/fileaction.db"
	}
	if cfg.Logging.Dir == "" {
		cfg.Logging.Dir = "./data/logs"
	}
	if cfg.Logging.AppLog == "" {
		cfg.Logging.AppLog = "./data/logs/app.log"
	}
	if cfg.Execution.DefaultConcurrency == 0 {
		cfg.Execution.DefaultConcurrency = 4
	}
	if cfg.Execution.MaxConcurrency == 0 {
		cfg.Execution.MaxConcurrency = 16
	}
	if cfg.Execution.TaskTimeout == 0 {
		cfg.Execution.TaskTimeout = 3600 * time.Second
	}
	if cfg.Execution.StepTimeout == 0 {
		cfg.Execution.StepTimeout = 1800 * time.Second
	}
	if cfg.Polling.Interval == 0 {
		cfg.Polling.Interval = 2 * time.Second
	}

	return &cfg, nil
}

// LoadFromEnv loads configuration with environment variable overrides
func LoadFromEnv(path string) (*Config, error) {
	cfg, err := Load(path)
	if err != nil {
		return nil, err
	}

	// Override with environment variables if set
	if dbPath := os.Getenv("DB_PATH"); dbPath != "" {
		cfg.Database.Path = dbPath
	}
	if logDir := os.Getenv("LOG_DIR"); logDir != "" {
		cfg.Logging.Dir = logDir
		cfg.Logging.AppLog = logDir + "/app.log"
	}

	return cfg, nil
}
