package storage

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// migrateNotesToEpics folds the legacy notes feature into epics: every
// <paceDir>/notes/*.md becomes a planning epic (body in the freeform spec),
// task_notes links become tasks.epic_id (first link wins), then task_notes is
// dropped and the note files are removed.
//
// It is safe to retry. Each imported filename is recorded in migrated_notes in
// the same transaction as its epic, so if removing the files fails (or the
// process dies) after the commit, the next open only finishes the cleanup
// instead of importing the notes a second time.
func (db *DB) migrateNotesToEpics(paceDir string) error {
	files, err := listNoteFiles(filepath.Join(paceDir, "notes"))
	if err != nil {
		return err
	}

	legacyTable, err := db.tableExists("task_notes")
	if err != nil {
		return err
	}
	imported, err := db.importedNotes()
	if err != nil {
		return err
	}

	var pending []string
	for _, path := range files {
		if _, done := imported[filepath.Base(path)]; !done {
			pending = append(pending, path)
		}
	}
	if len(pending) == 0 && !legacyTable {
		return removeFiles(files) // leftovers from an interrupted run, if any
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`CREATE TABLE IF NOT EXISTS migrated_notes (filename VARCHAR PRIMARY KEY, epic_id VARCHAR NOT NULL)`); err != nil {
		return err
	}
	epicByFile := imported
	for _, path := range pending {
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		summary, labels, body := splitNoteFrontmatter(string(raw))
		filename := filepath.Base(path)
		id, err := newEpicID(tx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(
			`INSERT INTO epics (id, title, summary, status, freeform, labels) VALUES (?, ?, ?, 0, ?, ?)`,
			id, strings.TrimSuffix(filename, ".md"), summary, body, labelsJSON(labels),
		); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO migrated_notes (filename, epic_id) VALUES (?, ?)`, filename, id); err != nil {
			return err
		}
		epicByFile[filename] = id
	}

	if legacyTable {
		rows, err := tx.Query(`SELECT task_id, note_filename FROM task_notes ORDER BY task_id, note_filename`)
		if err != nil {
			return err
		}
		type link struct{ task, epic string }
		var links []link
		for rows.Next() {
			var task, file string
			if err := rows.Scan(&task, &file); err != nil {
				rows.Close()
				return err
			}
			if epic, ok := epicByFile[file]; ok {
				links = append(links, link{task, epic})
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
		for _, l := range links {
			if _, err := tx.Exec(`UPDATE tasks SET epic_id = ? WHERE id = ? AND epic_id = ''`, l.epic, l.task); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(`DROP TABLE task_notes`); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	// Files go only after commit so a failed migration can be retried.
	return removeFiles(files)
}

// listNoteFiles returns the .md files directly inside notesDir, sorted. It
// reads the directory instead of globbing, so a project path containing
// glob characters (e.g. "/work/app[1]/.pace") works, and any read error other
// than "no notes directory" stops the migration before it drops anything.
func listNoteFiles(notesDir string) ([]string, error) {
	entries, err := os.ReadDir(notesDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			files = append(files, filepath.Join(notesDir, e.Name()))
		}
	}
	sort.Strings(files)
	return files, nil
}

func removeFiles(paths []string) error {
	for _, p := range paths {
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func (db *DB) tableExists(name string) (bool, error) {
	var found string
	err := db.conn.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name = ?`, name).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// importedNotes returns filename -> epic ID for notes already imported.
func (db *DB) importedNotes() (map[string]string, error) {
	out := map[string]string{}
	has, err := db.tableExists("migrated_notes")
	if err != nil || !has {
		return out, err
	}
	rows, err := db.conn.Query(`SELECT filename, epic_id FROM migrated_notes`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var file, epic string
		if err := rows.Scan(&file, &epic); err != nil {
			return nil, err
		}
		out[file] = epic
	}
	return out, rows.Err()
}

// splitNoteFrontmatter returns the frontmatter description and labels and the
// remaining body. Content without valid frontmatter is returned whole.
func splitNoteFrontmatter(content string) (description string, labels []string, body string) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	rest, ok := strings.CutPrefix(content, "---\n")
	if !ok {
		return "", nil, content
	}
	yml, after, ok := strings.Cut(rest, "\n---")
	if !ok {
		return "", nil, content
	}
	var fm struct {
		Description string   `yaml:"description"`
		Labels      []string `yaml:"labels"`
	}
	if err := yaml.Unmarshal([]byte(yml), &fm); err != nil {
		return "", nil, content
	}
	return fm.Description, fm.Labels, strings.TrimPrefix(strings.TrimPrefix(after, "\n"), "\n")
}

func labelsJSON(labels []string) string {
	if len(labels) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(labels)
	return string(b)
}

// newEpicID mirrors epic.GenerateID (epic imports storage, so it can't be
// called from here) and retries on collision within the migration tx.
func newEpicID(tx *sql.Tx) (string, error) {
	for {
		b := make([]byte, 2)
		if _, err := rand.Read(b); err != nil {
			return "", err
		}
		id := "epic-" + hex.EncodeToString(b)[:3]
		var n int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM epics WHERE id = ?`, id).Scan(&n); err != nil {
			return "", err
		}
		if n == 0 {
			return id, nil
		}
	}
}
