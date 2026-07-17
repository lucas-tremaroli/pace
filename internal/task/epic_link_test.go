package task

import (
	"errors"
	"testing"

	"github.com/lucas-tremaroli/pace/internal/storage"
)

// mkEpic inserts an epic straight into the shared storage so tasks can link to it.
func mkEpic(t *testing.T, svc *Service, id string) {
	t.Helper()
	db, ok := svc.db.(*storage.DB)
	if !ok {
		t.Fatalf("expected *storage.DB backing the service")
	}
	if err := db.CreateEpic(storage.EpicRecord{ID: id, Title: id}); err != nil {
		t.Fatalf("failed to create epic %s: %v", id, err)
	}
}

func createEpicTask(t *testing.T, svc *Service, epicID string) string {
	t.Helper()
	tk := NewTaskComplete(svc.GenerateTaskID(), Todo, "t", "", 3, "")
	tk.SetEpicID(epicID)
	if err := svc.CreateTask(tk); err != nil {
		t.Fatalf("create task: %v", err)
	}
	return tk.ID()
}

func TestCreateTaskRejectsUnknownEpic(t *testing.T) {
	svc := setupTestService(t)
	defer svc.Close()

	tk := NewTaskComplete(svc.GenerateTaskID(), Todo, "t", "", 3, "")
	tk.SetEpicID("does-not-exist")
	if err := svc.CreateTask(tk); !errors.Is(err, ErrEpicNotFound) {
		t.Fatalf("expected ErrEpicNotFound, got %v", err)
	}
}

func TestAddDependencyWithinEpic(t *testing.T) {
	svc := setupTestService(t)
	defer svc.Close()

	mkEpic(t, svc, "E1")
	a := createEpicTask(t, svc, "E1")
	b := createEpicTask(t, svc, "E1")

	if err := svc.AddDependency(a, b); err != nil {
		t.Fatalf("same-epic dependency should be allowed, got %v", err)
	}
}

func TestAddDependencyRejectsCrossEpic(t *testing.T) {
	svc := setupTestService(t)
	defer svc.Close()

	mkEpic(t, svc, "E1")
	mkEpic(t, svc, "E2")
	a := createEpicTask(t, svc, "E1")
	b := createEpicTask(t, svc, "E2")

	if err := svc.AddDependency(a, b); !errors.Is(err, ErrCrossEpicDep) {
		t.Fatalf("expected ErrCrossEpicDep, got %v", err)
	}
}

func TestSetEpicAndFilter(t *testing.T) {
	svc := setupTestService(t)
	defer svc.Close()

	mkEpic(t, svc, "E1")
	inEpic := createEpicTask(t, svc, "E1")
	loose := createEpicTask(t, svc, "") // no epic

	// Reassigning a loose task to a real epic works and is reflected on load.
	if err := svc.SetEpic(loose, "E1"); err != nil {
		t.Fatalf("SetEpic: %v", err)
	}

	tasks, err := svc.LoadAllTasks()
	if err != nil {
		t.Fatalf("LoadAllTasks: %v", err)
	}
	filter := &TaskFilter{}
	e := "E1"
	filter.EpicID = &e
	got := filter.Apply(tasks)
	if len(got) != 2 {
		t.Fatalf("expected 2 tasks in epic E1, got %d", len(got))
	}
	_ = inEpic
}
