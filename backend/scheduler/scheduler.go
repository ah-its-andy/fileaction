package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/andi/fileaction/backend/database"
	"github.com/andi/fileaction/backend/executor"
	"github.com/andi/fileaction/backend/models"
)

// Scheduler handles task scheduling and execution
type Scheduler struct {
	taskRepo     *database.TaskRepo
	executor     *executor.Executor
	maxRunning   int
	scanInterval time.Duration
	stopChan     chan struct{}
	wg           sync.WaitGroup
	mu           sync.Mutex
	stopped      bool
	runningTasks map[string]context.CancelFunc
	runningCount int
}

// New creates a new scheduler
func New(db *database.DB, exec *executor.Executor, maxRunning int, scanInterval time.Duration) *Scheduler {
	if maxRunning <= 0 {
		maxRunning = 2 // Default maximum running tasks
	}
	if scanInterval <= 0 {
		scanInterval = 2 * time.Second // Default scan interval
	}

	return &Scheduler{
		taskRepo:     database.NewTaskRepo(db),
		executor:     exec,
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
	s.mu.Lock()
	availableSlots := s.maxRunning - s.runningCount
	s.mu.Unlock()

	if availableSlots <= 0 {
		// No available slots, wait for running tasks to complete
		return
	}

	// Get pending tasks
	tasks, err := s.taskRepo.GetPendingTasks(availableSlots)
	if err != nil {
		log.Printf("Error getting pending tasks: %v", err)
		return
	}

	if len(tasks) == 0 {
		return
	}

	log.Printf("Found %d pending task(s), %d slot(s) available", len(tasks), availableSlots)

	// Execute tasks
	for _, task := range tasks {
		s.mu.Lock()
		if s.runningCount >= s.maxRunning {
			s.mu.Unlock()
			break
		}
		s.runningCount++
		s.mu.Unlock()

		s.executeTask(task)
	}
}

// executeTask executes a single task in a goroutine
func (s *Scheduler) executeTask(task *models.Task) {
	s.wg.Add(1)
	go func(taskID string) {
		defer s.wg.Done()
		defer func() {
			s.mu.Lock()
			s.runningCount--
			delete(s.runningTasks, taskID)
			s.mu.Unlock()
		}()

		log.Printf("Starting task execution: %s", taskID)

		// Create cancellable context for the task
		ctx, cancel := context.WithCancel(context.Background())

		s.mu.Lock()
		s.runningTasks[taskID] = cancel
		s.mu.Unlock()

		// Execute the task
		if err := s.executor.ExecuteTask(ctx, taskID); err != nil {
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
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.runningCount
}

// GetMaxRunning returns the maximum number of concurrent tasks
func (s *Scheduler) GetMaxRunning() int {
	return s.maxRunning
}
