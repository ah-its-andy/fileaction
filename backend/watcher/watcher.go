package watcher

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/andi/fileaction/backend/database"
	"github.com/andi/fileaction/backend/executor"
	"github.com/andi/fileaction/backend/models"
	"github.com/andi/fileaction/backend/scanner"
	"github.com/andi/fileaction/backend/workflow"
	"github.com/fsnotify/fsnotify"
)

// Watcher monitors file system changes and triggers workflows
type Watcher struct {
	db           *database.DB
	executor     *executor.Executor
	scanner      *scanner.Scanner
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
func New(db *database.DB, exec *executor.Executor, scan *scanner.Scanner) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		db:           db,
		executor:     exec,
		scanner:      scan,
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

	// Add watches for each workflow
	for _, wf := range workflows {
		if !wf.Enabled {
			continue
		}

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

	// Use scanner to process the file (need to export scanFile method)
	// For now, we'll create a temporary implementation
	fileRepo := database.NewFileRepo(w.db)
	taskRepo := database.NewTaskRepo(w.db)

	// Calculate file MD5
	md5Hash, fileSize, err := scanner.CalculateMD5(filePath)
	if err != nil {
		log.Printf("Error calculating MD5 for %s: %v", filePath, err)
		return
	}

	now := time.Now()
	existingFile, err := fileRepo.GetByWorkflowAndPath(wf.ID, filePath)
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
		if err := fileRepo.Create(file); err != nil {
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
			if err := fileRepo.Update(existingFile); err != nil {
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

		if err := taskRepo.Create(task); err != nil {
			log.Printf("Error creating task: %v", err)
			return
		}

		log.Printf("Task created for file: %s -> %s", filePath, outputPath)

		// Submit task to executor
		w.executor.SubmitTask(task.ID)
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
