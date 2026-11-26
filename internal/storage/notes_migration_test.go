package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateNotesToEpics(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tasks.db")

	db, err := NewDBWithPath(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"t-1", "t-2"} {
		if err := db.CreateTask(id, id, "", 0, 2, "", ""); err != nil {
			t.Fatal(err)
		}
	}
	// Recreate the legacy shape: notes on disk plus the task_notes table.
	if _, err := db.conn.Exec(`CREATE TABLE task_notes (task_id VARCHAR NOT NULL, note_filename VARCHAR NOT NULL, PRIMARY KEY (task_id, note_filename))`); err != nil {
		t.Fatal(err)
	}
	notesDir := filepath.Join(dir, "notes")
	if err := os.MkdirAll(notesDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeNote := func(name, content string) {
		if err := os.WriteFile(filepath.Join(notesDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	writeNote("auth.md", "---\ndescription: \"Login flow\"\nlabels: [design, api]\n---\n\n# Auth\n\nUse sessions.\n")
	writeNote("plain.md", "no frontmatter\n")
	// t-1 links to both notes (first by filename wins), t-2 to plain.
	for _, l := range [][2]string{{"t-1", "plain.md"}, {"t-1", "auth.md"}, {"t-2", "plain.md"}} {
		if _, err := db.conn.Exec(`INSERT INTO task_notes VALUES (?, ?)`, l[0], l[1]); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()

	db, err = NewDBWithPath(dbPath)
	if err != nil {
		t.Fatalf("reopen should migrate: %v", err)
	}
	defer db.Close()

	epics, err := db.GetAllEpics()
	if err != nil {
		t.Fatal(err)
	}
	byTitle := map[string]EpicRecord{}
	for _, e := range epics {
		byTitle[e.Title] = e
	}
	auth, ok := byTitle["auth"]
	if !ok || len(epics) != 2 {
		t.Fatalf("want epics auth+plain, got %+v", epics)
	}
	if auth.Summary != "Login flow" || auth.Freeform != "# Auth\n\nUse sessions.\n" {
		t.Errorf("auth epic wrong: %+v", auth)
	}
	if auth.Labels != `["design","api"]` {
		t.Errorf("labels not carried over: %s", auth.Labels)
	}
	if byTitle["plain"].Freeform != "no frontmatter\n" {
		t.Errorf("plain epic wrong: %+v", byTitle["plain"])
	}

	t1, _ := db.GetTaskByID("t-1")
	t2, _ := db.GetTaskByID("t-2")
	if t1.EpicID != auth.ID {
		t.Errorf("t-1 should land in auth (first link), got %q", t1.EpicID)
	}
	if t2.EpicID != byTitle["plain"].ID {
		t.Errorf("t-2 should land in plain, got %q", t2.EpicID)
	}

	if m, _ := filepath.Glob(filepath.Join(notesDir, "*.md")); len(m) != 0 {
		t.Errorf("note files should be removed, got %v", m)
	}
	// Idempotent: a third open adds nothing.
	db.Close()
	db, err = NewDBWithPath(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if again, _ := db.GetAllEpics(); len(again) != 2 {
		t.Errorf("migration should be idempotent, got %d epics", len(again))
	}
}

// legacyStore builds a store in dir with one task linked to one note.
func legacyStore(t *testing.T, dir string) string {
	t.Helper()
	dbPath := filepath.Join(dir, "tasks.db")
	db, err := NewDBWithPath(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.CreateTask("t-1", "task", "", 0, 2, "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := db.conn.Exec(`CREATE TABLE task_notes (task_id VARCHAR NOT NULL, note_filename VARCHAR NOT NULL, PRIMARY KEY (task_id, note_filename))`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.conn.Exec(`INSERT INTO task_notes VALUES ('t-1', 'plan.md')`); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "notes"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes", "plan.md"), []byte("the plan\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return dbPath
}

func TestMigrateNotesInPathWithGlobCharacters(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "app[1]", ".pace")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	dbPath := legacyStore(t, dir)

	db, err := NewDBWithPath(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	epics, _ := db.GetAllEpics()
	if len(epics) != 1 || epics[0].Title != "plan" {
		t.Fatalf("note in a glob-character path was not imported: %+v", epics)
	}
	if task, _ := db.GetTaskByID("t-1"); task.EpicID != epics[0].ID {
		t.Errorf("link lost: task epic = %q", task.EpicID)
	}
}

// If removing the note files fails after the commit, the next open must only
// finish the cleanup, not import the same notes again.
func TestMigrateNotesRetryDoesNotDuplicate(t *testing.T) {
	dir := t.TempDir()
	dbPath := legacyStore(t, dir)

	db, err := NewDBWithPath(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	// Simulate the interrupted cleanup: the note file is still on disk.
	if err := os.WriteFile(filepath.Join(dir, "notes", "plan.md"), []byte("the plan\n"), 0644); err != nil {
		t.Fatal(err)
	}

	db, err = NewDBWithPath(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if epics, _ := db.GetAllEpics(); len(epics) != 1 {
		t.Errorf("retry duplicated the epic: have %d", len(epics))
	}
	if m, _ := filepath.Glob(filepath.Join(dir, "notes", "*.md")); len(m) != 0 {
		t.Errorf("retry should finish cleanup, left %v", m)
	}
}

func TestNewDBWithPathNoMigrationLeavesStoreUntouched(t *testing.T) {
	dir := t.TempDir()
	dbPath := legacyStore(t, dir)

	db, err := NewDBWithPathNoMigration(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if epics, _ := db.GetAllEpics(); len(epics) != 0 {
		t.Errorf("no-migration open created epics: %+v", epics)
	}
	if has, _ := db.tableExists("task_notes"); !has {
		t.Error("task_notes should still exist")
	}
	if _, err := os.Stat(filepath.Join(dir, "notes", "plan.md")); err != nil {
		t.Errorf("note file should be untouched: %v", err)
	}
}
