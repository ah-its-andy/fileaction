package executor

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/andi/fileaction/backend/database"
	"github.com/andi/fileaction/backend/models"
	"github.com/andi/fileaction/backend/workflow"
)

// WorkflowStopSuccess indicates workflow should stop with success status
type WorkflowStopSuccess struct {
	Message string
}

func (e *WorkflowStopSuccess) Error() string {
	return e.Message
}

// WorkflowStopFailure indicates workflow should stop with failure status
type WorkflowStopFailure struct {
	Message string
}

func (e *WorkflowStopFailure) Error() string {
	return e.Message
}

// Executor handles task execution
type Executor struct {
	taskRepo     *database.TaskRepo
	stepRepo     *database.TaskStepRepo
	workflowRepo *database.WorkflowRepo
	logDir       string
	concurrency  int
	taskTimeout  time.Duration
	stepTimeout  time.Duration
	workers      int
	taskQueue    chan string
	stopChan     chan struct{}
	stopped      bool
	wg           sync.WaitGroup
	mu           sync.Mutex
	runningTasks map[string]context.CancelFunc
}

// New creates a new executor
func New(db *database.DB, logDir string, concurrency int, taskTimeout, stepTimeout time.Duration) *Executor {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("Failed to create log directory: %v", err)
	}

	return &Executor{
		taskRepo:     database.NewTaskRepo(db),
		stepRepo:     database.NewTaskStepRepo(db),
		workflowRepo: database.NewWorkflowRepo(db),
		logDir:       logDir,
		concurrency:  concurrency,
		taskTimeout:  taskTimeout,
		stepTimeout:  stepTimeout,
		taskQueue:    make(chan string, 100),
		stopChan:     make(chan struct{}),
		runningTasks: make(map[string]context.CancelFunc),
	}
}

// Start starts the executor workers
func (e *Executor) Start() {
	log.Printf("Starting executor with %d workers", e.concurrency)
	for i := 0; i < e.concurrency; i++ {
		e.wg.Add(1)
		go e.worker(i)
	}
}

// Stop stops the executor
func (e *Executor) Stop() {
	e.mu.Lock()
	if e.stopped {
		e.mu.Unlock()
		return
	}
	e.stopped = true
	e.mu.Unlock()

	log.Println("Stopping executor...")
	close(e.stopChan)
	e.wg.Wait()
	log.Println("Executor stopped")
}

// SubmitTask submits a task for execution
func (e *Executor) SubmitTask(taskID string) {
	select {
	case e.taskQueue <- taskID:
		log.Printf("Task %s submitted to queue", taskID)
	default:
		log.Printf("Task queue full, cannot submit task %s", taskID)
	}
}

// CancelTask cancels a running task
func (e *Executor) CancelTask(taskID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	cancel, exists := e.runningTasks[taskID]
	if !exists {
		return fmt.Errorf("task %s is not running", taskID)
	}

	cancel()
	delete(e.runningTasks, taskID)

	// Update task status
	if err := e.taskRepo.UpdateStatus(taskID, models.TaskStatusCancelled); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	log.Printf("Task %s cancelled", taskID)
	return nil
}

// worker processes tasks from the queue
func (e *Executor) worker(id int) {
	defer e.wg.Done()
	log.Printf("Worker %d started", id)

	for {
		select {
		case <-e.stopChan:
			log.Printf("Worker %d stopping", id)
			return
		case taskID := <-e.taskQueue:
			log.Printf("Worker %d processing task %s", id, taskID)
			if err := e.executeTask(taskID); err != nil {
				log.Printf("Worker %d failed to execute task %s: %v", id, taskID, err)
			}
		}
	}
}

// executeTask executes a single task
func (e *Executor) executeTask(taskID string) error {
	// Get task
	task, err := e.taskRepo.GetByID(taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Check if task is already running or completed
	if task.Status != models.TaskStatusPending {
		log.Printf("Task %s is not pending (status: %s), skipping", taskID, task.Status)
		return nil
	}

	// Get workflow
	wf, err := e.workflowRepo.GetByID(task.WorkflowID)
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	// Parse workflow
	workflowDef, err := workflow.Parse(wf.YAMLContent)
	if err != nil {
		return fmt.Errorf("failed to parse workflow: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), e.taskTimeout)
	defer cancel()

	// Register running task
	e.mu.Lock()
	e.runningTasks[taskID] = cancel
	e.mu.Unlock()

	// Remove from running tasks when done
	defer func() {
		e.mu.Lock()
		delete(e.runningTasks, taskID)
		e.mu.Unlock()
	}()

	// Create log file
	logFilePath := filepath.Join(e.logDir, fmt.Sprintf("%s.log", taskID))
	logFile, err := os.Create(logFilePath)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer logFile.Close()

	logWriter := bufio.NewWriter(logFile)
	defer logWriter.Flush()

	// Update task status to running
	now := time.Now()
	task.Status = models.TaskStatusRunning
	task.StartedAt = &now
	if err := e.taskRepo.Update(task); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	writeLog(logWriter, "Task started")
	writeLog(logWriter, fmt.Sprintf("Input: %s", task.InputPath))
	writeLog(logWriter, fmt.Sprintf("Output: %s", task.OutputPath))

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(task.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		writeLog(logWriter, fmt.Sprintf("ERROR: Failed to create output directory: %v", err))
		task.Status = models.TaskStatusFailed
		task.ErrorMessage = fmt.Sprintf("Failed to create output directory: %v", err)
		completedAt := time.Now()
		task.CompletedAt = &completedAt
		e.taskRepo.Update(task)
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	writeLog(logWriter, fmt.Sprintf("Output directory: %s", outputDir))

	// Get variables for substitution
	vars := workflow.GetVariables(task.InputPath, task.OutputPath)

	// Execute steps
	allStepsSucceeded := true
	workflowStoppedWithSuccess := false
	workflowStoppedWithFailure := false

	for i, step := range workflowDef.Steps {
		writeLog(logWriter, fmt.Sprintf("\n--- Step %d: %s ---", i+1, step.Name))

		// Create step record
		stepModel := &models.TaskStep{
			TaskID:  taskID,
			Name:    step.Name,
			Command: step.Run,
			Status:  models.StepStatusPending,
		}
		if err := e.stepRepo.Create(stepModel); err != nil {
			writeLog(logWriter, fmt.Sprintf("ERROR: Failed to create step record: %v", err))
			allStepsSucceeded = false
			break
		}

		// Execute step
		if err := e.executeStep(ctx, stepModel, step, vars, workflowDef.Env, logWriter); err != nil {
			// Check for workflow control errors
			if stopSuccess, ok := err.(*WorkflowStopSuccess); ok {
				writeLog(logWriter, fmt.Sprintf("INFO: %s", stopSuccess.Message))
				workflowStoppedWithSuccess = true
				break
			}
			if stopFailure, ok := err.(*WorkflowStopFailure); ok {
				writeLog(logWriter, fmt.Sprintf("INFO: %s", stopFailure.Message))
				workflowStoppedWithFailure = true
				allStepsSucceeded = false
				break
			}

			// Regular step failure
			writeLog(logWriter, fmt.Sprintf("ERROR: Step failed: %v", err))
			allStepsSucceeded = false
			break
		}

		// Check if context was cancelled
		if ctx.Err() != nil {
			writeLog(logWriter, "Task cancelled or timed out")
			allStepsSucceeded = false
			break
		}
	}

	// Update task status
	completedAt := time.Now()
	task.CompletedAt = &completedAt

	if workflowStoppedWithSuccess || allStepsSucceeded {
		task.Status = models.TaskStatusCompleted
		writeLog(logWriter, "\nTask completed successfully")
	} else {
		task.Status = models.TaskStatusFailed
		if workflowStoppedWithFailure {
			task.ErrorMessage = "Workflow stopped with failure"
		} else {
			task.ErrorMessage = "One or more steps failed"
		}
		writeLog(logWriter, "\nTask failed")
	}

	logWriter.Flush()

	// Read log file content and store in database
	logContent, err := os.ReadFile(logFilePath)
	if err != nil {
		log.Printf("Failed to read log file: %v", err)
	} else {
		task.LogText = string(logContent)
	}

	if err := e.taskRepo.Update(task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Remove log file after importing to database
	if err := os.Remove(logFilePath); err != nil {
		log.Printf("Failed to remove log file: %v", err)
	}

	log.Printf("Task %s completed with status: %s", taskID, task.Status)
	return nil
}

// executeStep executes a single step
func (e *Executor) executeStep(ctx context.Context, stepModel *models.TaskStep, step workflow.Step, vars workflow.Variables, globalEnv map[string]string, logWriter *bufio.Writer) error {
	// Substitute variables in command
	command := workflow.SubstituteVariables(step.Run, vars)
	writeLog(logWriter, fmt.Sprintf("Command: %s", command))

	// Update step status to running
	now := time.Now()
	stepModel.Status = models.StepStatusRunning
	stepModel.StartedAt = &now
	if err := e.stepRepo.Update(stepModel); err != nil {
		return fmt.Errorf("failed to update step status: %w", err)
	}

	// Create context with step timeout
	stepCtx, cancel := context.WithTimeout(ctx, e.stepTimeout)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(stepCtx, "sh", "-c", command)

	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range globalEnv {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}
	for key, value := range step.Env {
		substValue := workflow.SubstituteVariables(value, vars)
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, substValue))
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute command
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	// Write output to log
	if stdout.Len() > 0 {
		writeLog(logWriter, fmt.Sprintf("STDOUT:\n%s", stdout.String()))
	}
	if stderr.Len() > 0 {
		writeLog(logWriter, fmt.Sprintf("STDERR:\n%s", stderr.String()))
	}
	writeLog(logWriter, fmt.Sprintf("Exit code: %d", exitCode))

	// Update step
	completedAt := time.Now()
	stepModel.CompletedAt = &completedAt
	stepModel.ExitCode = &exitCode
	stepModel.Stdout = stdout.String()
	stepModel.Stderr = stderr.String()

	// Handle special exit codes:
	// 0: Success (continue to next step)
	// 100: Success and stop workflow (task succeeds)
	// 101: Failure and stop workflow (task fails)
	// Other non-zero: Step failure (task fails)
	stopWorkflow := false
	forceTaskSuccess := false
	forceTaskFailure := false

	switch exitCode {
	case 0:
		stepModel.Status = models.StepStatusCompleted
	case 100:
		// Success and stop workflow
		stepModel.Status = models.StepStatusCompleted
		stopWorkflow = true
		forceTaskSuccess = true
		writeLog(logWriter, "INFO: Workflow stopped with success (exit code 100)")
	case 101:
		// Failure and stop workflow
		stepModel.Status = models.StepStatusFailed
		stopWorkflow = true
		forceTaskFailure = true
		writeLog(logWriter, "INFO: Workflow stopped with failure (exit code 101)")
	default:
		stepModel.Status = models.StepStatusFailed
	}

	if err := e.stepRepo.Update(stepModel); err != nil {
		return fmt.Errorf("failed to update step: %w", err)
	}

	// Return special error types for workflow control
	if stopWorkflow {
		if forceTaskSuccess {
			return &WorkflowStopSuccess{Message: "Workflow stopped with success"}
		}
		if forceTaskFailure {
			return &WorkflowStopFailure{Message: "Workflow stopped with failure"}
		}
	}

	if exitCode != 0 && exitCode != 100 {
		return fmt.Errorf("step exited with code %d", exitCode)
	}

	return nil
}

// writeLog writes a timestamped log entry
func writeLog(w *bufio.Writer, message string) {
	timestamp := time.Now().Format(time.RFC3339)
	fmt.Fprintf(w, "[%s] %s\n", timestamp, message)
}

// ProcessPendingTasks fetches pending tasks and submits them
func (e *Executor) ProcessPendingTasks() error {
	tasks, err := e.taskRepo.GetPendingTasks(100)
	if err != nil {
		return fmt.Errorf("failed to get pending tasks: %w", err)
	}

	log.Printf("Found %d pending tasks", len(tasks))
	for _, task := range tasks {
		e.SubmitTask(task.ID)
	}

	return nil
}
