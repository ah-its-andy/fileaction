package database

import (
	"os"
	"testing"

	"github.com/andi/fileaction/backend/models"
)

func setupTestDB(t *testing.T) *DB {
	// Create temporary database
	dbPath := "./test_fileaction.db"
	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Clean up after test
	t.Cleanup(func() {
		db.Close()
		os.Remove(dbPath)
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
	})

	return db
}

func TestWorkflowCRUD(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkflowRepo(db)

	// Create
	workflow := &models.Workflow{
		Name:        "test-workflow",
		Description: "Test description",
		YAMLContent: "name: test",
		Enabled:     true,
	}

	err := repo.Create(workflow)
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	if workflow.ID == "" {
		t.Error("Workflow ID should be set after creation")
	}

	// Read
	retrieved, err := repo.GetByID(workflow.ID)
	if err != nil {
		t.Fatalf("Failed to get workflow: %v", err)
	}

	if retrieved.Name != workflow.Name {
		t.Errorf("Expected name '%s', got '%s'", workflow.Name, retrieved.Name)
	}

	// Update
	retrieved.Description = "Updated description"
	err = repo.Update(retrieved)
	if err != nil {
		t.Fatalf("Failed to update workflow: %v", err)
	}

	updated, err := repo.GetByID(workflow.ID)
	if err != nil {
		t.Fatalf("Failed to get updated workflow: %v", err)
	}

	if updated.Description != "Updated description" {
		t.Errorf("Expected description 'Updated description', got '%s'", updated.Description)
	}

	// List
	workflows, err := repo.List()
	if err != nil {
		t.Fatalf("Failed to list workflows: %v", err)
	}

	if len(workflows) != 1 {
		t.Errorf("Expected 1 workflow, got %d", len(workflows))
	}

	// Delete
	err = repo.Delete(workflow.ID)
	if err != nil {
		t.Fatalf("Failed to delete workflow: %v", err)
	}

	_, err = repo.GetByID(workflow.ID)
	if err == nil {
		t.Error("Expected error when getting deleted workflow")
	}
}

func TestTaskCRUD(t *testing.T) {
	db := setupTestDB(t)
	workflowRepo := NewWorkflowRepo(db)
	fileRepo := NewFileRepo(db)
	taskRepo := NewTaskRepo(db)

	// Create workflow first
	workflow := &models.Workflow{
		Name:        "test-workflow",
		YAMLContent: "name: test",
		Enabled:     true,
	}
	err := workflowRepo.Create(workflow)
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	// Create file
	file := &models.File{
		WorkflowID: workflow.ID,
		FilePath:   "/test/file.jpg",
		FileMD5:    "abc123",
		FileSize:   1024,
	}
	err = fileRepo.Create(file)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Create task
	task := &models.Task{
		WorkflowID: workflow.ID,
		FileID:     file.ID,
		InputPath:  "/test/file.jpg",
		OutputPath: "/test/file.png",
		Status:     models.TaskStatusPending,
	}

	err = taskRepo.Create(task)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	if task.ID == "" {
		t.Error("Task ID should be set after creation")
	}

	// Read
	retrieved, err := taskRepo.GetByID(task.ID)
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}

	if retrieved.Status != models.TaskStatusPending {
		t.Errorf("Expected status 'pending', got '%s'", retrieved.Status)
	}

	// Update status
	err = taskRepo.UpdateStatus(task.ID, models.TaskStatusCompleted)
	if err != nil {
		t.Fatalf("Failed to update task status: %v", err)
	}

	updated, err := taskRepo.GetByID(task.ID)
	if err != nil {
		t.Fatalf("Failed to get updated task: %v", err)
	}

	if updated.Status != models.TaskStatusCompleted {
		t.Errorf("Expected status 'completed', got '%s'", updated.Status)
	}

	// List
	tasks, err := taskRepo.List("", "", 10, 0)
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}

	if len(tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(tasks))
	}

	// Count
	count, err := taskRepo.Count("", "")
	if err != nil {
		t.Fatalf("Failed to count tasks: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}
}

func TestFileCRUD(t *testing.T) {
	db := setupTestDB(t)
	workflowRepo := NewWorkflowRepo(db)
	fileRepo := NewFileRepo(db)

	// Create workflow first
	workflow := &models.Workflow{
		Name:        "test-workflow",
		YAMLContent: "name: test",
		Enabled:     true,
	}
	err := workflowRepo.Create(workflow)
	if err != nil {
		t.Fatalf("Failed to create workflow: %v", err)
	}

	// Create file
	file := &models.File{
		WorkflowID: workflow.ID,
		FilePath:   "/test/file.jpg",
		FileMD5:    "abc123",
		FileSize:   1024,
	}

	err = fileRepo.Create(file)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Get by workflow and path
	retrieved, err := fileRepo.GetByWorkflowAndPath(workflow.ID, "/test/file.jpg")
	if err != nil {
		t.Fatalf("Failed to get file: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected file to be found")
	}

	if retrieved.FileMD5 != "abc123" {
		t.Errorf("Expected MD5 'abc123', got '%s'", retrieved.FileMD5)
	}

	// Update
	retrieved.FileMD5 = "def456"
	err = fileRepo.Update(retrieved)
	if err != nil {
		t.Fatalf("Failed to update file: %v", err)
	}

	updated, err := fileRepo.GetByWorkflowAndPath(workflow.ID, "/test/file.jpg")
	if err != nil {
		t.Fatalf("Failed to get updated file: %v", err)
	}

	if updated.FileMD5 != "def456" {
		t.Errorf("Expected MD5 'def456', got '%s'", updated.FileMD5)
	}

	// List
	files, err := fileRepo.ListByWorkflow(workflow.ID, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}

	// Count
	count, err := fileRepo.CountByWorkflow(workflow.ID)
	if err != nil {
		t.Fatalf("Failed to count files: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}
}
