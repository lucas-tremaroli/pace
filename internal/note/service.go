package note

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/lucas-tremaroli/pace/internal/storage"
)

// NoteService defines the interface for note operations
type NoteService interface {
	SaveNote(filename, content string) error
}

// FileNoteService implements NoteService using the filesystem
type FileNoteService struct{}

// NewFileNoteService creates a new FileNoteService
func NewFileNoteService() *FileNoteService {
	return &FileNoteService{}
}

// SaveNote saves a note to the filesystem
func (s *FileNoteService) SaveNote(filename, content string) error {
	// Ensure filename has .md extension
	if !strings.HasSuffix(filename, ".md") {
		filename += ".md"
	}

	paceDir, err := storage.GetpaceConfigDir()
	if err != nil {
		return err
	}

	notesDir := filepath.Join(paceDir, "notes")
	if err := os.MkdirAll(notesDir, 0755); err != nil {
		return err
	}

	filePath := filepath.Join(notesDir, filename)
	return os.WriteFile(filePath, []byte(content), 0644)
}
