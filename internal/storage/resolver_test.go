package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePaceDir_ProjectDirectory(t *testing.T) {
	// Set up temp directory with .pace/
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "myproject")
	paceDir := filepath.Join(projectDir, ".pace")
	if err := os.MkdirAll(paceDir, 0755); err != nil {
		t.Fatalf("failed to create test dirs: %v", err)
	}

	// Change to project directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(projectDir)

	resolved, err := ResolvePaceDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Evaluate symlinks for comparison (macOS /var -> /private/var)
	expectedPath, _ := filepath.EvalSymlinks(paceDir)
	actualPath, _ := filepath.EvalSymlinks(resolved.Path)

	if actualPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, actualPath)
	}
	if resolved.Type != StorageTypeProject {
		t.Errorf("expected type %q, got %q", StorageTypeProject, resolved.Type)
	}
}

func TestResolvePaceDir_ProjectSubdirectory(t *testing.T) {
	// Set up temp directory with .pace/ and a subdirectory
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "myproject")
	paceDir := filepath.Join(projectDir, ".pace")
	subDir := filepath.Join(projectDir, "src", "pkg")
	if err := os.MkdirAll(paceDir, 0755); err != nil {
		t.Fatalf("failed to create .pace dir: %v", err)
	}
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	// Change to subdirectory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(subDir)

	resolved, err := ResolvePaceDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Evaluate symlinks for comparison (macOS /var -> /private/var)
	expectedPath, _ := filepath.EvalSymlinks(paceDir)
	actualPath, _ := filepath.EvalSymlinks(resolved.Path)

	if actualPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, actualPath)
	}
	if resolved.Type != StorageTypeProject {
		t.Errorf("expected type %q, got %q", StorageTypeProject, resolved.Type)
	}
}

func TestResolvePaceDir_GlobalFallback(t *testing.T) {
	// Set up temp directory WITHOUT .pace/
	tmpDir := t.TempDir()

	// Change to temp directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	resolved, err := ResolvePaceDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	homeDir, _ := os.UserHomeDir()
	expectedPath := filepath.Join(homeDir, ".config", "pace")

	if resolved.Path != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, resolved.Path)
	}
	if resolved.Type != StorageTypeGlobal {
		t.Errorf("expected type %q, got %q", StorageTypeGlobal, resolved.Type)
	}
}

func TestInitProjectDir(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "newproject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	paceDir, err := InitProjectDir(projectDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedPaceDir := filepath.Join(projectDir, ".pace")
	if paceDir != expectedPaceDir {
		t.Errorf("expected %q, got %q", expectedPaceDir, paceDir)
	}

	// Verify .pace/ was created
	if _, err := os.Stat(expectedPaceDir); os.IsNotExist(err) {
		t.Error(".pace directory was not created")
	}

	// Verify notes/ subdirectory was created
	notesDir := filepath.Join(expectedPaceDir, "notes")
	if _, err := os.Stat(notesDir); os.IsNotExist(err) {
		t.Error("notes directory was not created")
	}
}

func TestFindExistingProjectDir(t *testing.T) {
	// Set up temp directory with .pace/
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "myproject")
	paceDir := filepath.Join(projectDir, ".pace")
	subDir := filepath.Join(projectDir, "src", "pkg")
	if err := os.MkdirAll(paceDir, 0755); err != nil {
		t.Fatalf("failed to create .pace dir: %v", err)
	}
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	// Test from subdirectory
	result := FindExistingProjectDir(subDir)
	if result != paceDir {
		t.Errorf("expected %q, got %q", paceDir, result)
	}

	// Test from project root
	result = FindExistingProjectDir(projectDir)
	if result != paceDir {
		t.Errorf("expected %q, got %q", paceDir, result)
	}

	// Test from directory without .pace/
	result = FindExistingProjectDir(tmpDir)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestFindProjectRoot(t *testing.T) {
	// Set up temp directory structure
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "myproject")
	paceDir := filepath.Join(projectDir, ".pace")
	subDir := filepath.Join(projectDir, "src", "pkg", "deep")

	if err := os.MkdirAll(paceDir, 0755); err != nil {
		t.Fatalf("failed to create .pace dir: %v", err)
	}
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	// Test from deep subdirectory
	result := findProjectRoot(subDir)
	if result != projectDir {
		t.Errorf("expected %q, got %q", projectDir, result)
	}

	// Test from project root
	result = findProjectRoot(projectDir)
	if result != projectDir {
		t.Errorf("expected %q, got %q", projectDir, result)
	}

	// Test from outside project
	result = findProjectRoot(tmpDir)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}
