package api

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/andi/fileaction/backend/database"
	"github.com/andi/fileaction/backend/models"
	"github.com/andi/fileaction/backend/watcher"
	"github.com/andi/fileaction/backend/workflow"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/html/v2"
)

// TaskCanceller defines the interface for cancelling tasks
type TaskCanceller interface {
	CancelTask(taskID string) error
}

// SchedulerStats defines the interface for getting scheduler statistics
type SchedulerStats interface {
	GetExecutorPoolStats() map[string]int
	GetExecutorStatus() interface{}
}

// Scheduler combines both interfaces
type Scheduler interface {
	TaskCanceller
	SchedulerStats
}

// Server represents the HTTP API server
type Server struct {
	app       *fiber.App
	db        *database.DB
	scheduler Scheduler
	watcher   *watcher.Watcher
	logDir    string
}

// New creates a new API server
func New(db *database.DB, scheduler Scheduler, watch *watcher.Watcher, logDir string) *Server {
	// Initialize HTML template engine
	engine := html.New("./frontend/templates", ".html")

	app := fiber.New(fiber.Config{
		Views:        engine,
		ErrorHandler: errorHandler,
	})

	// Middleware
	app.Use(recover.New())

	// Configure logger to write only to file
	accessLogPath := filepath.Join(logDir, "access.log")
	accessLogFile, err := os.OpenFile(accessLogPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("Warning: Failed to open access log file: %v", err)
		// If file creation fails, disable logging entirely by using io.Discard
		app.Use(logger.New(logger.Config{
			Output: io.Discard,
		}))
	} else {
		// Write access logs only to file, not to console
		app.Use(logger.New(logger.Config{
			Output: accessLogFile,
		}))
	}

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	server := &Server{
		app:       app,
		db:        db,
		scheduler: scheduler,
		watcher:   watch,
		logDir:    logDir,
	}

	server.setupRoutes()
	return server
}

// setupRoutes sets up all API routes
func (s *Server) setupRoutes() {
	// Home page with server-side rendering
	s.app.Get("/", s.renderIndex)

	// Static files
	s.app.Static("/static", "./frontend/static")

	// API routes
	api := s.app.Group("/api")

	// Workflows
	api.Get("/workflows", s.listWorkflows)
	api.Post("/workflows", s.createWorkflow)
	api.Get("/workflows/:id", s.getWorkflow)
	api.Put("/workflows/:id", s.updateWorkflow)
	api.Put("/workflows/:id/toggle", s.toggleWorkflow)
	api.Delete("/workflows/:id", s.deleteWorkflow)
	api.Post("/workflows/:id/scan", s.scanWorkflow)
	api.Post("/workflows/:id/clear-index", s.clearWorkflowIndex)

	// Tasks
	api.Get("/tasks", s.listTasks)
	api.Get("/tasks/:id", s.getTask)
	api.Post("/tasks/:id/retry", s.retryTask)
	api.Post("/tasks/:id/cancel", s.cancelTask)
	api.Delete("/tasks/:id", s.deleteTask)
	api.Get("/tasks/:id/steps", s.getTaskSteps)
	api.Get("/tasks/:id/log/tail", s.tailTaskLog)

	// Files
	api.Get("/files", s.listFiles)

	// Scheduler/Monitoring
	api.Get("/scheduler/stats", s.getSchedulerStats)
	api.Get("/scheduler/executors", s.getExecutorStatus)
}

// Start starts the HTTP server
func (s *Server) Start(addr string) error {
	log.Printf("Starting HTTP server on %s", addr)
	return s.app.Listen(addr)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	return s.app.Shutdown()
}

// Error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// Success response
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// errorHandler handles fiber errors
func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}
	return c.Status(code).JSON(ErrorResponse{Error: err.Error()})
}

// ============== Page Rendering ==============

func (s *Server) renderIndex(c *fiber.Ctx) error {
	return c.Render("index", fiber.Map{
		"Title": "FileAction - Workflow Automation",
	})
}

// ============== Workflow Handlers ==============

func (s *Server) listWorkflows(c *fiber.Ctx) error {
	repo := database.NewWorkflowRepo(s.db)
	workflows, err := repo.List()
	if err != nil {
		return c.Status(500).JSON(ErrorResponse{Error: err.Error()})
	}
	return c.JSON(workflows)
}

type CreateWorkflowRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	YAMLContent string `json:"yaml_content"`
	Enabled     bool   `json:"enabled"`
}

func (s *Server) createWorkflow(c *fiber.Ctx) error {
	var req CreateWorkflowRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(ErrorResponse{Error: "Invalid request body"})
	}

	// Validate YAML
	workflowDef, err := workflow.Parse(req.YAMLContent)
	if err != nil {
		return c.Status(400).JSON(ErrorResponse{Error: fmt.Sprintf("Invalid workflow YAML: %v", err)})
	}

	if err := workflow.Validate(workflowDef); err != nil {
		return c.Status(400).JSON(ErrorResponse{Error: fmt.Sprintf("Workflow validation failed: %v", err)})
	}

	// Create workflow
	wf := &models.Workflow{
		Name:        req.Name,
		Description: req.Description,
		YAMLContent: req.YAMLContent,
		Enabled:     req.Enabled,
	}

	repo := database.NewWorkflowRepo(s.db)
	if err := repo.Create(wf); err != nil {
		return c.Status(500).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.Status(201).JSON(wf)
}

func (s *Server) getWorkflow(c *fiber.Ctx) error {
	id := c.Params("id")
	repo := database.NewWorkflowRepo(s.db)

	wf, err := repo.GetByID(id)
	if err != nil {
		return c.Status(404).JSON(ErrorResponse{Error: "Workflow not found"})
	}

	return c.JSON(wf)
}

func (s *Server) updateWorkflow(c *fiber.Ctx) error {
	id := c.Params("id")

	var req CreateWorkflowRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(ErrorResponse{Error: "Invalid request body"})
	}

	// Validate YAML
	workflowDef, err := workflow.Parse(req.YAMLContent)
	if err != nil {
		return c.Status(400).JSON(ErrorResponse{Error: fmt.Sprintf("Invalid workflow YAML: %v", err)})
	}

	if err := workflow.Validate(workflowDef); err != nil {
		return c.Status(400).JSON(ErrorResponse{Error: fmt.Sprintf("Workflow validation failed: %v", err)})
	}

	repo := database.NewWorkflowRepo(s.db)
	wf, err := repo.GetByID(id)
	if err != nil {
		return c.Status(404).JSON(ErrorResponse{Error: "Workflow not found"})
	}

	wf.Name = req.Name
	wf.Description = req.Description
	wf.YAMLContent = req.YAMLContent
	wf.Enabled = req.Enabled

	if err := repo.Update(wf); err != nil {
		return c.Status(500).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.JSON(wf)
}

func (s *Server) toggleWorkflow(c *fiber.Ctx) error {
	id := c.Params("id")

	repo := database.NewWorkflowRepo(s.db)
	wf, err := repo.GetByID(id)
	if err != nil {
		return c.Status(404).JSON(ErrorResponse{Error: "Workflow not found"})
	}

	// Toggle enabled status
	wf.Enabled = !wf.Enabled

	if err := repo.Update(wf); err != nil {
		return c.Status(500).JSON(ErrorResponse{Error: err.Error()})
	}

	// Enable or disable watcher
	if wf.Enabled {
		if err := s.watcher.EnableWorkflow(id); err != nil {
			log.Printf("Warning: Failed to enable watcher for workflow %s: %v", id, err)
		}
	} else {
		if err := s.watcher.DisableWorkflow(id); err != nil {
			log.Printf("Warning: Failed to disable watcher for workflow %s: %v", id, err)
		}
	}

	return c.JSON(wf)
}

func (s *Server) deleteWorkflow(c *fiber.Ctx) error {
	id := c.Params("id")
	repo := database.NewWorkflowRepo(s.db)

	if err := repo.Delete(id); err != nil {
		return c.Status(404).JSON(ErrorResponse{Error: "Workflow not found"})
	}

	return c.JSON(SuccessResponse{Message: "Workflow deleted"})
}

func (s *Server) scanWorkflow(c *fiber.Ctx) error {
	id := c.Params("id")

	// Run scan in background
	go func() {
		result, err := s.watcher.ScanWorkflow(id)
		if err != nil {
			log.Printf("Scan failed for workflow %s: %v", id, err)
			return
		}
		log.Printf("Scan completed for workflow %s: %+v", id, result)
		// Tasks will be picked up by scheduler automatically
	}()

	return c.JSON(SuccessResponse{Message: "Scan started"})
}

func (s *Server) clearWorkflowIndex(c *fiber.Ctx) error {
	id := c.Params("id")

	// Verify workflow exists
	repo := database.NewWorkflowRepo(s.db)
	_, err := repo.GetByID(id)
	if err != nil {
		return c.Status(404).JSON(ErrorResponse{Error: "Workflow not found"})
	}

	// Delete all tasks for this workflow
	taskRepo := database.NewTaskRepo(s.db)
	if err := taskRepo.DeleteByWorkflow(id); err != nil {
		return c.Status(500).JSON(ErrorResponse{Error: fmt.Sprintf("Failed to clear tasks: %v", err)})
	}

	// Delete all files for this workflow
	fileRepo := database.NewFileRepo(s.db)
	if err := fileRepo.DeleteByWorkflow(id); err != nil {
		return c.Status(500).JSON(ErrorResponse{Error: fmt.Sprintf("Failed to clear files: %v", err)})
	}

	log.Printf("Cleared index for workflow %s", id)

	// Run scan in background
	go func() {
		result, err := s.watcher.ScanWorkflow(id)
		if err != nil {
			log.Printf("Scan failed for workflow %s: %v", id, err)
			return
		}
		log.Printf("Scan completed for workflow %s: %+v", id, result)
		// Tasks will be picked up by scheduler automatically
	}()

	return c.JSON(SuccessResponse{Message: "Index cleared and scan started"})
}

// Task handlers

func (s *Server) listTasks(c *fiber.Ctx) error {
	workflowID := c.Query("workflow_id", "")
	status := c.Query("status", "")
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	if limit > 1000 {
		limit = 1000
	}

	repo := database.NewTaskRepo(s.db)
	tasks, err := repo.List(workflowID, status, limit, offset)
	if err != nil {
		return c.Status(500).JSON(ErrorResponse{Error: err.Error()})
	}

	count, err := repo.Count(workflowID, status)
	if err != nil {
		return c.Status(500).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.JSON(fiber.Map{
		"tasks":  tasks,
		"total":  count,
		"limit":  limit,
		"offset": offset,
	})
}

func (s *Server) getTask(c *fiber.Ctx) error {
	id := c.Params("id")
	repo := database.NewTaskRepo(s.db)

	task, err := repo.GetByID(id)
	if err != nil {
		return c.Status(404).JSON(ErrorResponse{Error: "Task not found"})
	}

	return c.JSON(task)
}

func (s *Server) retryTask(c *fiber.Ctx) error {
	id := c.Params("id")
	repo := database.NewTaskRepo(s.db)

	task, err := repo.GetByID(id)
	if err != nil {
		return c.Status(404).JSON(ErrorResponse{Error: "Task not found"})
	}

	// Reset task status
	task.Status = models.TaskStatusPending
	task.ErrorMessage = ""
	task.StartedAt = nil
	task.CompletedAt = nil

	if err := repo.Update(task); err != nil {
		return c.Status(500).JSON(ErrorResponse{Error: err.Error()})
	}

	// Task will be picked up by scheduler automatically
	return c.JSON(SuccessResponse{Message: "Task reset to pending, will be executed by scheduler"})
}

func (s *Server) cancelTask(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := s.scheduler.CancelTask(id); err != nil {
		return c.Status(400).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.JSON(SuccessResponse{Message: "Task cancelled"})
}

func (s *Server) deleteTask(c *fiber.Ctx) error {
	id := c.Params("id")
	repo := database.NewTaskRepo(s.db)

	if err := repo.Delete(id); err != nil {
		return c.Status(404).JSON(ErrorResponse{Error: "Task not found"})
	}

	return c.JSON(SuccessResponse{Message: "Task deleted"})
}

func (s *Server) getTaskSteps(c *fiber.Ctx) error {
	id := c.Params("id")
	repo := database.NewTaskStepRepo(s.db)

	steps, err := repo.GetByTaskID(id)
	if err != nil {
		return c.Status(500).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.JSON(steps)
}

func (s *Server) tailTaskLog(c *fiber.Ctx) error {
	id := c.Params("id")
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	repo := database.NewTaskRepo(s.db)
	task, err := repo.GetByID(id)
	if err != nil {
		return c.Status(404).JSON(ErrorResponse{Error: "Task not found"})
	}

	// If task is completed or failed, return from database
	if task.Status == models.TaskStatusCompleted || task.Status == models.TaskStatusFailed || task.Status == models.TaskStatusCancelled {
		content := task.LogText
		if offset > 0 && offset < len(content) {
			content = content[offset:]
		}
		return c.JSON(fiber.Map{
			"content":   content,
			"offset":    len(task.LogText),
			"completed": true,
		})
	}

	// If task is running, try to read from log file
	logFilePath := filepath.Join(s.logDir, fmt.Sprintf("%s.log", id))
	if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
		return c.JSON(fiber.Map{
			"content":   "",
			"offset":    0,
			"completed": false,
		})
	}

	// Read log file
	data, err := os.ReadFile(logFilePath)
	if err != nil {
		return c.Status(500).JSON(ErrorResponse{Error: "Failed to read log file"})
	}

	content := string(data)
	if offset > 0 && offset < len(content) {
		content = content[offset:]
	}

	return c.JSON(fiber.Map{
		"content":   content,
		"offset":    len(data),
		"completed": false,
	})
}

// File handlers

func (s *Server) listFiles(c *fiber.Ctx) error {
	workflowID := c.Query("workflow_id", "")
	if workflowID == "" {
		return c.Status(400).JSON(ErrorResponse{Error: "workflow_id is required"})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	if limit > 1000 {
		limit = 1000
	}

	repo := database.NewFileRepo(s.db)
	files, err := repo.ListByWorkflow(workflowID, limit, offset)
	if err != nil {
		return c.Status(500).JSON(ErrorResponse{Error: err.Error()})
	}

	count, err := repo.CountByWorkflow(workflowID)
	if err != nil {
		return c.Status(500).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.JSON(fiber.Map{
		"files":  files,
		"total":  count,
		"limit":  limit,
		"offset": offset,
	})
}

// Scheduler/Monitoring handlers

func (s *Server) getSchedulerStats(c *fiber.Ctx) error {
	stats := s.scheduler.GetExecutorPoolStats()
	return c.JSON(stats)
}

func (s *Server) getExecutorStatus(c *fiber.Ctx) error {
	status := s.scheduler.GetExecutorStatus()
	return c.JSON(status)
}
