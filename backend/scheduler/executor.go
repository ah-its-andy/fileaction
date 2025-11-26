package scheduler

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

// ExecutionRecord stores detailed execution information
type ExecutionRecord struct {
	TaskID      string
	StartTime   time.Time
	EndTime     time.Time
	Environment map[string]string
	Steps       []StepRecord
	LogEntries  []string
}

// StepRecord stores information about a step execution
type StepRecord struct {
	Name        string
	Command     string
	Environment map[string]string
	StartTime   time.Time
	EndTime     time.Time
	ExitCode    int
	Stdout      string
	Stderr      string
	LogEntries  []string
}

// Executor handles task execution with detailed logging
type Executor struct {
	id           int
	taskRepo     *database.TaskRepo
	stepRepo     *database.TaskStepRepo
	workflowRepo *database.WorkflowRepo
	pluginRepo   *database.PluginRepo
	logDir       string
	taskTimeout  time.Duration
	stepTimeout  time.Duration
	busy         bool
	currentTask  string
	wsHub        WebSocketHub
	wsHubMu      sync.RWMutex
}

// newExecutor creates a new executor instance
func newExecutor(id int, db *database.DB, logDir string, taskTimeout, stepTimeout time.Duration) *Executor {
	return &Executor{
		id:           id,
		taskRepo:     database.NewTaskRepo(db),
		stepRepo:     database.NewTaskStepRepo(db),
		workflowRepo: database.NewWorkflowRepo(db),
		pluginRepo:   database.NewPluginRepo(db),
		logDir:       logDir,
		taskTimeout:  taskTimeout,
		stepTimeout:  stepTimeout,
		busy:         false,
	}
}

// IsBusy returns whether the executor is currently busy
func (e *Executor) IsBusy() bool {
	return e.busy
}

// GetID returns the executor's ID
func (e *Executor) GetID() int {
	return e.id
}

// GetCurrentTask returns the ID of the current task being executed
func (e *Executor) GetCurrentTask() string {
	return e.currentTask
}

// SetWebSocketHub sets the WebSocket hub for real-time log broadcasting
func (e *Executor) SetWebSocketHub(hub WebSocketHub) {
	e.wsHubMu.Lock()
	defer e.wsHubMu.Unlock()
	e.wsHub = hub
}

// broadcastLog sends log content to WebSocket clients if hub is available
func (e *Executor) broadcastLog(taskID, content string) {
	e.wsHubMu.RLock()
	defer e.wsHubMu.RUnlock()
	if e.wsHub != nil {
		e.wsHub.BroadcastLog(taskID, content)
	}
}

// broadcastTaskComplete notifies WebSocket clients that task is complete
func (e *Executor) broadcastTaskComplete(taskID string) {
	e.wsHubMu.RLock()
	defer e.wsHubMu.RUnlock()
	if e.wsHub != nil {
		e.wsHub.BroadcastTaskComplete(taskID)
	}
}

// ExecuteTask executes a single task with detailed logging
func (e *Executor) ExecuteTask(ctx context.Context, taskID string) error {
	e.busy = true
	e.currentTask = taskID
	defer func() {
		e.busy = false
		e.currentTask = ""
	}()

	// Get task
	task, err := e.taskRepo.GetByID(taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Check if task is already running or completed
	if task.Status != models.TaskStatusPending {
		log.Printf("[Executor-%d] Task %s is not pending (status: %s), skipping", e.id, taskID, task.Status)
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

	// Create context with timeout if not provided
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), e.taskTimeout)
		defer cancel()
	}

	// Create log file
	logFilePath := filepath.Join(e.logDir, fmt.Sprintf("%s.log", taskID))
	logFile, err := os.Create(logFilePath)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer logFile.Close()

	logWriter := bufio.NewWriter(logFile)
	defer logWriter.Flush()

	// Create execution record
	execRecord := &ExecutionRecord{
		TaskID:      taskID,
		StartTime:   time.Now(),
		Environment: make(map[string]string),
		Steps:       make([]StepRecord, 0),
		LogEntries:  make([]string, 0),
	}

	// Record global environment variables
	for key, value := range workflowDef.Env {
		execRecord.Environment[key] = value
	}

	// Update task status to running
	now := time.Now()
	task.Status = models.TaskStatusRunning
	task.StartedAt = &now
	if err := e.taskRepo.Update(task); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	e.writeLog(logWriter, execRecord, fmt.Sprintf("[Executor-%d] Task started", e.id))
	e.writeLog(logWriter, execRecord, fmt.Sprintf("Input: %s", task.InputPath))
	e.writeLog(logWriter, execRecord, fmt.Sprintf("Output: %s", task.OutputPath))
	e.writeLog(logWriter, execRecord, fmt.Sprintf("Workflow: %s", wf.Name))

	// Log environment variables
	if len(workflowDef.Env) > 0 {
		e.writeLog(logWriter, execRecord, "Environment variables:")
		for key, value := range workflowDef.Env {
			e.writeLog(logWriter, execRecord, fmt.Sprintf("  %s=%s", key, value))
		}
	}

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(task.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		e.writeLog(logWriter, execRecord, fmt.Sprintf("ERROR: Failed to create output directory: %v", err))
		task.Status = models.TaskStatusFailed
		task.ErrorMessage = fmt.Sprintf("Failed to create output directory: %v", err)
		completedAt := time.Now()
		task.CompletedAt = &completedAt
		e.taskRepo.Update(task)
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	e.writeLog(logWriter, execRecord, fmt.Sprintf("Output directory: %s", outputDir))

	// Get variables for substitution
	vars := workflow.GetVariables(task.InputPath, task.OutputPath)

	// Execute steps
	allStepsSucceeded := true
	workflowStoppedWithSuccess := false
	workflowStoppedWithFailure := false

	for i, step := range workflowDef.Steps {
		e.writeLog(logWriter, execRecord, fmt.Sprintf("\n--- Step %d: %s ---", i+1, step.Name))

		// Check if this is a plugin step
		if step.Uses != "" {
			e.writeLog(logWriter, execRecord, fmt.Sprintf("Plugin: %s", step.Uses))

			// Execute plugin
			pluginErr := e.executePluginStep(ctx, taskID, step, vars, workflowDef.Env, logWriter, execRecord)
			if pluginErr != nil {
				// Check for workflow control errors
				if stopSuccess, ok := pluginErr.(*WorkflowStopSuccess); ok {
					e.writeLog(logWriter, execRecord, fmt.Sprintf("INFO: %s", stopSuccess.Message))
					workflowStoppedWithSuccess = true
					break
				}
				if stopFailure, ok := pluginErr.(*WorkflowStopFailure); ok {
					e.writeLog(logWriter, execRecord, fmt.Sprintf("INFO: %s", stopFailure.Message))
					workflowStoppedWithFailure = true
					allStepsSucceeded = false
					break
				}

				e.writeLog(logWriter, execRecord, fmt.Sprintf("ERROR: Plugin step failed: %v", pluginErr))
				allStepsSucceeded = false
				break
			}

			// Check if context was cancelled
			if ctx.Err() != nil {
				e.writeLog(logWriter, execRecord, "Task cancelled or timed out")
				allStepsSucceeded = false
				break
			}

			continue
		}

		// Create step record
		stepModel := &models.TaskStep{
			TaskID:  taskID,
			Name:    step.Name,
			Command: step.Run,
			Status:  models.StepStatusPending,
		}
		if err := e.stepRepo.Create(stepModel); err != nil {
			e.writeLog(logWriter, execRecord, fmt.Sprintf("ERROR: Failed to create step record: %v", err))
			allStepsSucceeded = false
			break
		}

		// Execute step and get detailed record
		stepRecord, err := e.executeStep(ctx, stepModel, step, vars, workflowDef.Env, logWriter, execRecord)
		if stepRecord != nil {
			execRecord.Steps = append(execRecord.Steps, *stepRecord)
		}

		if err != nil {
			// Check for workflow control errors
			if stopSuccess, ok := err.(*WorkflowStopSuccess); ok {
				e.writeLog(logWriter, execRecord, fmt.Sprintf("INFO: %s", stopSuccess.Message))
				workflowStoppedWithSuccess = true
				break
			}
			if stopFailure, ok := err.(*WorkflowStopFailure); ok {
				e.writeLog(logWriter, execRecord, fmt.Sprintf("INFO: %s", stopFailure.Message))
				workflowStoppedWithFailure = true
				allStepsSucceeded = false
				break
			}

			// Regular step failure
			e.writeLog(logWriter, execRecord, fmt.Sprintf("ERROR: Step failed: %v", err))
			allStepsSucceeded = false
			break
		}

		// Check if context was cancelled
		if ctx.Err() != nil {
			e.writeLog(logWriter, execRecord, "Task cancelled or timed out")
			allStepsSucceeded = false
			break
		}
	}

	execRecord.EndTime = time.Now()

	// Update task status
	completedAt := time.Now()
	task.CompletedAt = &completedAt

	if workflowStoppedWithSuccess || allStepsSucceeded {
		task.Status = models.TaskStatusCompleted
		e.writeLog(logWriter, execRecord, fmt.Sprintf("\n[Executor-%d] Task completed successfully", e.id))
	} else {
		task.Status = models.TaskStatusFailed
		if workflowStoppedWithFailure {
			task.ErrorMessage = "Workflow stopped with failure"
		} else {
			task.ErrorMessage = "One or more steps failed"
		}
		e.writeLog(logWriter, execRecord, fmt.Sprintf("\n[Executor-%d] Task failed", e.id))
	}

	duration := execRecord.EndTime.Sub(execRecord.StartTime)
	e.writeLog(logWriter, execRecord, fmt.Sprintf("Total execution time: %v", duration))

	logWriter.Flush()

	// Read log file content and store in database
	logContent, err := os.ReadFile(logFilePath)
	if err != nil {
		log.Printf("[Executor-%d] Failed to read log file: %v", e.id, err)
	} else {
		task.LogText = string(logContent)
	}

	if err := e.taskRepo.Update(task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Broadcast task completion to WebSocket clients
	e.broadcastTaskComplete(taskID)

	// Remove log file after importing to database
	if err := os.Remove(logFilePath); err != nil {
		log.Printf("[Executor-%d] Failed to remove log file: %v", e.id, err)
	}

	log.Printf("[Executor-%d] Task %s completed with status: %s (duration: %v)", e.id, taskID, task.Status, duration)
	return nil
}

// executeStep executes a single step with detailed logging
func (e *Executor) executeStep(ctx context.Context, stepModel *models.TaskStep, step workflow.Step, vars workflow.Variables, globalEnv map[string]string, logWriter *bufio.Writer, execRecord *ExecutionRecord) (*StepRecord, error) {
	stepRecord := &StepRecord{
		Name:        step.Name,
		Command:     step.Run,
		Environment: make(map[string]string),
		StartTime:   time.Now(),
		LogEntries:  make([]string, 0),
	}

	// Substitute variables in command
	command := workflow.SubstituteVariables(step.Run, vars)
	stepRecord.Command = command
	e.writeLog(logWriter, execRecord, fmt.Sprintf("Command: %s", command))

	// Update step status to running
	now := time.Now()
	stepModel.Status = models.StepStatusRunning
	stepModel.StartedAt = &now
	if err := e.stepRepo.Update(stepModel); err != nil {
		return stepRecord, fmt.Errorf("failed to update step status: %w", err)
	}

	// Create context with step timeout
	stepCtx, cancel := context.WithTimeout(ctx, e.stepTimeout)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(stepCtx, "sh", "-c", command)

	// Set environment variables
	cmd.Env = os.Environ()

	// Add global environment variables
	for key, value := range globalEnv {
		envVar := fmt.Sprintf("%s=%s", key, value)
		cmd.Env = append(cmd.Env, envVar)
		stepRecord.Environment[key] = value
	}

	// Add step-specific environment variables
	for key, value := range step.Env {
		substValue := workflow.SubstituteVariables(value, vars)
		envVar := fmt.Sprintf("%s=%s", key, substValue)
		cmd.Env = append(cmd.Env, envVar)
		stepRecord.Environment[key] = substValue
	}

	// Log environment variables for this step
	if len(step.Env) > 0 {
		e.writeLog(logWriter, execRecord, "Step environment variables:")
		for key, value := range step.Env {
			substValue := workflow.SubstituteVariables(value, vars)
			e.writeLog(logWriter, execRecord, fmt.Sprintf("  %s=%s", key, substValue))
		}
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	e.writeLog(logWriter, execRecord, "Executing command...")

	// Execute command
	err := cmd.Run()
	stepRecord.EndTime = time.Now()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}
	stepRecord.ExitCode = exitCode

	// Write output to log
	if stdout.Len() > 0 {
		stepRecord.Stdout = stdout.String()
		e.writeLog(logWriter, execRecord, fmt.Sprintf("STDOUT:\n%s", stdout.String()))
	}
	if stderr.Len() > 0 {
		stepRecord.Stderr = stderr.String()
		e.writeLog(logWriter, execRecord, fmt.Sprintf("STDERR:\n%s", stderr.String()))
	}

	duration := stepRecord.EndTime.Sub(stepRecord.StartTime)
	e.writeLog(logWriter, execRecord, fmt.Sprintf("Exit code: %d", exitCode))
	e.writeLog(logWriter, execRecord, fmt.Sprintf("Step duration: %v", duration))

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
		e.writeLog(logWriter, execRecord, "INFO: Workflow stopped with success (exit code 100)")
	case 101:
		// Failure and stop workflow
		stepModel.Status = models.StepStatusFailed
		stopWorkflow = true
		forceTaskFailure = true
		e.writeLog(logWriter, execRecord, "INFO: Workflow stopped with failure (exit code 101)")
	default:
		stepModel.Status = models.StepStatusFailed
	}

	if err := e.stepRepo.Update(stepModel); err != nil {
		return stepRecord, fmt.Errorf("failed to update step: %w", err)
	}

	// Return special error types for workflow control
	if stopWorkflow {
		if forceTaskSuccess {
			return stepRecord, &WorkflowStopSuccess{Message: "Workflow stopped with success"}
		}
		if forceTaskFailure {
			return stepRecord, &WorkflowStopFailure{Message: "Workflow stopped with failure"}
		}
	}

	if exitCode != 0 && exitCode != 100 {
		return stepRecord, fmt.Errorf("step exited with code %d", exitCode)
	}

	return stepRecord, nil
}

// writeLog writes a timestamped log entry to both the writer and execution record
// and broadcasts it via WebSocket if available
func (e *Executor) writeLog(w *bufio.Writer, record *ExecutionRecord, message string) {
	timestamp := time.Now().Format(time.RFC3339)
	logEntry := fmt.Sprintf("[%s] %s\n", timestamp, message)
	fmt.Fprint(w, logEntry)
	if record != nil {
		record.LogEntries = append(record.LogEntries, logEntry)
		// Broadcast to WebSocket clients
		e.broadcastLog(record.TaskID, logEntry)
	}
}

// executePluginStep executes a plugin-based step
func (e *Executor) executePluginStep(ctx context.Context, taskID string, step workflow.Step, vars workflow.Variables, globalEnv map[string]string, logWriter *bufio.Writer, execRecord *ExecutionRecord) error {
	// Parse plugin reference
	pluginName, version, err := workflow.ParsePluginReference(step.Uses)
	if err != nil {
		return fmt.Errorf("invalid plugin reference: %w", err)
	}

	e.writeLog(logWriter, execRecord, fmt.Sprintf("Loading plugin: %s (version: %s)", pluginName, version))

	// Get plugin version from database
	var pluginVersion *database.PluginVersion
	var loadErr error
	if version != "" {
		pluginVersion, loadErr = e.pluginRepo.GetPluginVersionByNumber(pluginName, version)
	} else {
		// Get current version if no version specified
		plugin, pluginErr := e.pluginRepo.GetPluginByName(pluginName)
		if pluginErr != nil {
			return fmt.Errorf("plugin not found: %w", pluginErr)
		}
		pluginVersion, loadErr = e.pluginRepo.GetPluginCurrentVersion(plugin.ID)
	}

	if loadErr != nil {
		return fmt.Errorf("failed to load plugin: %w", loadErr)
	}

	// Parse plugin definition
	pluginDef, err := workflow.ParsePlugin(pluginVersion.YAMLContent)
	if err != nil {
		return fmt.Errorf("failed to parse plugin: %w", err)
	}

	e.writeLog(logWriter, execRecord, fmt.Sprintf("Plugin loaded: %s v%s", pluginDef.Name, pluginDef.Version))
	e.writeLog(logWriter, execRecord, fmt.Sprintf("Description: %s", pluginDef.Description))

	// Validate dependencies
	if len(pluginDef.Dependencies) > 0 {
		e.writeLog(logWriter, execRecord, "Checking dependencies...")
		if err := workflow.ValidatePluginDependencies(pluginDef.Dependencies); err != nil {
			e.writeLog(logWriter, execRecord, fmt.Sprintf("ERROR: Dependency check failed: %v", err))
			return fmt.Errorf("dependency check failed: %w", err)
		}
		e.writeLog(logWriter, execRecord, "All dependencies satisfied")
	}

	// Prepare inputs
	inputs, err := workflow.PreparePluginInputs(pluginDef, step.With)
	if err != nil {
		return fmt.Errorf("failed to prepare inputs: %w", err)
	}

	if len(inputs) > 0 {
		e.writeLog(logWriter, execRecord, "Plugin inputs:")
		for key, value := range inputs {
			e.writeLog(logWriter, execRecord, fmt.Sprintf("  %s: %s", key, value))
		}
	}

	// Execute plugin steps
	for i, pluginStep := range pluginDef.Steps {
		e.writeLog(logWriter, execRecord, fmt.Sprintf("\n  Plugin Step %d: %s", i+1, pluginStep.Name))

		// Evaluate condition
		if pluginStep.Condition != "" {
			shouldExecute := workflow.EvaluateCondition(pluginStep.Condition, inputs, vars)
			e.writeLog(logWriter, execRecord, fmt.Sprintf("  Condition: %s = %v", pluginStep.Condition, shouldExecute))
			if !shouldExecute {
				e.writeLog(logWriter, execRecord, "  Skipping step (condition not met)")
				continue
			}
		}

		// Create step record
		stepModel := &models.TaskStep{
			TaskID:  taskID,
			Name:    fmt.Sprintf("%s / %s", step.Name, pluginStep.Name),
			Command: pluginStep.Run,
			Status:  models.StepStatusPending,
		}
		if err := e.stepRepo.Create(stepModel); err != nil {
			e.writeLog(logWriter, execRecord, fmt.Sprintf("  ERROR: Failed to create step record: %v", err))
			return err
		}

		// Substitute inputs and variables in command
		command := workflow.SubstitutePluginInputs(pluginStep.Run, inputs)
		command = workflow.SubstituteVariables(command, vars)

		e.writeLog(logWriter, execRecord, fmt.Sprintf("  Command: %s", command))

		// Update step status to running
		now := time.Now()
		stepModel.Status = models.StepStatusRunning
		stepModel.StartedAt = &now
		stepModel.Command = command
		if err := e.stepRepo.Update(stepModel); err != nil {
			return fmt.Errorf("failed to update step status: %w", err)
		}

		// Create context with step timeout (use plugin timeout if specified)
		timeout := e.stepTimeout
		if pluginStep.Timeout > 0 {
			timeout = time.Duration(pluginStep.Timeout) * time.Second
		}
		stepCtx, cancel := context.WithTimeout(ctx, timeout)

		// Create command
		cmd := exec.CommandContext(stepCtx, "sh", "-c", command)

		// Merge environment variables
		mergedEnv := workflow.MergeEnvironment(
			make(map[string]string), // base env (we use os.Environ() instead)
			globalEnv,
			pluginDef.Env,
			pluginStep.Env,
		)

		cmd.Env = os.Environ()
		for key, value := range mergedEnv {
			substValue := workflow.SubstituteVariables(value, vars)
			substValue = workflow.SubstitutePluginInputs(substValue, inputs)
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, substValue))
		}

		// Capture output
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		e.writeLog(logWriter, execRecord, "  Executing command...")

		// Execute command
		startTime := time.Now()
		err := cmd.Run()
		endTime := time.Now()
		cancel() // Clean up context

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
			e.writeLog(logWriter, execRecord, fmt.Sprintf("  STDOUT:\n%s", stdout.String()))
		}
		if stderr.Len() > 0 {
			e.writeLog(logWriter, execRecord, fmt.Sprintf("  STDERR:\n%s", stderr.String()))
		}

		duration := endTime.Sub(startTime)
		e.writeLog(logWriter, execRecord, fmt.Sprintf("  Exit code: %d", exitCode))
		e.writeLog(logWriter, execRecord, fmt.Sprintf("  Duration: %v", duration))

		// Update step
		completedAt := time.Now()
		stepModel.CompletedAt = &completedAt
		stepModel.ExitCode = &exitCode
		stepModel.Stdout = stdout.String()
		stepModel.Stderr = stderr.String()

		// Handle exit codes
		stopWorkflow := false
		forceTaskSuccess := false
		forceTaskFailure := false

		switch exitCode {
		case 0:
			stepModel.Status = models.StepStatusCompleted
		case 100:
			stepModel.Status = models.StepStatusCompleted
			stopWorkflow = true
			forceTaskSuccess = true
			e.writeLog(logWriter, execRecord, "  INFO: Workflow stopped with success (exit code 100)")
		case 101:
			stepModel.Status = models.StepStatusFailed
			stopWorkflow = true
			forceTaskFailure = true
			e.writeLog(logWriter, execRecord, "  INFO: Workflow stopped with failure (exit code 101)")
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
			return fmt.Errorf("plugin step '%s' exited with code %d", pluginStep.Name, exitCode)
		}

		// Check if context was cancelled
		if ctx.Err() != nil {
			return fmt.Errorf("task cancelled or timed out")
		}
	}

	e.writeLog(logWriter, execRecord, fmt.Sprintf("Plugin '%s' completed successfully", pluginDef.Name))
	return nil
}
