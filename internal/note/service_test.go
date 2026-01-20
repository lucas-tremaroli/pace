package note

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGetNotePath_WithFilename(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewServiceWithDir(tempDir)

	path := svc.GetNotePath("mynote")
	expected := filepath.Join(tempDir, "mynote.md")
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}

	// With .md extension already
	path = svc.GetNotePath("mynote.md")
	expected = filepath.Join(tempDir, "mynote.md")
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

func TestGetNotePath_EmptyFilename(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewServiceWithDir(tempDir)

	path := svc.GetNotePath("")
	today := time.Now().Format("2006-01-02")
	expected := filepath.Join(tempDir, today+".md")
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

func TestWriteNote_Success(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewServiceWithDir(tempDir)

	err := svc.WriteNote("test", "Hello, World!")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify file was created
	path := filepath.Join(tempDir, "test.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	expected := "Hello, World!\n"
	if string(content) != expected {
		t.Errorf("expected content %q, got %q", expected, string(content))
	}
}

func TestReadNote_Success(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewServiceWithDir(tempDir)

	// Create a test file
	testContent := "# Test Note\n\nThis is a test."
	path := filepath.Join(tempDir, "test.md")
	if err := os.WriteFile(path, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	content, err := svc.ReadNote("test")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if content != testContent {
		t.Errorf("expected content %q, got %q", testContent, content)
	}
}

func TestReadNote_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewServiceWithDir(tempDir)

	_, err := svc.ReadNote("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}

	if !os.IsNotExist(err) {
		t.Errorf("expected IsNotExist error, got %v", err)
	}
}

func TestDeleteNote_Success(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewServiceWithDir(tempDir)

	// Create a test file
	path := filepath.Join(tempDir, "todelete.md")
	if err := os.WriteFile(path, []byte("delete me"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	err := svc.DeleteNote("todelete.md")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify file was deleted
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected file to be deleted")
	}
}

func TestGetNotesDir(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewServiceWithDir(tempDir)

	if svc.GetNotesDir() != tempDir {
		t.Errorf("expected %q, got %q", tempDir, svc.GetNotesDir())
	}
}

func TestWriteAndReadNote_RoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewServiceWithDir(tempDir)

	originalContent := "# My Note\n\nSome content here."
	err := svc.WriteNote("roundtrip", originalContent)
	if err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	content, err := svc.ReadNote("roundtrip")
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}

	// Note: WriteNote adds a newline
	expected := originalContent + "\n"
	if content != expected {
		t.Errorf("round trip failed: expected %q, got %q", expected, content)
	}
}

func TestGetNotePath_SpecialCharacters(t *testing.T) {
	tempDir := t.TempDir()
	svc := NewServiceWithDir(tempDir)

	// Test with spaces
	path := svc.GetNotePath("my note")
	if !strings.HasSuffix(path, "my note.md") {
		t.Errorf("expected path to end with 'my note.md', got %q", path)
	}
}
