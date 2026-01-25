package note

import (
	"bufio"
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
	paceDir, err := storage.GetPaceConfigDir()
	if err != nil {
		return nil, err
	}
	notesDir := filepath.Join(paceDir, "notes")
	if err := os.MkdirAll(notesDir, 0755); err != nil {
		return nil, err
	}
	return &Service{notesDir: notesDir}, nil
}

// NewServiceWithDir creates a service with a custom notes directory (for testing)
func NewServiceWithDir(notesDir string) *Service {
	return &Service{notesDir: notesDir}
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

func (s *Service) OpenInEditor(filename, editor string) error {
	path := s.GetNotePath(filename)
	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (s *Service) WriteNote(filename, content string) error {
	path := s.GetNotePath(filename)
	return os.WriteFile(path, []byte(content+"\n"), 0644)
}

func (s *Service) GetNotesDir() string {
	return s.notesDir
}

func (s *Service) DeleteNote(filename string) error {
	path := filepath.Join(s.notesDir, filename)
	return os.Remove(path)
}

func (s *Service) ReadNote(filename string) (string, error) {
	path := s.GetNotePath(filename)
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

type NoteInfo struct {
	Filename  string    `json:"filename"`
	Path      string    `json:"path"`
	FirstLine string    `json:"firstLine"`
	ModTime   time.Time `json:"modTime"`
}

func (s *Service) ListNotes() ([]NoteInfo, error) {
	entries, err := os.ReadDir(s.notesDir)
	if err != nil {
		return nil, err
	}

	var notes []NoteInfo
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			path := filepath.Join(s.notesDir, e.Name())
			firstLine := readFirstLineFromPath(path)
			info, err := e.Info()
			var modTime time.Time
			if err == nil {
				modTime = info.ModTime()
			}
			notes = append(notes, NoteInfo{
				Filename:  e.Name(),
				Path:      path,
				FirstLine: firstLine,
				ModTime:   modTime,
			})
		}
	}
	return notes, nil
}

func readFirstLineFromPath(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		line = strings.TrimLeft(line, "# ")
		return line
	}
	return ""
}
