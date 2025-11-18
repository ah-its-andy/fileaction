package workflow

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// WorkflowDef represents a parsed workflow definition
type WorkflowDef struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	On          OnConfig          `yaml:"on"`
	Convert     ConvertConfig     `yaml:"convert"`
	Steps       []Step            `yaml:"steps"`
	Options     Options           `yaml:"options"`
	Env         map[string]string `yaml:"env"`
}

// OnConfig specifies trigger conditions
type OnConfig struct {
	Paths []string `yaml:"paths"`
}

// ConvertConfig specifies conversion settings
type ConvertConfig struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

// Step represents a workflow step
type Step struct {
	Name string            `yaml:"name"`
	Run  string            `yaml:"run"`
	Env  map[string]string `yaml:"env"`
}

// Options represents workflow execution options
type Options struct {
	Concurrency      int      `yaml:"concurrency"`
	IncludeSubdirs   bool     `yaml:"include_subdirs"`
	FileGlob         string   `yaml:"file_glob"`
	SkipOnNoChange   bool     `yaml:"skip_on_nochange"`
	OutputDirPattern string   `yaml:"output_dir_pattern"`
	Ignore           []string `yaml:"ignore"`
}

// Variables available for substitution
type Variables struct {
	InputPath  string
	OutputPath string
	FileName   string
	FileDir    string
	FileBase   string
	FileExt    string
}

// Parse parses a YAML workflow definition
func Parse(yamlContent string) (*WorkflowDef, error) {
	var workflow WorkflowDef
	if err := yaml.Unmarshal([]byte(yamlContent), &workflow); err != nil {
		return nil, fmt.Errorf("failed to parse workflow YAML: %w", err)
	}

	// Set defaults
	if workflow.Options.Concurrency == 0 {
		workflow.Options.Concurrency = 4
	}
	if workflow.Options.FileGlob == "" {
		workflow.Options.FileGlob = "*"
	}
	workflow.Options.SkipOnNoChange = true // Default to true

	// Validate required fields
	if workflow.Name == "" {
		return nil, fmt.Errorf("workflow name is required")
	}
	if len(workflow.On.Paths) == 0 {
		return nil, fmt.Errorf("at least one path must be specified in 'on.paths'")
	}
	if len(workflow.Steps) == 0 {
		return nil, fmt.Errorf("at least one step is required")
	}

	return &workflow, nil
}

// SubstituteVariables replaces variables in a string
func SubstituteVariables(template string, vars Variables) string {
	result := template

	replacements := map[string]string{
		"${{ input_path }}":  vars.InputPath,
		"${{ output_path }}": vars.OutputPath,
		"${{ file_name }}":   vars.FileName,
		"${{ file_dir }}":    vars.FileDir,
		"${{ file_base }}":   vars.FileBase,
		"${{ file_ext }}":    vars.FileExt,
	}

	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}

// GenerateOutputPath generates the output path based on conversion config
func GenerateOutputPath(inputPath string, convertConfig ConvertConfig, outputDirPattern string) string {
	dir := filepath.Dir(inputPath)
	base := filepath.Base(inputPath)
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)

	// If output directory pattern is specified, use it
	if outputDirPattern != "" {
		// Support relative patterns like "../heic"
		if strings.HasPrefix(outputDirPattern, "..") || strings.HasPrefix(outputDirPattern, ".") {
			dir = filepath.Join(dir, outputDirPattern)
		} else {
			dir = outputDirPattern
		}
	}

	// Replace extension based on conversion target
	newExt := "." + convertConfig.To
	if convertConfig.To == "" {
		newExt = ext
	}

	return filepath.Join(dir, nameWithoutExt+newExt)
}

// MatchesFileGlob checks if a file matches the glob pattern
// Supports multiple patterns separated by comma or pipe, e.g., "*.jpg,*.jpeg" or "*.jpg|*.jpeg"
func MatchesFileGlob(filePath, globPattern string) bool {
	fileName := filepath.Base(filePath)

	// Split pattern by comma or pipe to support multiple patterns
	patterns := strings.FieldsFunc(globPattern, func(r rune) bool {
		return r == ',' || r == '|'
	})

	// If no separator found, treat as single pattern
	if len(patterns) == 0 {
		patterns = []string{globPattern}
	}

	// Check if file matches any of the patterns
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		matched, err := filepath.Match(pattern, fileName)
		if err != nil {
			continue
		}
		if matched {
			return true
		}
	}

	return false
}

// MatchesIgnorePattern checks if a file path matches any of the ignore patterns
// Supports:
// - Glob patterns for filenames (e.g., "*.tmp", "*.log")
// - Directory names (e.g., ".git", "node_modules")
// - File names (e.g., ".DS_Store", "Thumbs.db")
// - Path patterns (e.g., "**/temp/**", "**/.git/**")
func MatchesIgnorePattern(filePath string, ignorePatterns []string) bool {
	if len(ignorePatterns) == 0 {
		return false
	}

	// Get filename and directory components
	fileName := filepath.Base(filePath)
	dirPath := filepath.Dir(filePath)

	for _, pattern := range ignorePatterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}

		// Check if it's a path pattern with wildcards
		if strings.Contains(pattern, "**") || strings.Contains(pattern, string(filepath.Separator)) {
			// Handle path patterns like "**/temp/**" or "temp/**"
			matched, err := filepath.Match(pattern, filePath)
			if err == nil && matched {
				return true
			}

			// Also try matching against the full path with forward slashes (for cross-platform compatibility)
			normalizedPath := filepath.ToSlash(filePath)
			normalizedPattern := filepath.ToSlash(pattern)
			matched, err = filepath.Match(normalizedPattern, normalizedPath)
			if err == nil && matched {
				return true
			}

			// Check if any directory in the path matches the pattern
			if strings.Contains(pattern, "**") {
				patternParts := strings.Split(strings.Trim(pattern, "**/"), "/")
				for _, part := range patternParts {
					if part == "" {
						continue
					}
					if strings.Contains(dirPath, part) || strings.Contains(filePath, part) {
						return true
					}
				}
			}
		} else {
			// Simple pattern - check against filename
			matched, err := filepath.Match(pattern, fileName)
			if err == nil && matched {
				return true
			}

			// Check if pattern matches any directory component
			pathParts := strings.Split(filePath, string(filepath.Separator))
			for _, part := range pathParts {
				if part == pattern {
					return true
				}
			}
		}
	}

	return false
}

// GetVariables extracts variables from a file path
func GetVariables(inputPath, outputPath string) Variables {
	fileName := filepath.Base(inputPath)
	fileDir := filepath.Dir(inputPath)
	fileExt := filepath.Ext(fileName)
	fileBase := strings.TrimSuffix(fileName, fileExt)

	return Variables{
		InputPath:  inputPath,
		OutputPath: outputPath,
		FileName:   fileName,
		FileDir:    fileDir,
		FileBase:   fileBase,
		FileExt:    fileExt,
	}
}

// Validate validates a workflow definition
func Validate(workflow *WorkflowDef) error {
	if workflow.Name == "" {
		return fmt.Errorf("workflow name is required")
	}

	// Validate name format (alphanumeric, hyphens, underscores)
	validName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validName.MatchString(workflow.Name) {
		return fmt.Errorf("workflow name must contain only alphanumeric characters, hyphens, and underscores")
	}

	if len(workflow.On.Paths) == 0 {
		return fmt.Errorf("at least one path must be specified")
	}

	if len(workflow.Steps) == 0 {
		return fmt.Errorf("at least one step is required")
	}

	for i, step := range workflow.Steps {
		if step.Name == "" {
			return fmt.Errorf("step %d: name is required", i+1)
		}
		if step.Run == "" {
			return fmt.Errorf("step %d (%s): run command is required", i+1, step.Name)
		}
	}

	if workflow.Options.Concurrency < 1 {
		return fmt.Errorf("concurrency must be at least 1")
	}

	return nil
}
