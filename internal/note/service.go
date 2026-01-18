package note

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lucas-tremaroli/pace/internal/storage"
)

type Service struct {
	notesDir string
}

func NewService() (*Service, error) {
	paceDir, err := storage.GetpaceConfigDir()
	if err != nil {
		return nil, err
	}
	notesDir := filepath.Join(paceDir, "notes")
	if err := os.MkdirAll(notesDir, 0755); err != nil {
		return nil, err
	}
	return &Service{notesDir: notesDir}, nil
}

func (s *Service) GetNotePath(filename string) string {
	if filename == "" {
		filename = time.Now().Format("2006-01-02")
	}
	if !strings.HasSuffix(filename, ".md") {
		filename += ".md"
	}
	return filepath.Join(s.notesDir, filename)
}

func (s *Service) OpenInEditor(filename string) error {
	path := s.GetNotePath(filename)
	nvim := exec.Command("nvim", path)
	nvim.Stdin = os.Stdin
	nvim.Stdout = os.Stdout
	nvim.Stderr = os.Stderr
	return nvim.Run()
}
