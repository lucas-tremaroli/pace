package task

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lucas-tremaroli/pace/internal/storage"
)

func setupTestService(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := storage.NewDBWithPath(dbPath)
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}

	// Set PACE_DIR so GetOrInitPrefix doesn't fail
	os.Setenv("PACE_DIR", dir)
	t.Cleanup(func() { os.Unsetenv("PACE_DIR") })

	prefix, err := GetOrInitPrefix(db)
	if err != nil {
		t.Fatalf("failed to get prefix: %v", err)
	}

	return &Service{db: db, prefix: prefix}
}

func TestLogEntry(t *testing.T) {
	svc := setupTestService(t)
	defer svc.Close()

	task := NewTaskComplete(svc.GenerateTaskID(), Todo, TypeTask, "Test task", "", 3, "")
	if err := svc.CreateTask(task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	if err := svc.LogEntry(task.ID(), "started working on this"); err != nil {
		t.Fatalf("LogEntry failed: %v", err)
	}

	logs, err := svc.GetTaskLogs(task.ID())
	if err != nil {
		t.Fatalf("GetTaskLogs failed: %v", err)
	}

	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].Message != "started working on this" {
		t.Errorf("expected message 'started working on this', got %q", logs[0].Message)
	}
	if logs[0].Type != "log" {
		t.Errorf("expected type 'log', got %q", logs[0].Type)
	}
}

func TestLogEntryTaskNotFound(t *testing.T) {
	svc := setupTestService(t)
	defer svc.Close()

	err := svc.LogEntry("nonexistent-id", "message")
	if err == nil {
		t.Fatal("expected error for nonexistent task, got nil")
	}
}

func TestCloseTask(t *testing.T) {
	svc := setupTestService(t)
	defer svc.Close()

	task := NewTaskComplete(svc.GenerateTaskID(), InProgress, TypeTask, "Test task", "", 3, "")
	if err := svc.CreateTask(task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	if err := svc.CloseTask(task.ID(), ""); err != nil {
		t.Fatalf("CloseTask failed: %v", err)
	}

	updated, err := svc.GetTaskByID(task.ID())
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	if updated.Status() != Done {
		t.Errorf("expected status Done, got %v", updated.Status())
	}

	// No outcome means no log
	logs, err := svc.GetTaskLogs(task.ID())
	if err != nil {
		t.Fatalf("GetTaskLogs failed: %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("expected 0 logs without outcome, got %d", len(logs))
	}
}

func TestCloseTaskWithOutcome(t *testing.T) {
	svc := setupTestService(t)
	defer svc.Close()

	task := NewTaskComplete(svc.GenerateTaskID(), Todo, TypeTask, "Test task", "", 3, "")
	if err := svc.CreateTask(task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	if err := svc.CloseTask(task.ID(), "completed successfully after refactoring"); err != nil {
		t.Fatalf("CloseTask failed: %v", err)
	}

	updated, err := svc.GetTaskByID(task.ID())
	if err != nil {
		t.Fatalf("GetTaskByID failed: %v", err)
	}
	if updated.Status() != Done {
		t.Errorf("expected status Done, got %v", updated.Status())
	}

	logs, err := svc.GetTaskLogs(task.ID())
	if err != nil {
		t.Fatalf("GetTaskLogs failed: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].Type != "outcome" {
		t.Errorf("expected type 'outcome', got %q", logs[0].Type)
	}
	if logs[0].Message != "completed successfully after refactoring" {
		t.Errorf("unexpected message: %q", logs[0].Message)
	}
}

func TestGetTaskLogsChronological(t *testing.T) {
	svc := setupTestService(t)
	defer svc.Close()

	task := NewTaskComplete(svc.GenerateTaskID(), Todo, TypeTask, "Test task", "", 3, "")
	if err := svc.CreateTask(task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	messages := []string{"first", "second", "third"}
	for _, msg := range messages {
		if err := svc.LogEntry(task.ID(), msg); err != nil {
			t.Fatalf("LogEntry failed: %v", err)
		}
	}

	logs, err := svc.GetTaskLogs(task.ID())
	if err != nil {
		t.Fatalf("GetTaskLogs failed: %v", err)
	}

	if len(logs) != 3 {
		t.Fatalf("expected 3 logs, got %d", len(logs))
	}
	for i, msg := range messages {
		if logs[i].Message != msg {
			t.Errorf("log[%d]: expected %q, got %q", i, msg, logs[i].Message)
		}
	}
}

func TestDeleteTaskCleansUpLogs(t *testing.T) {
	svc := setupTestService(t)
	defer svc.Close()

	task := NewTaskComplete(svc.GenerateTaskID(), Todo, TypeTask, "Test task", "", 3, "")
	if err := svc.CreateTask(task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	if err := svc.LogEntry(task.ID(), "some progress"); err != nil {
		t.Fatalf("LogEntry failed: %v", err)
	}

	if err := svc.DeleteTask(task.ID()); err != nil {
		t.Fatalf("DeleteTask failed: %v", err)
	}

	// Logs should be cleaned up — search shouldn't find them
	results, err := svc.SearchLogs("progress", 10)
	if err != nil {
		t.Fatalf("SearchLogs failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 search results after delete, got %d", len(results))
	}
}

func TestCloseTaskRemovesBlockingDeps(t *testing.T) {
	svc := setupTestService(t)
	defer svc.Close()

	blocker := NewTaskComplete(svc.GenerateTaskID(), InProgress, TypeTask, "Blocker", "", 3, "")
	blocked := NewTaskComplete(svc.GenerateTaskID(), Todo, TypeTask, "Blocked", "", 3, "")
	if err := svc.CreateTask(blocker); err != nil {
		t.Fatalf("failed to create blocker: %v", err)
	}
	if err := svc.CreateTask(blocked); err != nil {
		t.Fatalf("failed to create blocked: %v", err)
	}
	if err := svc.AddDependency(blocker.ID(), blocked.ID()); err != nil {
		t.Fatalf("failed to add dep: %v", err)
	}

	// Verify blocked task has a blocker
	task, _ := svc.GetTaskByID(blocked.ID())
	if len(task.BlockedBy()) != 1 {
		t.Fatalf("expected 1 blocker, got %d", len(task.BlockedBy()))
	}

	// Close the blocker
	if err := svc.CloseTask(blocker.ID(), "done"); err != nil {
		t.Fatalf("CloseTask failed: %v", err)
	}

	// Blocked task should no longer have blockers
	task, _ = svc.GetTaskByID(blocked.ID())
	if len(task.BlockedBy()) != 0 {
		t.Errorf("expected 0 blockers after close, got %d", len(task.BlockedBy()))
	}
}

func TestSearchLogs(t *testing.T) {
	svc := setupTestService(t)
	defer svc.Close()

	task := NewTaskComplete(svc.GenerateTaskID(), Todo, TypeTask, "Test task", "", 3, "")
	if err := svc.CreateTask(task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	if err := svc.LogEntry(task.ID(), "refactored the authentication module"); err != nil {
		t.Fatalf("LogEntry failed: %v", err)
	}
	if err := svc.LogEntry(task.ID(), "fixed a bug in the parser"); err != nil {
		t.Fatalf("LogEntry failed: %v", err)
	}

	results, err := svc.SearchLogs("authentication", 10)
	if err != nil {
		t.Fatalf("SearchLogs failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'authentication', got %d", len(results))
	}

	results, err = svc.SearchLogs("parser", 10)
	if err != nil {
		t.Fatalf("SearchLogs failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'parser', got %d", len(results))
	}
}
