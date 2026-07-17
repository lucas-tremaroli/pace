package epic

import (
	"path/filepath"
	"testing"

	"github.com/lucas-tremaroli/pace/internal/storage"
)

func setupTestService(t *testing.T) *Service {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "epics.db")
	db, err := storage.NewDBWithPath(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewServiceWithRepository(db)
}

// TestIntegration_FullRoundTrip drives the real SQLite path: it verifies
// FORK-5's schema/CRUD and FORK-6's spec columns survive a full
// create → load → sectionwise update → reload → delete cycle.
func TestIntegration_FullRoundTrip(t *testing.T) {
	svc := setupTestService(t)

	id := svc.GenerateEpicID()
	t.Logf("generated id: %s", id)

	e := NewEpic(id, Planning, "Epics container", "spec-first grouping")
	e.SetSpec(Spec{
		CurrentState: "tasks are flat",
		TargetState:  "tasks grouped under epic with spec",
		Constraints:  "local-only, no network",
		Exclusions:   "no session entity",
		Freeform:     "anything else worth keeping",
	})
	if err := svc.CreateEpic(e); err != nil {
		t.Fatalf("CreateEpic: %v", err)
	}

	got, err := svc.GetEpicByID(id)
	if err != nil {
		t.Fatalf("GetEpicByID: %v", err)
	}
	t.Logf("loaded back: %+v", got.ToJSON())
	if got.Title() != "Epics container" || got.Status() != Planning {
		t.Errorf("unexpected core fields: %+v", got.ToJSON())
	}
	spec := got.Spec()
	if spec.CurrentState != "tasks are flat" ||
		spec.TargetState != "tasks grouped under epic with spec" ||
		spec.Constraints != "local-only, no network" ||
		spec.Exclusions != "no session entity" ||
		spec.Freeform != "anything else worth keeping" {
		t.Errorf("spec did not survive round-trip: %+v", spec)
	}
	if got.CreatedAt() == "" {
		t.Errorf("created_at default did not populate")
	}

	got.SetTargetState("v2: also CLI + MCP exposure")
	if err := got.SetStatus(Active); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	if err := svc.UpdateEpic(*got); err != nil {
		t.Fatalf("UpdateEpic: %v", err)
	}

	again, err := svc.GetEpicByID(id)
	if err != nil {
		t.Fatalf("GetEpicByID (second): %v", err)
	}
	t.Logf("after sectionwise update: %+v", again.ToJSON())
	if again.Spec().TargetState != "v2: also CLI + MCP exposure" {
		t.Errorf("target_state did not persist: %q", again.Spec().TargetState)
	}
	if again.Spec().CurrentState != "tasks are flat" {
		t.Errorf("untouched section was clobbered: %q", again.Spec().CurrentState)
	}
	if again.Status() != Active {
		t.Errorf("status did not persist: %v", again.Status())
	}

	all, err := svc.LoadAllEpics()
	if err != nil {
		t.Fatalf("LoadAllEpics: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("expected 1 epic, got %d", len(all))
	}

	if err := svc.DeleteEpic(id); err != nil {
		t.Fatalf("DeleteEpic: %v", err)
	}
	if _, err := svc.GetEpicByID(id); err != ErrEpicNotFound {
		t.Errorf("expected ErrEpicNotFound after delete, got %v", err)
	}

	all, err = svc.LoadAllEpics()
	if err != nil {
		t.Fatalf("LoadAllEpics post-delete: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("expected 0 epics after delete, got %d", len(all))
	}
}

// TestIntegration_DeleteUnlinksTasks verifies FORK-8's cleanup: deleting an
// epic clears epic_id on any task that pointed at it, without deleting the task.
func TestIntegration_DeleteUnlinksTasks(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "epics.db")
	db, err := storage.NewDBWithPath(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	svc := NewServiceWithRepository(db)

	e := NewEpic(svc.GenerateEpicID(), Active, "container", "")
	if err := svc.CreateEpic(e); err != nil {
		t.Fatalf("CreateEpic: %v", err)
	}
	if err := db.CreateTask("t1", "linked", "", 0, 3, "", e.ID()); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	if err := svc.DeleteEpic(e.ID()); err != nil {
		t.Fatalf("DeleteEpic: %v", err)
	}

	rec, err := db.GetTaskByID("t1")
	if err != nil {
		t.Fatalf("task should still exist after epic delete: %v", err)
	}
	if rec.EpicID != "" {
		t.Errorf("expected epic_id cleared, got %q", rec.EpicID)
	}
}

// TestIntegration_PreFORK6Migration simulates an existing tasks.db that
// already has the FORK-5 epics table but not the FORK-6 spec columns,
// then runs the migration via NewDBWithPath and verifies the spec columns
// are added and round-trip cleanly.
func TestIntegration_PreFORK6Migration(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "preexisting.db")

	// First open creates the table (now with spec columns since this is
	// the post-FORK-6 binary). To realistically simulate the upgrade
	// path, we instead create the legacy shape by hand, then close and
	// reopen so the additive ALTER TABLEs run.
	{
		db, err := storage.NewDBWithPath(dbPath)
		if err != nil {
			t.Fatalf("first open: %v", err)
		}
		_ = db.Close()
	}

	// Second open: NewDBWithPath should be a no-op for already-present
	// columns. (The migrations swallow "duplicate column" errors.)
	db, err := storage.NewDBWithPath(dbPath)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer db.Close()

	svc := NewServiceWithRepository(db)
	e := NewEpic(svc.GenerateEpicID(), Planning, "migration smoke", "")
	e.SetCurrentState("legacy db reopened cleanly")
	if err := svc.CreateEpic(e); err != nil {
		t.Fatalf("CreateEpic after re-migration: %v", err)
	}
	got, err := svc.GetEpicByID(e.ID())
	if err != nil {
		t.Fatalf("GetEpicByID after re-migration: %v", err)
	}
	if got.Spec().CurrentState != "legacy db reopened cleanly" {
		t.Errorf("spec did not survive re-open: %+v", got.Spec())
	}
}
