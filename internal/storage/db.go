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
	Priority    int    `json:"priority"`
}

func NewDB() (*DB, error) {
	dbPath, err := getDBPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get database path: %w", err)
	}

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
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	paceDir := filepath.Join(homeDir, ".config", "pace")
	if err := os.MkdirAll(paceDir, 0755); err != nil {
		return "", err
	}

	return paceDir, nil
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

func (db *DB) CreateTask(id, title, description string, status, priority int) error {
	query := `INSERT INTO tasks (id, title, description, status, priority) VALUES (?, ?, ?, ?, ?)`
	_, err := db.conn.Exec(query, id, title, description, status, priority)
	return err
}

func (db *DB) GetAllTasks() ([]TaskRecord, error) {
	query := `SELECT id, title, description, status, priority FROM tasks ORDER BY priority DESC, title`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []TaskRecord
	for rows.Next() {
		var task TaskRecord
		err := rows.Scan(&task.ID, &task.Title, &task.Description, &task.Status, &task.Priority)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

func (db *DB) UpdateTask(id, title, description string, status, priority int) error {
	query := `UPDATE tasks SET title = ?, description = ?, status = ?, priority = ? WHERE id = ?`
	_, err := db.conn.Exec(query, title, description, status, priority, id)
	return err
}

func (db *DB) DeleteTask(id string) error {
	query := `DELETE FROM tasks WHERE id = ?`
	_, err := db.conn.Exec(query, id)
	return err
}

func (db *DB) GetTaskByID(id string) (*TaskRecord, error) {
	query := `SELECT id, title, description, status, priority FROM tasks WHERE id = ?`
	row := db.conn.QueryRow(query, id)

	var task TaskRecord
	err := row.Scan(&task.ID, &task.Title, &task.Description, &task.Status, &task.Priority)
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
