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
	Status      int    `json:"status"`
	TaskType    int    `json:"task_type"`
	Priority    int    `json:"priority"`
	Link        string `json:"link"`
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

	// Migration: add task_type column if it doesn't exist (0 = task, 1 = bug, 2 = feature, 3 = chore, 4 = docs)
	_, err = db.conn.Exec(`ALTER TABLE tasks ADD COLUMN task_type INTEGER NOT NULL DEFAULT 0`)
	// Ignore error if column already exists
	_ = err

	// Migration: add link column if it doesn't exist
	_, err = db.conn.Exec(`ALTER TABLE tasks ADD COLUMN link VARCHAR DEFAULT ''`)
	// Ignore error if column already exists
	_ = err

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

func (db *DB) CreateTask(id, title, description string, status, taskType, priority int, link string) error {
	query := `INSERT INTO tasks (id, title, description, status, task_type, priority, link) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := db.conn.Exec(query, id, title, description, status, taskType, priority, link)
	return err
}

func (db *DB) GetAllTasks() ([]TaskRecord, error) {
	query := `SELECT id, title, description, status, task_type, priority, COALESCE(link, '') FROM tasks ORDER BY priority DESC, title`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []TaskRecord
	for rows.Next() {
		var task TaskRecord
		err := rows.Scan(&task.ID, &task.Title, &task.Description, &task.Status, &task.TaskType, &task.Priority, &task.Link)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

func (db *DB) UpdateTask(id, title, description string, status, taskType, priority int, link string) error {
	query := `UPDATE tasks SET title = ?, description = ?, status = ?, task_type = ?, priority = ?, link = ? WHERE id = ?`
	_, err := db.conn.Exec(query, title, description, status, taskType, priority, link, id)
	return err
}

func (db *DB) DeleteTask(id string) error {
	query := `DELETE FROM tasks WHERE id = ?`
	_, err := db.conn.Exec(query, id)
	return err
}

func (db *DB) GetTaskByID(id string) (*TaskRecord, error) {
	query := `SELECT id, title, description, status, task_type, priority, COALESCE(link, '') FROM tasks WHERE id = ?`
	row := db.conn.QueryRow(query, id)

	var task TaskRecord
	err := row.Scan(&task.ID, &task.Title, &task.Description, &task.Status, &task.TaskType, &task.Priority, &task.Link)
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
