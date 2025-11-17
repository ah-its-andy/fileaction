package database

import (
	"github.com/andi/fileaction/backend/models"
)

// ToWorkflow converts WorkflowModel to models.Workflow
func (m *WorkflowModel) ToWorkflow() *models.Workflow {
	return &models.Workflow{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		YAMLContent: m.YAMLContent,
		Enabled:     m.Enabled,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// FromWorkflow converts models.Workflow to WorkflowModel
func FromWorkflow(w *models.Workflow) *WorkflowModel {
	return &WorkflowModel{
		ID:          w.ID,
		Name:        w.Name,
		Description: w.Description,
		YAMLContent: w.YAMLContent,
		Enabled:     w.Enabled,
		CreatedAt:   w.CreatedAt,
		UpdatedAt:   w.UpdatedAt,
	}
}

// ToFile converts FileModel to models.File
func (m *FileModel) ToFile() *models.File {
	return &models.File{
		ID:            m.ID,
		WorkflowID:    m.WorkflowID,
		FilePath:      m.FilePath,
		FileMD5:       m.FileMD5,
		FileSize:      m.FileSize,
		LastScannedAt: m.LastScannedAt,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}

// FromFile converts models.File to FileModel
func FromFile(f *models.File) *FileModel {
	return &FileModel{
		ID:            f.ID,
		WorkflowID:    f.WorkflowID,
		FilePath:      f.FilePath,
		FileMD5:       f.FileMD5,
		FileSize:      f.FileSize,
		LastScannedAt: f.LastScannedAt,
		CreatedAt:     f.CreatedAt,
		UpdatedAt:     f.UpdatedAt,
	}
}

// ToTask converts TaskModel to models.Task
func (m *TaskModel) ToTask() *models.Task {
	return &models.Task{
		ID:           m.ID,
		WorkflowID:   m.WorkflowID,
		FileID:       m.FileID,
		InputPath:    m.InputPath,
		OutputPath:   m.OutputPath,
		Status:       m.Status,
		LogText:      m.LogText,
		ErrorMessage: m.ErrorMessage,
		StartedAt:    m.StartedAt,
		CompletedAt:  m.CompletedAt,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

// FromTask converts models.Task to TaskModel
func FromTask(t *models.Task) *TaskModel {
	return &TaskModel{
		ID:           t.ID,
		WorkflowID:   t.WorkflowID,
		FileID:       t.FileID,
		InputPath:    t.InputPath,
		OutputPath:   t.OutputPath,
		Status:       t.Status,
		LogText:      t.LogText,
		ErrorMessage: t.ErrorMessage,
		StartedAt:    t.StartedAt,
		CompletedAt:  t.CompletedAt,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
}

// ToTaskStep converts TaskStepModel to models.TaskStep
func (m *TaskStepModel) ToTaskStep() *models.TaskStep {
	return &models.TaskStep{
		ID:          m.ID,
		TaskID:      m.TaskID,
		Name:        m.Name,
		Command:     m.Command,
		Status:      m.Status,
		ExitCode:    m.ExitCode,
		Stdout:      m.Stdout,
		Stderr:      m.Stderr,
		StartedAt:   m.StartedAt,
		CompletedAt: m.CompletedAt,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// FromTaskStep converts models.TaskStep to TaskStepModel
func FromTaskStep(ts *models.TaskStep) *TaskStepModel {
	return &TaskStepModel{
		ID:          ts.ID,
		TaskID:      ts.TaskID,
		Name:        ts.Name,
		Command:     ts.Command,
		Status:      ts.Status,
		ExitCode:    ts.ExitCode,
		Stdout:      ts.Stdout,
		Stderr:      ts.Stderr,
		StartedAt:   ts.StartedAt,
		CompletedAt: ts.CompletedAt,
		CreatedAt:   ts.CreatedAt,
		UpdatedAt:   ts.UpdatedAt,
	}
}
