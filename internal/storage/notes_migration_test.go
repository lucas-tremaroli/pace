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
