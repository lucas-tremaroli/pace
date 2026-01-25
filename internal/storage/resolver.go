package storage

import (
	"os"
	"path/filepath"
)

// StorageType indicates the type of storage location
type StorageType string

const (
	StorageTypeProject StorageType = "project"
	StorageTypeGlobal  StorageType = "global"
)

// ResolvedPath contains the resolved pace directory path and its type
type ResolvedPath struct {
	Path string      `json:"path"`
	Type StorageType `json:"type"`
}

// PaceDirName is the name of the project-specific pace directory
const PaceDirName = ".pace"

// ResolvePaceDir determines the appropriate pace directory using the following priority:
// 1. Search upward from cwd for .pace/ directory (project storage)
// 2. Fall back to ~/.config/pace/ (global storage)
func ResolvePaceDir() (ResolvedPath, error) {
	// Priority 1: Search upward from cwd for .pace/ directory
	cwd, err := os.Getwd()
	if err != nil {
		return ResolvedPath{}, err
	}

	if projectRoot := findProjectRoot(cwd); projectRoot != "" {
		return ResolvedPath{
			Path: filepath.Join(projectRoot, PaceDirName),
			Type: StorageTypeProject,
		}, nil
	}

	// Priority 2: Fall back to global ~/.config/pace/
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ResolvedPath{}, err
	}

	return ResolvedPath{
		Path: filepath.Join(homeDir, ".config", "pace"),
		Type: StorageTypeGlobal,
	}, nil
}

// findProjectRoot searches upward from startDir looking for a .pace/ directory.
// Returns the directory containing .pace/, or empty string if not found.
func findProjectRoot(startDir string) string {
	dir := startDir
	for {
		paceDir := filepath.Join(dir, PaceDirName)
		if info, err := os.Stat(paceDir); err == nil && info.IsDir() {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return ""
		}
		dir = parent
	}
}

// InitProjectDir creates a .pace/ directory structure in the target directory.
// Returns the path to the created .pace/ directory.
func InitProjectDir(targetDir string) (string, error) {
	paceDir := filepath.Join(targetDir, PaceDirName)

	// Create .pace/ directory
	if err := os.MkdirAll(paceDir, 0755); err != nil {
		return "", err
	}

	// Create notes/ subdirectory
	notesDir := filepath.Join(paceDir, "notes")
	if err := os.MkdirAll(notesDir, 0755); err != nil {
		return "", err
	}

	return paceDir, nil
}

// FindExistingProjectDir searches upward from startDir for an existing .pace/ directory.
// Returns the path to the .pace/ directory if found, or empty string if not found.
func FindExistingProjectDir(startDir string) string {
	if projectRoot := findProjectRoot(startDir); projectRoot != "" {
		return filepath.Join(projectRoot, PaceDirName)
	}
	return ""
}
