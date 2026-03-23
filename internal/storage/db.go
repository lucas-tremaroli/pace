package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

type TaskRecord struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status   int    `json:"status"`
	Priority int    `json:"priority"`
	Link     string `json:"link"`
}

type LogRecord struct {
	ID        int    `json:"id"`
	TaskID    string `json:"task_id,omitempty"`
	Message   string `json:"message"`
	Type      string `json:"type"`
	CreatedAt string `json:"created_at"`
}

func NewDB() (*DB, error) {
	dbPath, err := getDBPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get database path: %w", err)
	}

	return NewDBWithPath(dbPath)
}

// NewDBWithPath creates a new DB instance with a specific database path
func NewDBWithPath(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.createTables(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

// GetPaceConfigDir returns the pace configuration directory path
func GetPaceConfigDir() (string, error) {
	resolved, err := ResolvePaceDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(resolved.Path, 0755); err != nil {
		return "", err
	}

	return resolved.Path, nil
}

func getDBPath() (string, error) {
	paceDir, err := GetPaceConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(paceDir, "tasks.db"), nil
}

func (db *DB) createTables() error {
	query := `
		CREATE TABLE IF NOT EXISTS tasks (
			id VARCHAR PRIMARY KEY,
			title VARCHAR NOT NULL,
			description VARCHAR,
			status INTEGER NOT NULL,
			priority INTEGER NOT NULL DEFAULT 0
		);
	`
	if _, err := db.conn.Exec(query); err != nil {
		return err
	}

	// Migration: add priority column if it doesn't exist
	_, err := db.conn.Exec(`ALTER TABLE tasks ADD COLUMN priority INTEGER NOT NULL DEFAULT 0`)
	// Ignore error if column already exists
	_ = err

	// Migration: add link column if it doesn't exist
	_, err = db.conn.Exec(`ALTER TABLE tasks ADD COLUMN link VARCHAR DEFAULT ''`)
	// Ignore error if column already exists
	_ = err

	// Migration: consolidate P4 into P3 (4-level → 3-level priority)
	_, _ = db.conn.Exec(`UPDATE tasks SET priority = 3 WHERE priority = 4`)

	// Create task_dependencies table for blocking relationships
	depQuery := `
		CREATE TABLE IF NOT EXISTS task_dependencies (
			blocker_id VARCHAR NOT NULL,
			blocked_id VARCHAR NOT NULL,
			PRIMARY KEY (blocker_id, blocked_id),
			FOREIGN KEY (blocker_id) REFERENCES tasks(id) ON DELETE CASCADE,
			FOREIGN KEY (blocked_id) REFERENCES tasks(id) ON DELETE CASCADE
		);
	`
	if _, err := db.conn.Exec(depQuery); err != nil {
		return err
	}

	// Create config table for settings like id_prefix
	configQuery := `
		CREATE TABLE IF NOT EXISTS config (
			key VARCHAR PRIMARY KEY,
			value VARCHAR NOT NULL
		);
	`
	if _, err := db.conn.Exec(configQuery); err != nil {
		return err
	}

	// Create task_labels table for label associations
	labelsQuery := `
		CREATE TABLE IF NOT EXISTS task_labels (
			task_id VARCHAR NOT NULL,
			label VARCHAR NOT NULL,
			PRIMARY KEY (task_id, label),
			FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
		);
	`
	if _, err := db.conn.Exec(labelsQuery); err != nil {
		return err
	}

	// Create task_logs table for progress/outcome logging
	logsQuery := `
		CREATE TABLE IF NOT EXISTS task_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id VARCHAR,
			message TEXT NOT NULL,
			type VARCHAR NOT NULL DEFAULT 'log',
			created_at DATETIME NOT NULL DEFAULT (datetime('now'))
		);
	`
	if _, err := db.conn.Exec(logsQuery); err != nil {
		return err
	}

	// Create FTS5 virtual table for full-text search on logs
	ftsQuery := `
		CREATE VIRTUAL TABLE IF NOT EXISTS task_logs_fts USING fts5(
			message, content='task_logs', content_rowid='id'
		);
	`
	if _, err := db.conn.Exec(ftsQuery); err != nil {
		return err
	}

	// Create task_notes join table for note-task associations
	taskNotesQuery := `
		CREATE TABLE IF NOT EXISTS task_notes (
			task_id VARCHAR NOT NULL,
			note_filename VARCHAR NOT NULL,
			PRIMARY KEY (task_id, note_filename),
			FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
		);
	`
	if _, err := db.conn.Exec(taskNotesQuery); err != nil {
		return err
	}

	return nil
}

// GetConfig retrieves a config value by key
func (db *DB) GetConfig(key string) (string, error) {
	query := `SELECT value FROM config WHERE key = ?`
	row := db.conn.QueryRow(query, key)
	var value string
	err := row.Scan(&value)
	return value, err
}

// SetConfig sets a config value
func (db *DB) SetConfig(key, value string) error {
	query := `INSERT OR REPLACE INTO config (key, value) VALUES (?, ?)`
	_, err := db.conn.Exec(query, key, value)
	return err
}

// DeleteConfig removes a config value by key
func (db *DB) DeleteConfig(key string) error {
	query := `DELETE FROM config WHERE key = ?`
	result, err := db.conn.Exec(query, key)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// GetAllConfig retrieves all config key-value pairs
func (db *DB) GetAllConfig() (map[string]string, error) {
	query := `SELECT key, value FROM config ORDER BY key`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	config := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		config[key] = value
	}
	return config, rows.Err()
}

func (db *DB) CreateTask(id, title, description string, status, priority int, link string) error {
	query := `INSERT INTO tasks (id, title, description, status, priority, link) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := db.conn.Exec(query, id, title, description, status, priority, link)
	return err
}

func (db *DB) GetAllTasks() ([]TaskRecord, error) {
	query := `SELECT id, title, description, status, priority, COALESCE(link, '') FROM tasks ORDER BY priority DESC, title`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []TaskRecord
	for rows.Next() {
		var task TaskRecord
		err := rows.Scan(&task.ID, &task.Title, &task.Description, &task.Status, &task.Priority, &task.Link)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

func (db *DB) UpdateTask(id, title, description string, status, priority int, link string) error {
	query := `UPDATE tasks SET title = ?, description = ?, status = ?, priority = ?, link = ? WHERE id = ?`
	_, err := db.conn.Exec(query, title, description, status, priority, link, id)
	return err
}

func (db *DB) DeleteTask(id string) error {
	query := `DELETE FROM tasks WHERE id = ?`
	_, err := db.conn.Exec(query, id)
	return err
}

func (db *DB) GetTaskByID(id string) (*TaskRecord, error) {
	query := `SELECT id, title, description, status, priority, COALESCE(link, '') FROM tasks WHERE id = ?`
	row := db.conn.QueryRow(query, id)

	var task TaskRecord
	err := row.Scan(&task.ID, &task.Title, &task.Description, &task.Status, &task.Priority, &task.Link)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// AddDependency creates a blocking relationship where blocker blocks blocked
func (db *DB) AddDependency(blockerID, blockedID string) error {
	query := `INSERT OR IGNORE INTO task_dependencies (blocker_id, blocked_id) VALUES (?, ?)`
	_, err := db.conn.Exec(query, blockerID, blockedID)
	return err
}

// RemoveDependency removes a blocking relationship
func (db *DB) RemoveDependency(blockerID, blockedID string) error {
	query := `DELETE FROM task_dependencies WHERE blocker_id = ? AND blocked_id = ?`
	_, err := db.conn.Exec(query, blockerID, blockedID)
	return err
}

// GetBlockers returns the IDs of tasks that block the given task
func (db *DB) GetBlockers(taskID string) ([]string, error) {
	query := `SELECT blocker_id FROM task_dependencies WHERE blocked_id = ?`
	rows, err := db.conn.Query(query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blockers []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		blockers = append(blockers, id)
	}
	return blockers, rows.Err()
}

// GetBlocking returns the IDs of tasks that the given task blocks
func (db *DB) GetBlocking(taskID string) ([]string, error) {
	query := `SELECT blocked_id FROM task_dependencies WHERE blocker_id = ?`
	rows, err := db.conn.Query(query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blocking []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		blocking = append(blocking, id)
	}
	return blocking, rows.Err()
}

// GetAllDependencies returns all dependency relationships
func (db *DB) GetAllDependencies() (map[string][]string, map[string][]string, error) {
	query := `SELECT blocker_id, blocked_id FROM task_dependencies`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	blockedBy := make(map[string][]string) // task -> tasks that block it
	blocks := make(map[string][]string)    // task -> tasks it blocks

	for rows.Next() {
		var blockerID, blockedID string
		if err := rows.Scan(&blockerID, &blockedID); err != nil {
			return nil, nil, err
		}
		blockedBy[blockedID] = append(blockedBy[blockedID], blockerID)
		blocks[blockerID] = append(blocks[blockerID], blockedID)
	}

	return blockedBy, blocks, rows.Err()
}

// RemoveBlockingDeps removes dependencies where the given task is the blocker
func (db *DB) RemoveBlockingDeps(taskID string) error {
	query := `DELETE FROM task_dependencies WHERE blocker_id = ?`
	_, err := db.conn.Exec(query, taskID)
	return err
}

// RemoveAllDependencies removes all dependencies for a task (both directions)
func (db *DB) RemoveAllDependencies(taskID string) error {
	query := `DELETE FROM task_dependencies WHERE blocker_id = ? OR blocked_id = ?`
	_, err := db.conn.Exec(query, taskID, taskID)
	return err
}

// AddLabel adds a label to a task
func (db *DB) AddLabel(taskID, label string) error {
	query := `INSERT OR IGNORE INTO task_labels (task_id, label) VALUES (?, ?)`
	_, err := db.conn.Exec(query, taskID, label)
	return err
}

// RemoveLabel removes a label from a task
func (db *DB) RemoveLabel(taskID, label string) error {
	query := `DELETE FROM task_labels WHERE task_id = ? AND label = ?`
	_, err := db.conn.Exec(query, taskID, label)
	return err
}

// GetLabels returns all labels for a task
func (db *DB) GetLabels(taskID string) ([]string, error) {
	query := `SELECT label FROM task_labels WHERE task_id = ? ORDER BY label`
	rows, err := db.conn.Query(query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var labels []string
	for rows.Next() {
		var label string
		if err := rows.Scan(&label); err != nil {
			return nil, err
		}
		labels = append(labels, label)
	}
	return labels, rows.Err()
}

// GetAllLabels returns a map of task ID to labels for all tasks
func (db *DB) GetAllLabels() (map[string][]string, error) {
	query := `SELECT task_id, label FROM task_labels ORDER BY task_id, label`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	labels := make(map[string][]string)
	for rows.Next() {
		var taskID, label string
		if err := rows.Scan(&taskID, &label); err != nil {
			return nil, err
		}
		labels[taskID] = append(labels[taskID], label)
	}
	return labels, rows.Err()
}

// RemoveAllLabels removes all labels for a task
func (db *DB) RemoveAllLabels(taskID string) error {
	query := `DELETE FROM task_labels WHERE task_id = ?`
	_, err := db.conn.Exec(query, taskID)
	return err
}

// SetLabel atomically replaces all labels for a task with a single label.
// Pass an empty string to clear all labels.
func (db *DB) SetLabel(taskID, label string) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM task_labels WHERE task_id = ?`, taskID); err != nil {
		return err
	}
	if label != "" {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO task_labels (task_id, label) VALUES (?, ?)`, taskID, label); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// CreateLog inserts a log entry and syncs it to the FTS5 index
func (db *DB) CreateLog(taskID, message, logType string) error {
	result, err := db.conn.Exec(
		`INSERT INTO task_logs (task_id, message, type) VALUES (?, ?, ?)`,
		taskID, message, logType,
	)
	if err != nil {
		return err
	}

	rowID, err := result.LastInsertId()
	if err != nil {
		return err
	}

	// Manual FTS5 sync (triggers may not work with modernc.org/sqlite)
	_, err = db.conn.Exec(
		`INSERT INTO task_logs_fts(rowid, message) VALUES (?, ?)`,
		rowID, message,
	)
	return err
}

// GetLogsByTaskID returns all logs for a task ordered chronologically
func (db *DB) GetLogsByTaskID(taskID string) ([]LogRecord, error) {
	query := `SELECT id, task_id, message, type, created_at FROM task_logs WHERE task_id = ? ORDER BY created_at ASC`
	rows, err := db.conn.Query(query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []LogRecord
	for rows.Next() {
		var l LogRecord
		if err := rows.Scan(&l.ID, &l.TaskID, &l.Message, &l.Type, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// SearchLogs performs a full-text search across log messages
func (db *DB) SearchLogs(query string, limit int) ([]LogRecord, error) {
	sqlQuery := `
		SELECT tl.id, tl.task_id, tl.message, tl.type, tl.created_at
		FROM task_logs_fts fts
		JOIN task_logs tl ON tl.id = fts.rowid
		WHERE task_logs_fts MATCH ?
		ORDER BY bm25(task_logs_fts)
		LIMIT ?
	`
	rows, err := db.conn.Query(sqlQuery, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []LogRecord
	for rows.Next() {
		var l LogRecord
		if err := rows.Scan(&l.ID, &l.TaskID, &l.Message, &l.Type, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// AddTaskNote links a note to a task
func (db *DB) AddTaskNote(taskID, noteFilename string) error {
	query := `INSERT OR IGNORE INTO task_notes (task_id, note_filename) VALUES (?, ?)`
	_, err := db.conn.Exec(query, taskID, noteFilename)
	return err
}

// RemoveTaskNote unlinks a note from a task
func (db *DB) RemoveTaskNote(taskID, noteFilename string) error {
	query := `DELETE FROM task_notes WHERE task_id = ? AND note_filename = ?`
	_, err := db.conn.Exec(query, taskID, noteFilename)
	return err
}

// GetNotesForTask returns all note filenames linked to a task
func (db *DB) GetNotesForTask(taskID string) ([]string, error) {
	query := `SELECT note_filename FROM task_notes WHERE task_id = ? ORDER BY note_filename`
	rows, err := db.conn.Query(query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []string
	for rows.Next() {
		var filename string
		if err := rows.Scan(&filename); err != nil {
			return nil, err
		}
		notes = append(notes, filename)
	}
	return notes, rows.Err()
}

// GetTasksForNote returns all task IDs linked to a note
func (db *DB) GetTasksForNote(noteFilename string) ([]string, error) {
	query := `SELECT task_id FROM task_notes WHERE note_filename = ? ORDER BY task_id`
	rows, err := db.conn.Query(query, noteFilename)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []string
	for rows.Next() {
		var taskID string
		if err := rows.Scan(&taskID); err != nil {
			return nil, err
		}
		tasks = append(tasks, taskID)
	}
	return tasks, rows.Err()
}

// GetAllTaskNotes returns a map of task ID to linked note filenames
func (db *DB) GetAllTaskNotes() (map[string][]string, error) {
	query := `SELECT task_id, note_filename FROM task_notes ORDER BY task_id, note_filename`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notes := make(map[string][]string)
	for rows.Next() {
		var taskID, filename string
		if err := rows.Scan(&taskID, &filename); err != nil {
			return nil, err
		}
		notes[taskID] = append(notes[taskID], filename)
	}
	return notes, rows.Err()
}

// RemoveAllTaskNotes removes all note links for a task
func (db *DB) RemoveAllTaskNotes(taskID string) error {
	query := `DELETE FROM task_notes WHERE task_id = ?`
	_, err := db.conn.Exec(query, taskID)
	return err
}

// RemoveAllNoteLinks removes all task links for a note
func (db *DB) RemoveAllNoteLinks(noteFilename string) error {
	query := `DELETE FROM task_notes WHERE note_filename = ?`
	_, err := db.conn.Exec(query, noteFilename)
	return err
}

// DeleteLogsByTaskID removes all logs for a task and cleans up FTS5
func (db *DB) DeleteLogsByTaskID(taskID string) error {
	// Get IDs for FTS cleanup
	rows, err := db.conn.Query(`SELECT id, message FROM task_logs WHERE task_id = ?`, taskID)
	if err != nil {
		return err
	}
	defer rows.Close()

	type logEntry struct {
		id      int
		message string
	}
	var entries []logEntry
	for rows.Next() {
		var e logEntry
		if err := rows.Scan(&e.id, &e.message); err != nil {
			return err
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Remove from FTS5 index
	for _, e := range entries {
		_, err := db.conn.Exec(
			`INSERT INTO task_logs_fts(task_logs_fts, rowid, message) VALUES('delete', ?, ?)`,
			e.id, e.message,
		)
		if err != nil {
			return err
		}
	}

	// Delete from main table
	_, err = db.conn.Exec(`DELETE FROM task_logs WHERE task_id = ?`, taskID)
	return err
}
