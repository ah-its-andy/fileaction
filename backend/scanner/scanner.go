package scanner

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/andi/fileaction/backend/database"
	"github.com/andi/fileaction/backend/models"
	"github.com/andi/fileaction/backend/workflow"
)

// Scanner handles file scanning and task creation
type Scanner struct {
	fileRepo     *database.FileRepo
	taskRepo     *database.TaskRepo
	workflowRepo *database.WorkflowRepo
}

// New creates a new scanner
func New(db *database.DB) *Scanner {
	return &Scanner{
		fileRepo:     database.NewFileRepo(db),
		taskRepo:     database.NewTaskRepo(db),
		workflowRepo: database.NewWorkflowRepo(db),
	}
}

// ScanResult represents the result of a scan operation
type ScanResult struct {
	FilesScanned int
	FilesNew     int
	FilesChanged int
	FilesSkipped int
	TasksCreated int
	Errors       []error
}

// ScanWorkflow scans paths for a workflow and creates tasks
func (s *Scanner) ScanWorkflow(workflowID string) (*ScanResult, error) {
	result := &ScanResult{}

	// Get workflow
	wf, err := s.workflowRepo.GetByID(workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	// Parse workflow
	workflowDef, err := workflow.Parse(wf.YAMLContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse workflow: %w", err)
	}

	// Scan each path
	for _, scanPath := range workflowDef.On.Paths {
		pathResult, err := s.scanPath(workflowID, scanPath, workflowDef)
		if err != nil {
			result.Errors = append(result.Errors, err)
			continue
		}

		result.FilesScanned += pathResult.FilesScanned
		result.FilesNew += pathResult.FilesNew
		result.FilesChanged += pathResult.FilesChanged
		result.FilesSkipped += pathResult.FilesSkipped
		result.TasksCreated += pathResult.TasksCreated
		result.Errors = append(result.Errors, pathResult.Errors...)
	}

	return result, nil
}

// scanPath scans a single path
func (s *Scanner) scanPath(workflowID, scanPath string, workflowDef *workflow.WorkflowDef) (*ScanResult, error) {
	result := &ScanResult{}

	// Resolve absolute path
	absPath, err := filepath.Abs(scanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path %s: %w", scanPath, err)
	}

	// Check if path exists
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("path not found %s: %w", absPath, err)
	}

	// If it's a file, scan just that file
	if !info.IsDir() {
		if err := s.scanFile(workflowID, absPath, workflowDef, result); err != nil {
			result.Errors = append(result.Errors, err)
		}
		return result, nil
	}

	// Walk directory
	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			// Skip subdirectories if not enabled
			if !workflowDef.Options.IncludeSubdirs && path != absPath {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file matches glob pattern
		if !workflow.MatchesFileGlob(path, workflowDef.Options.FileGlob) {
			return nil
		}

		// Scan file
		if err := s.scanFile(workflowID, path, workflowDef, result); err != nil {
			result.Errors = append(result.Errors, err)
		}

		return nil
	}

	if err := filepath.Walk(absPath, walkFn); err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", absPath, err)
	}

	return result, nil
}

// scanFile processes a single file
func (s *Scanner) scanFile(workflowID, filePath string, workflowDef *workflow.WorkflowDef, result *ScanResult) error {
	result.FilesScanned++

	// Calculate MD5
	md5Hash, fileSize, err := calculateMD5(filePath)
	if err != nil {
		return fmt.Errorf("failed to calculate MD5 for %s: %w", filePath, err)
	}

	now := time.Now()

	// Check if file already indexed
	existingFile, err := s.fileRepo.GetByWorkflowAndPath(workflowID, filePath)
	if err != nil {
		return fmt.Errorf("failed to check file index: %w", err)
	}

	fileChanged := false
	var fileID string

	if existingFile == nil {
		// New file
		file := &models.File{
			WorkflowID:    workflowID,
			FilePath:      filePath,
			FileMD5:       md5Hash,
			FileSize:      fileSize,
			LastScannedAt: now,
		}
		if err := s.fileRepo.Create(file); err != nil {
			return fmt.Errorf("failed to create file record: %w", err)
		}
		fileID = file.ID
		result.FilesNew++
		fileChanged = true
		log.Printf("New file detected: %s", filePath)
	} else {
		// Existing file
		fileID = existingFile.ID
		if existingFile.FileMD5 != md5Hash {
			// File changed
			existingFile.FileMD5 = md5Hash
			existingFile.FileSize = fileSize
			existingFile.LastScannedAt = now
			if err := s.fileRepo.Update(existingFile); err != nil {
				return fmt.Errorf("failed to update file record: %w", err)
			}
			result.FilesChanged++
			fileChanged = true
			log.Printf("File changed: %s", filePath)
		} else {
			// File unchanged
			result.FilesSkipped++
			if workflowDef.Options.SkipOnNoChange {
				log.Printf("File unchanged, skipping: %s", filePath)
				return nil
			}
		}
	}

	// Create task if file is new or changed
	if fileChanged || !workflowDef.Options.SkipOnNoChange {
		outputPath := workflow.GenerateOutputPath(filePath, workflowDef.Convert, workflowDef.Options.OutputDirPattern)

		task := &models.Task{
			WorkflowID: workflowID,
			FileID:     fileID,
			InputPath:  filePath,
			OutputPath: outputPath,
			Status:     models.TaskStatusPending,
		}

		if err := s.taskRepo.Create(task); err != nil {
			return fmt.Errorf("failed to create task: %w", err)
		}

		result.TasksCreated++
		log.Printf("Task created for file: %s -> %s", filePath, outputPath)
	}

	return nil
}

// CalculateMD5 calculates the MD5 hash of a file (exported for watcher)
func CalculateMD5(filePath string) (string, int64, error) {
	return calculateMD5(filePath)
}

// calculateMD5 calculates the MD5 hash of a file
func calculateMD5(filePath string) (string, int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	hash := md5.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return "", 0, err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), size, nil
}
