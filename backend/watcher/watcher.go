package watcher

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/andi/fileaction/backend/database"
	"github.com/andi/fileaction/backend/models"
	"github.com/andi/fileaction/backend/workflow"
	"github.com/fsnotify/fsnotify"
)

// ScanResult represents the result of a scan operation
type ScanResult struct {
	FilesScanned int
	FilesNew     int
	FilesChanged int
	FilesSkipped int
	TasksCreated int
	Errors       []error
}

// Watcher monitors file system changes and triggers workflows
type Watcher struct {
	db           *database.DB
	fileRepo     *database.FileRepo
	taskRepo     *database.TaskRepo
	workflowRepo *database.WorkflowRepo
	watcher      *fsnotify.Watcher
	stopChan     chan struct{}
	wg           sync.WaitGroup
	mu           sync.Mutex
	stopped      bool

	// Map of workflow ID to watched paths
	watchedPaths map[string][]string

	// Debounce map to avoid processing same file multiple times
	debounceMap map[string]*debounceEntry
	debounceMu  sync.Mutex
}

type debounceEntry struct {
	timer      *time.Timer
	workflowID string
	path       string
}

// New creates a new file watcher
func New(db *database.DB) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		db:           db,
		fileRepo:     database.NewFileRepo(db),
		taskRepo:     database.NewTaskRepo(db),
		workflowRepo: database.NewWorkflowRepo(db),
		watcher:      fsWatcher,
		stopChan:     make(chan struct{}),
		watchedPaths: make(map[string][]string),
		debounceMap:  make(map[string]*debounceEntry),
	}, nil
}

// Start starts the file watcher
func (w *Watcher) Start() error {
	// Get all enabled workflows
	workflows, err := w.workflowRepo.List()
	if err != nil {
		return err
	}

	// Scan and watch each enabled workflow
	for _, wf := range workflows {
		if !wf.Enabled {
			continue
		}

		// Perform initial scan
		log.Printf("Performing initial scan for workflow: %s", wf.Name)
		result, err := w.scanWorkflow(wf.ID)
		if err != nil {
			log.Printf("Warning: Failed to scan workflow %s: %v", wf.Name, err)
		} else {
			log.Printf("Scan completed for workflow %s: scanned=%d, new=%d, changed=%d, skipped=%d, tasks=%d",
				wf.Name, result.FilesScanned, result.FilesNew, result.FilesChanged, result.FilesSkipped, result.TasksCreated)
		}

		// Add file system watches
		if err := w.addWorkflowWatch(wf); err != nil {
			log.Printf("Warning: Failed to add watch for workflow %s: %v", wf.Name, err)
		}
	}

	// Start event processing
	w.wg.Add(1)
	go w.processEvents()

	log.Printf("File watcher started, monitoring %d workflow(s)", len(w.watchedPaths))
	return nil
}

// Stop stops the file watcher
func (w *Watcher) Stop() {
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return
	}
	w.stopped = true
	w.mu.Unlock()

	log.Println("Stopping file watcher...")
	close(w.stopChan)
	w.watcher.Close()
	w.wg.Wait()
	log.Println("File watcher stopped")
}

// addWorkflowWatch adds file system watches for a workflow
func (w *Watcher) addWorkflowWatch(wf *models.Workflow) error {
	workflowDef, err := workflow.Parse(wf.YAMLContent)
	if err != nil {
		return err
	}

	var paths []string
	for _, scanPath := range workflowDef.On.Paths {
		absPath, err := filepath.Abs(scanPath)
		if err != nil {
			log.Printf("Warning: Failed to resolve path %s: %v", scanPath, err)
			continue
		}

		// Add the path itself
		if err := w.watcher.Add(absPath); err != nil {
			log.Printf("Warning: Failed to watch path %s: %v", absPath, err)
			continue
		}
		paths = append(paths, absPath)
		log.Printf("Watching path: %s (workflow: %s)", absPath, wf.Name)

		// If include_subdirs is enabled, walk and add all subdirectories
		if workflowDef.Options.IncludeSubdirs {
			filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if info.IsDir() && path != absPath {
					if err := w.watcher.Add(path); err != nil {
						log.Printf("Warning: Failed to watch subdirectory %s: %v", path, err)
					} else {
						paths = append(paths, path)
					}
				}
				return nil
			})
		}
	}

	w.watchedPaths[wf.ID] = paths
	return nil
}

// processEvents processes file system events
func (w *Watcher) processEvents() {
	defer w.wg.Done()

	for {
		select {
		case <-w.stopChan:
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Only handle Create and Write events
			if event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Write == fsnotify.Write {
				w.handleFileEvent(event.Name)
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

// handleFileEvent handles a file system event with debouncing
func (w *Watcher) handleFileEvent(path string) {
	// Find which workflow(s) this path belongs to
	workflows := w.findWorkflowsForPath(path)
	if len(workflows) == 0 {
		return
	}

	// Debounce: wait a bit to see if more events come for the same file
	w.debounceMu.Lock()
	defer w.debounceMu.Unlock()

	for _, wf := range workflows {
		key := wf.ID + ":" + path

		if entry, exists := w.debounceMap[key]; exists {
			// Reset the timer
			entry.timer.Stop()
			entry.timer = time.AfterFunc(500*time.Millisecond, func() {
				w.processFile(wf, path)
				w.debounceMu.Lock()
				delete(w.debounceMap, key)
				w.debounceMu.Unlock()
			})
		} else {
			// Create new debounce timer
			timer := time.AfterFunc(500*time.Millisecond, func() {
				w.processFile(wf, path)
				w.debounceMu.Lock()
				delete(w.debounceMap, key)
				w.debounceMu.Unlock()
			})

			w.debounceMap[key] = &debounceEntry{
				timer:      timer,
				workflowID: wf.ID,
				path:       path,
			}
		}
	}
}

// findWorkflowsForPath finds workflows that should process this path
func (w *Watcher) findWorkflowsForPath(path string) []*models.Workflow {
	var result []*models.Workflow

	for workflowID, paths := range w.watchedPaths {
		for _, watchedPath := range paths {
			// Check if the file is under a watched path
			if isPathUnder(path, watchedPath) {
				wf, err := w.workflowRepo.GetByID(workflowID)
				if err != nil {
					log.Printf("Error getting workflow %s: %v", workflowID, err)
					continue
				}

				// Check if file matches the workflow's file glob
				workflowDef, err := workflow.Parse(wf.YAMLContent)
				if err != nil {
					log.Printf("Error parsing workflow %s: %v", wf.Name, err)
					continue
				}

				if workflow.MatchesFileGlob(path, workflowDef.Options.FileGlob) {
					result = append(result, wf)
				}
				break
			}
		}
	}

	return result
}

// processFile processes a single file for a workflow
func (w *Watcher) processFile(wf *models.Workflow, filePath string) {
	log.Printf("Processing file change: %s (workflow: %s)", filePath, wf.Name)

	// Parse workflow definition
	workflowDef, err := workflow.Parse(wf.YAMLContent)
	if err != nil {
		log.Printf("Error parsing workflow %s: %v", wf.Name, err)
		return
	}

	// Check if file matches glob pattern
	if !workflow.MatchesFileGlob(filePath, workflowDef.Options.FileGlob) {
		log.Printf("File %s does not match glob pattern %s, skipping", filePath, workflowDef.Options.FileGlob)
		return
	}

	// Calculate file MD5
	md5Hash, fileSize, err := w.calculateMD5(filePath)
	if err != nil {
		log.Printf("Error calculating MD5 for %s: %v", filePath, err)
		return
	}

	now := time.Now()
	existingFile, err := w.fileRepo.GetByWorkflowAndPath(wf.ID, filePath)
	if err != nil {
		log.Printf("Error checking file index: %v", err)
		return
	}

	fileChanged := false
	var fileID string

	if existingFile == nil {
		// New file
		file := &models.File{
			WorkflowID:    wf.ID,
			FilePath:      filePath,
			FileMD5:       md5Hash,
			FileSize:      fileSize,
			LastScannedAt: now,
		}
		if err := w.fileRepo.Create(file); err != nil {
			log.Printf("Error creating file record: %v", err)
			return
		}
		fileID = file.ID
		fileChanged = true
		log.Printf("New file detected: %s", filePath)
	} else {
		fileID = existingFile.ID
		if existingFile.FileMD5 != md5Hash {
			existingFile.FileMD5 = md5Hash
			existingFile.FileSize = fileSize
			existingFile.LastScannedAt = now
			if err := w.fileRepo.Update(existingFile); err != nil {
				log.Printf("Error updating file record: %v", err)
				return
			}
			fileChanged = true
			log.Printf("File changed: %s", filePath)
		} else if workflowDef.Options.SkipOnNoChange {
			log.Printf("File unchanged, skipping: %s", filePath)
			return
		}
	}

	// Create task if file is new or changed
	if fileChanged || !workflowDef.Options.SkipOnNoChange {
		outputPath := workflow.GenerateOutputPath(filePath, workflowDef.Convert, workflowDef.Options.OutputDirPattern)

		task := &models.Task{
			WorkflowID: wf.ID,
			FileID:     fileID,
			InputPath:  filePath,
			OutputPath: outputPath,
			Status:     models.TaskStatusPending,
		}

		if err := w.taskRepo.Create(task); err != nil {
			log.Printf("Error creating task: %v", err)
			return
		}

		log.Printf("Task created for file: %s -> %s", filePath, outputPath)
	}
}

// isPathUnder checks if path is under basePath
func isPathUnder(path, basePath string) bool {
	rel, err := filepath.Rel(basePath, path)
	if err != nil {
		return false
	}
	// If rel starts with "..", it's not under basePath
	return len(rel) > 0 && rel[0] != '.' && (len(rel) < 2 || rel[:2] != "..")
}

// ReloadWorkflows reloads all workflow watches
func (w *Watcher) ReloadWorkflows() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Remove all existing watches
	for _, paths := range w.watchedPaths {
		for _, path := range paths {
			w.watcher.Remove(path)
		}
	}
	w.watchedPaths = make(map[string][]string)

	// Reload workflows
	workflows, err := w.workflowRepo.List()
	if err != nil {
		return err
	}

	for _, wf := range workflows {
		if !wf.Enabled {
			continue
		}

		if err := w.addWorkflowWatch(wf); err != nil {
			log.Printf("Warning: Failed to add watch for workflow %s: %v", wf.Name, err)
		}
	}

	log.Printf("Workflows reloaded, monitoring %d workflow(s)", len(w.watchedPaths))
	return nil
}

// scanWorkflow scans all paths for a workflow and creates tasks
func (w *Watcher) scanWorkflow(workflowID string) (*ScanResult, error) {
	result := &ScanResult{}

	// Get workflow
	wf, err := w.workflowRepo.GetByID(workflowID)
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
		pathResult, err := w.scanPath(workflowID, scanPath, workflowDef)
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
func (w *Watcher) scanPath(workflowID, scanPath string, workflowDef *workflow.WorkflowDef) (*ScanResult, error) {
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
		if err := w.scanFile(workflowID, absPath, workflowDef, result); err != nil {
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
		if err := w.scanFile(workflowID, path, workflowDef, result); err != nil {
			result.Errors = append(result.Errors, err)
		}

		return nil
	}

	if err := filepath.Walk(absPath, walkFn); err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", absPath, err)
	}

	return result, nil
}

// scanFile processes a single file during scan
func (w *Watcher) scanFile(workflowID, filePath string, workflowDef *workflow.WorkflowDef, result *ScanResult) error {
	result.FilesScanned++

	// Double-check if file matches glob pattern before processing
	if !workflow.MatchesFileGlob(filePath, workflowDef.Options.FileGlob) {
		log.Printf("File %s does not match glob pattern %s, skipping", filePath, workflowDef.Options.FileGlob)
		result.FilesSkipped++
		return nil
	}

	// Calculate MD5
	md5Hash, fileSize, err := w.calculateMD5(filePath)
	if err != nil {
		return fmt.Errorf("failed to calculate MD5 for %s: %w", filePath, err)
	}

	now := time.Now()

	// Check if file already indexed
	existingFile, err := w.fileRepo.GetByWorkflowAndPath(workflowID, filePath)
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
		if err := w.fileRepo.Create(file); err != nil {
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
			if err := w.fileRepo.Update(existingFile); err != nil {
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
		// Wait if pending task limit is reached for this workflow
		w.waitForTaskSlot(workflowID)

		outputPath := workflow.GenerateOutputPath(filePath, workflowDef.Convert, workflowDef.Options.OutputDirPattern)

		task := &models.Task{
			WorkflowID: workflowID,
			FileID:     fileID,
			InputPath:  filePath,
			OutputPath: outputPath,
			Status:     models.TaskStatusPending,
		}

		if err := w.taskRepo.Create(task); err != nil {
			return fmt.Errorf("failed to create task: %w", err)
		}

		result.TasksCreated++
		log.Printf("Task created for file: %s -> %s", filePath, outputPath)
	}

	return nil
}

// calculateMD5 calculates the MD5 hash of a file
func (w *Watcher) calculateMD5(filePath string) (string, int64, error) {
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

// EnableWorkflow enables a workflow and starts watching it
func (w *Watcher) EnableWorkflow(workflowID string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Check if already watching
	if _, exists := w.watchedPaths[workflowID]; exists {
		log.Printf("Workflow %s is already being watched", workflowID)
		return nil
	}

	// Get workflow
	wf, err := w.workflowRepo.GetByID(workflowID)
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	// Perform initial scan
	log.Printf("Performing initial scan for enabled workflow: %s", wf.Name)
	result, err := w.scanWorkflow(workflowID)
	if err != nil {
		log.Printf("Warning: Failed to scan workflow %s: %v", wf.Name, err)
	} else {
		log.Printf("Scan completed for workflow %s: scanned=%d, new=%d, changed=%d, skipped=%d, tasks=%d",
			wf.Name, result.FilesScanned, result.FilesNew, result.FilesChanged, result.FilesSkipped, result.TasksCreated)
	}

	// Add file system watches
	if err := w.addWorkflowWatch(wf); err != nil {
		return fmt.Errorf("failed to add watch for workflow %s: %w", wf.Name, err)
	}

	log.Printf("Workflow %s enabled and watching started", wf.Name)
	return nil
}

// DisableWorkflow disables a workflow and stops watching it
func (w *Watcher) DisableWorkflow(workflowID string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Get watched paths
	paths, exists := w.watchedPaths[workflowID]
	if !exists {
		log.Printf("Workflow %s is not being watched", workflowID)
		return nil
	}

	// Remove file system watches
	for _, path := range paths {
		if err := w.watcher.Remove(path); err != nil {
			log.Printf("Warning: Failed to remove watch for path %s: %v", path, err)
		}
	}

	// Remove from watched paths map
	delete(w.watchedPaths, workflowID)

	// Cancel any pending debounce timers for this workflow
	w.debounceMu.Lock()
	for key, entry := range w.debounceMap {
		if entry.workflowID == workflowID {
			entry.timer.Stop()
			delete(w.debounceMap, key)
		}
	}
	w.debounceMu.Unlock()

	log.Printf("Workflow %s disabled and watching stopped", workflowID)
	return nil
}

// ScanWorkflow scans a workflow (public method for API)
func (w *Watcher) ScanWorkflow(workflowID string) (*ScanResult, error) {
	return w.scanWorkflow(workflowID)
}

// waitForTaskSlot waits until pending task count is below 50 for the given workflow
func (w *Watcher) waitForTaskSlot(workflowID string) {
	const maxPending = 50
	const checkInterval = 2 * time.Second

	for {
		// Check if stopped
		select {
		case <-w.stopChan:
			return
		default:
		}

		// Get pending task count for this workflow
		pendingCount, err := w.taskRepo.Count(workflowID, models.TaskStatusPending)
		if err != nil {
			log.Printf("Warning: Failed to count pending tasks for workflow %s: %v", workflowID, err)
			time.Sleep(checkInterval)
			continue
		}

		// If below limit, proceed
		if pendingCount < maxPending {
			return
		}

		// Log and wait
		log.Printf("Workflow %s: Pending task limit reached (%d/%d), waiting for tasks to be processed...", workflowID, pendingCount, maxPending)
		time.Sleep(checkInterval)
	}
}
