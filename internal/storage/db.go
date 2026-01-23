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

	return nil
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
