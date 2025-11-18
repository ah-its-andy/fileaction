package workflow

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	yamlContent := `
name: test-workflow
description: Test workflow
on:
  paths:
    - ./test
convert:
  from: jpg
  to: png
steps:
  - name: convert
    run: convert input output
options:
  concurrency: 2
  include_subdirs: true
  file_glob: "*.jpg"
`

	workflow, err := Parse(yamlContent)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	if workflow.Name != "test-workflow" {
		t.Errorf("Expected name 'test-workflow', got '%s'", workflow.Name)
	}

	if len(workflow.On.Paths) != 1 {
		t.Errorf("Expected 1 path, got %d", len(workflow.On.Paths))
	}

	if workflow.Convert.From != "jpg" {
		t.Errorf("Expected from 'jpg', got '%s'", workflow.Convert.From)
	}

	if workflow.Convert.To != "png" {
		t.Errorf("Expected to 'png', got '%s'", workflow.Convert.To)
	}

	if len(workflow.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(workflow.Steps))
	}

	if workflow.Options.Concurrency != 2 {
		t.Errorf("Expected concurrency 2, got %d", workflow.Options.Concurrency)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		workflow    *WorkflowDef
		shouldError bool
	}{
		{
			name: "valid workflow",
			workflow: &WorkflowDef{
				Name: "test",
				On: OnConfig{
					Paths: []string{"./test"},
				},
				Steps: []Step{
					{Name: "step1", Run: "echo test"},
				},
				Options: Options{Concurrency: 1},
			},
			shouldError: false,
		},
		{
			name: "missing name",
			workflow: &WorkflowDef{
				On: OnConfig{
					Paths: []string{"./test"},
				},
				Steps: []Step{
					{Name: "step1", Run: "echo test"},
				},
			},
			shouldError: true,
		},
		{
			name: "invalid name",
			workflow: &WorkflowDef{
				Name: "test workflow!",
				On: OnConfig{
					Paths: []string{"./test"},
				},
				Steps: []Step{
					{Name: "step1", Run: "echo test"},
				},
			},
			shouldError: true,
		},
		{
			name: "no paths",
			workflow: &WorkflowDef{
				Name:  "test",
				Steps: []Step{{Name: "step1", Run: "echo test"}},
			},
			shouldError: true,
		},
		{
			name: "no steps",
			workflow: &WorkflowDef{
				Name: "test",
				On: OnConfig{
					Paths: []string{"./test"},
				},
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.workflow)
			if tt.shouldError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestSubstituteVariables(t *testing.T) {
	vars := Variables{
		InputPath:  "/path/to/input.jpg",
		OutputPath: "/path/to/output.png",
		FileName:   "input.jpg",
		FileDir:    "/path/to",
		FileBase:   "input",
		FileExt:    ".jpg",
	}

	tests := []struct {
		template string
		expected string
	}{
		{
			template: "convert ${{ input_path }} ${{ output_path }}",
			expected: "convert /path/to/input.jpg /path/to/output.png",
		},
		{
			template: "File: ${{ file_name }}",
			expected: "File: input.jpg",
		},
		{
			template: "Base: ${{ file_base }}, Ext: ${{ file_ext }}",
			expected: "Base: input, Ext: .jpg",
		},
		{
			template: "Dir: ${{ file_dir }}",
			expected: "Dir: /path/to",
		},
	}

	for _, tt := range tests {
		t.Run(tt.template, func(t *testing.T) {
			result := SubstituteVariables(tt.template, vars)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGenerateOutputPath(t *testing.T) {
	tests := []struct {
		name             string
		inputPath        string
		convertConfig    ConvertConfig
		outputDirPattern string
		expectedContains string
	}{
		{
			name:      "basic conversion",
			inputPath: "/input/test.jpg",
			convertConfig: ConvertConfig{
				From: "jpg",
				To:   "png",
			},
			outputDirPattern: "",
			expectedContains: "test.png",
		},
		{
			name:      "with output directory",
			inputPath: "/input/test.jpg",
			convertConfig: ConvertConfig{
				From: "jpg",
				To:   "png",
			},
			outputDirPattern: "/output",
			expectedContains: "/output/test.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateOutputPath(tt.inputPath, tt.convertConfig, tt.outputDirPattern)
			if !strings.Contains(result, tt.expectedContains) {
				t.Errorf("Expected output to contain '%s', got '%s'", tt.expectedContains, result)
			}
		})
	}
}

func TestMatchesFileGlob(t *testing.T) {
	tests := []struct {
		filePath string
		pattern  string
		expected bool
	}{
		{"/path/to/file.jpg", "*.jpg", true},
		{"/path/to/file.png", "*.jpg", false},
		{"/path/to/file.jpeg", "*.jp*g", true},
		{"/path/to/test.txt", "test.*", true},
		{"/path/to/other.txt", "test.*", false},
	}

	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			result := MatchesFileGlob(tt.filePath, tt.pattern)
			if result != tt.expected {
				t.Errorf("Expected %v for pattern '%s' on file '%s', got %v",
					tt.expected, tt.pattern, tt.filePath, result)
			}
		})
	}
}

func TestMatchesIgnorePattern(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		patterns []string
		expected bool
	}{
		{
			name:     "match .DS_Store",
			filePath: "/path/to/.DS_Store",
			patterns: []string{".DS_Store"},
			expected: true,
		},
		{
			name:     "match Thumbs.db",
			filePath: "/path/to/Thumbs.db",
			patterns: []string{"Thumbs.db"},
			expected: true,
		},
		{
			name:     "match *.tmp files",
			filePath: "/path/to/file.tmp",
			patterns: []string{"*.tmp"},
			expected: true,
		},
		{
			name:     "match .git directory",
			filePath: "/path/to/.git/config",
			patterns: []string{".git"},
			expected: true,
		},
		{
			name:     "match **/.git/** pattern",
			filePath: "/path/to/.git/objects/abc",
			patterns: []string{"**/.git/**"},
			expected: true,
		},
		{
			name:     "match **/temp/** pattern",
			filePath: "/path/to/temp/file.txt",
			patterns: []string{"**/temp/**"},
			expected: true,
		},
		{
			name:     "no match",
			filePath: "/path/to/file.jpg",
			patterns: []string{".DS_Store", "*.tmp"},
			expected: false,
		},
		{
			name:     "multiple patterns - first matches",
			filePath: "/path/to/.DS_Store",
			patterns: []string{".DS_Store", "Thumbs.db", "*.tmp"},
			expected: true,
		},
		{
			name:     "multiple patterns - last matches",
			filePath: "/path/to/file.tmp",
			patterns: []string{".DS_Store", "Thumbs.db", "*.tmp"},
			expected: true,
		},
		{
			name:     "empty patterns",
			filePath: "/path/to/file.jpg",
			patterns: []string{},
			expected: false,
		},
		{
			name:     "match node_modules directory",
			filePath: "/path/to/node_modules/package/index.js",
			patterns: []string{"node_modules"},
			expected: true,
		},
		{
			name:     "match drafts in any subdirectory",
			filePath: "/images/drafts/photo.jpg",
			patterns: []string{"**/drafts/**"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchesIgnorePattern(tt.filePath, tt.patterns)
			if result != tt.expected {
				t.Errorf("Expected %v for patterns %v on file '%s', got %v",
					tt.expected, tt.patterns, tt.filePath, result)
			}
		})
	}
}
