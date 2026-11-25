package storage

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
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
// dropped and the note files are removed. Idempotent: a no-op once done.
func (db *DB) migrateNotesToEpics(paceDir string) error {
	notesDir := filepath.Join(paceDir, "notes")
	files, err := filepath.Glob(filepath.Join(notesDir, "*.md"))
	if err != nil {
		return err
	}
	sort.Strings(files)

	var legacyTable string
	err = db.conn.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='task_notes'`).Scan(&legacyTable)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if len(files) == 0 && legacyTable == "" {
		return nil
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	epicByFile := make(map[string]string, len(files))
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		summary, body := splitNoteFrontmatter(string(raw))
		filename := filepath.Base(path)
		id, err := newEpicID(tx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(
			`INSERT INTO epics (id, title, summary, status, freeform) VALUES (?, ?, ?, 0, ?)`,
			id, strings.TrimSuffix(filename, ".md"), summary, body,
		); err != nil {
			return err
		}
		epicByFile[filename] = id
	}

	if legacyTable != "" {
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
	for _, path := range files {
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	return nil
}

// splitNoteFrontmatter returns the frontmatter description and the remaining
// body. Content without valid frontmatter is returned whole.
func splitNoteFrontmatter(content string) (description, body string) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	rest, ok := strings.CutPrefix(content, "---\n")
	if !ok {
		return "", content
	}
	yml, after, ok := strings.Cut(rest, "\n---")
	if !ok {
		return "", content
	}
	var fm struct {
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal([]byte(yml), &fm); err != nil {
		return "", content
	}
	return fm.Description, strings.TrimPrefix(strings.TrimPrefix(after, "\n"), "\n")
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
