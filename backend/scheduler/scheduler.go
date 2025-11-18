package scheduler

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"github.com/andi/fileaction/backend/database"
	"github.com/andi/fileaction/backend/models"
)

// Scheduler handles task scheduling and execution
type Scheduler struct {
	taskRepo     *database.TaskRepo
	executorPool *ExecutorPool
	db           *database.DB
	maxRunning   int
	scanInterval time.Duration
	stopChan     chan struct{}
	wg           sync.WaitGroup
	mu           sync.Mutex
	stopped      bool
	runningTasks map[string]context.CancelFunc
}

// New creates a new scheduler
func New(db *database.DB, maxRunning int, scanInterval time.Duration, logDir string, taskTimeout, stepTimeout time.Duration) *Scheduler {
	if maxRunning <= 0 {
		maxRunning = 2 // Default maximum running tasks
	}
	if scanInterval <= 0 {
		scanInterval = 2 * time.Second // Default scan interval
	}
	if taskTimeout <= 0 {
		taskTimeout = 30 * time.Minute // Default task timeout
	}
	if stepTimeout <= 0 {
		stepTimeout = 10 * time.Minute // Default step timeout
	}

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("Failed to create log directory: %v", err)
	}

	// Create executor pool
	executorPool := NewExecutorPool(maxRunning, db, logDir, taskTimeout, stepTimeout)

	return &Scheduler{
		taskRepo:     database.NewTaskRepo(db),
		executorPool: executorPool,
		db:           db,
		maxRunning:   maxRunning,
		scanInterval: scanInterval,
		stopChan:     make(chan struct{}),
		runningTasks: make(map[string]context.CancelFunc),
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() {
	log.Printf("Starting scheduler with max %d concurrent tasks, scan interval: %v", s.maxRunning, s.scanInterval)

	s.wg.Add(1)
	go s.run()
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	s.mu.Unlock()

	log.Println("Stopping scheduler...")
	close(s.stopChan)
	s.wg.Wait()

	// Close the executor pool
	s.executorPool.Close()

	log.Println("Scheduler stopped")
}

// run is the main scheduler loop
func (s *Scheduler) run() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.scanInterval)
	defer ticker.Stop()

	// Initial scan on startup
	s.scanAndExecute()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.scanAndExecute()
		}
	}
}

// scanAndExecute scans for pending tasks and executes them if possible
func (s *Scheduler) scanAndExecute() {
	availableExecutors := s.executorPool.GetAvailableCount()
	busyExecutors := s.executorPool.GetBusyCount()

	log.Printf("Scheduler scan: busy=%d, available=%d, max=%d", busyExecutors, availableExecutors, s.maxRunning)

	if availableExecutors <= 0 {
		// No available executors, wait for one to become free
		log.Println("No available executors, skipping scan")
		return
	}

	// Get pending tasks
	tasks, err := s.taskRepo.GetPendingTasks(availableExecutors)
	if err != nil {
		log.Printf("Error getting pending tasks: %v", err)
		return
	}

	if len(tasks) == 0 {
		log.Println("No pending tasks found")
		return
	}

	log.Printf("Found %d pending task(s), %d executor(s) available", len(tasks), availableExecutors)

	// Execute tasks
	for _, task := range tasks {
		s.executeTask(task)
	}
}

// executeTask executes a single task in a goroutine
func (s *Scheduler) executeTask(task *models.Task) {
	s.wg.Add(1)
	go func(taskID string) {
		defer s.wg.Done()

		log.Printf("Starting task execution: %s", taskID)

		// Create cancellable context for the task
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		s.mu.Lock()
		s.runningTasks[taskID] = cancel
		s.mu.Unlock()

		// Acquire an executor from the pool
		executor, err := s.executorPool.Acquire(ctx)
		if err != nil {
			log.Printf("Failed to acquire executor for task %s: %v", taskID, err)
			s.mu.Lock()
			delete(s.runningTasks, taskID)
			s.mu.Unlock()
			return
		}

		// Ensure executor is released back to pool when done
		defer s.executorPool.Release(executor)
		defer func() {
			s.mu.Lock()
			delete(s.runningTasks, taskID)
			s.mu.Unlock()
		}()

		// Execute the task
		if err := executor.ExecuteTask(ctx, taskID); err != nil {
			log.Printf("Error executing task %s: %v", taskID, err)
		} else {
			log.Printf("Task execution completed: %s", taskID)
		}
	}(task.ID)
}

// CancelTask cancels a running task
func (s *Scheduler) CancelTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cancel, exists := s.runningTasks[taskID]
	if !exists {
		log.Printf("Task %s is not running", taskID)
		return nil
	}

	log.Printf("Cancelling task: %s", taskID)
	cancel()
	delete(s.runningTasks, taskID)

	// Update task status to cancelled
	if err := s.taskRepo.UpdateStatus(taskID, models.TaskStatusCancelled); err != nil {
		log.Printf("Failed to update task status: %v", err)
		return err
	}

	return nil
}

// GetRunningCount returns the current number of running tasks
func (s *Scheduler) GetRunningCount() int {
	return s.executorPool.GetBusyCount()
}

// GetMaxRunning returns the maximum number of concurrent tasks
func (s *Scheduler) GetMaxRunning() int {
	return s.maxRunning
}

// GetExecutorStatus returns the status of all executors in the pool
func (s *Scheduler) GetExecutorStatus() interface{} {
	return s.executorPool.GetExecutorStatus()
}

// GetExecutorPoolStats returns statistics about the executor pool
func (s *Scheduler) GetExecutorPoolStats() map[string]int {
	return map[string]int{
		"total":     s.executorPool.GetPoolSize(),
		"available": s.executorPool.GetAvailableCount(),
		"busy":      s.executorPool.GetBusyCount(),
	}
}
