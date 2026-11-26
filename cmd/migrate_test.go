package cmd

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/lucas-tremaroli/pace/v2/internal/storage"
)

func newStore(t *testing.T) (string, *storage.DB) {
	t.Helper()
	dir := t.TempDir()
	db, err := storage.NewDBWithPath(filepath.Join(dir, "tasks.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return dir, db
}

func mustEpic(t *testing.T, db *storage.DB, id, title string) {
	t.Helper()
	if err := db.CreateEpic(storage.EpicRecord{ID: id, Title: title, Labels: "[]"}); err != nil {
		t.Fatal(err)
	}
}

// Epic IDs are per-store, so the destination can hold an unrelated epic under
// the same ID. It must be left alone and the task must follow its own epic.
func TestMigrateTasksRemapsConflictingEpicID(t *testing.T) {
	srcDir, src := newStore(t)
	dstDir, dst := newStore(t)
	mustEpic(t, src, "epic-aaa", "source epic")
	mustEpic(t, dst, "epic-aaa", "unrelated destination epic")
	if err := src.CreateTask("t-1", "task", "", 0, 2, "", "epic-aaa"); err != nil {
		t.Fatal(err)
	}

	result, err := migrateTasks(srcDir, dstDir, false)
	if err != nil {
		t.Fatal(err)
	}

	task, err := dst.GetTaskByID("t-1")
	if err != nil {
		t.Fatal(err)
	}
	if task.EpicID == "epic-aaa" || task.EpicID == "" {
		t.Fatalf("task attached to the unrelated epic or unlinked: %q", task.EpicID)
	}
	moved, err := dst.GetEpicByID(task.EpicID)
	if err != nil || moved.Title != "source epic" {
		t.Errorf("task's epic should be the copied source epic, got %+v (err %v)", moved, err)
	}
	if orig, _ := dst.GetEpicByID("epic-aaa"); orig.Title != "unrelated destination epic" {
		t.Errorf("existing destination epic was modified: %+v", orig)
	}
	if remapped, _ := result["epics_remapped"].(map[string]string); remapped["epic-aaa"] != task.EpicID {
		t.Errorf("remap not reported: %v", result["epics_remapped"])
	}
}

func TestMigrateTasksReusesIdenticalEpic(t *testing.T) {
	srcDir, src := newStore(t)
	dstDir, dst := newStore(t)
	mustEpic(t, src, "epic-aaa", "same")
	mustEpic(t, dst, "epic-aaa", "same")
	if err := src.CreateTask("t-1", "task", "", 0, 2, "", "epic-aaa"); err != nil {
		t.Fatal(err)
	}

	if _, err := migrateTasks(srcDir, dstDir, false); err != nil {
		t.Fatal(err)
	}
	if task, _ := dst.GetTaskByID("t-1"); task.EpicID != "epic-aaa" {
		t.Errorf("identical epic should be reused, got %q", task.EpicID)
	}
	if all, _ := dst.GetAllEpics(); len(all) != 1 {
		t.Errorf("no duplicate epic expected, have %d", len(all))
	}
}

// A preview must not convert legacy notes or move anything.
func TestMigrateTasksDryRunLeavesLegacyStoreUntouched(t *testing.T) {
	srcDir, src := newStore(t)
	dstDir, _ := newStore(t)
	if err := src.CreateTask("t-1", "task", "", 0, 2, "", ""); err != nil {
		t.Fatal(err)
	}
	raw, err := sql.Open("sqlite", filepath.Join(srcDir, "tasks.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	if _, err := raw.Exec(`CREATE TABLE task_notes (task_id VARCHAR NOT NULL, note_filename VARCHAR NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(srcDir, "notes"), 0755); err != nil {
		t.Fatal(err)
	}
	notePath := filepath.Join(srcDir, "notes", "plan.md")
	if err := os.WriteFile(notePath, []byte("plan"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := migrateTasks(srcDir, dstDir, true)
	if err != nil {
		t.Fatal(err)
	}
	if result["migrated"] != 1 {
		t.Errorf("preview should still count the task, got %v", result["migrated"])
	}
	if _, err := os.Stat(notePath); err != nil {
		t.Errorf("dry run deleted the note file: %v", err)
	}
	var n int
	if err := raw.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name = 'task_notes'`).Scan(&n); err != nil || n != 1 {
		t.Errorf("dry run dropped task_notes (count %d, err %v)", n, err)
	}
	if _, err := src.GetTaskByID("t-1"); err != nil {
		t.Errorf("dry run moved the task: %v", err)
	}
}
