package note

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lucas-tremaroli/pace/internal/storage"
	"gopkg.in/yaml.v3"
)

type Service struct {
	notesDir string
	db       NoteRepository
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
	db, err := storage.NewDB()
	if err != nil {
		return nil, err
	}
	return &Service{notesDir: notesDir, db: db}, nil
}

// NewServiceWithDir creates a service with a custom notes directory (for testing).
func NewServiceWithDir(notesDir string) *Service {
	return &Service{notesDir: notesDir}
}

// Close closes the database connection
func (s *Service) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
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

// GetLinkedTaskIDs returns task IDs linked to the note, or nil when no
// DB is wired. Cheap query — no file IO.
func (s *Service) GetLinkedTaskIDs(filename string) ([]string, error) {
	if s.db == nil {
		return nil, nil
	}
	return s.db.GetTasksForNote(filename)
}

func (s *Service) DeleteNote(filename string) error {
	// Clean up task links before deleting
	if s.db != nil {
		if err := s.db.RemoveAllNoteLinks(filename); err != nil {
			return err
		}
	}
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
	Filename    string    `json:"filename"`
	Path        string    `json:"path"`
	Description string    `json:"description"`
	ModTime     time.Time `json:"modTime"`
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
			description := getDescriptionFromPath(path)
			info, err := e.Info()
			var modTime time.Time
			if err == nil {
				modTime = info.ModTime()
			}
			notes = append(notes, NoteInfo{
				Filename:    e.Name(),
				Path:        path,
				Description: description,
				ModTime:     modTime,
			})
		}
	}
	return notes, nil
}

// metaReadLimit bounds how many bytes we read when we only need the
// frontmatter and the first body line. Larger than any plausible
// frontmatter block plus a leading paragraph.
const metaReadLimit = 4096

// readPrefix reads up to limit bytes from path. Short reads on small
// files are fine — we only need enough to cover frontmatter + first
// non-empty line.
func readPrefix(path string, limit int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, limit)
	n, err := f.Read(buf)
	if err != nil && n == 0 {
		return nil, err
	}
	return buf[:n], nil
}

// descriptionFrom returns frontmatter description if present, otherwise
// the first non-empty body line. Single parse — caller passes raw content.
func descriptionFrom(content string) string {
	fm, body, _ := ParseFrontmatter(content)
	if fm != nil && fm.Description != "" {
		return fm.Description
	}
	return firstNonEmptyLine(body)
}

// getDescriptionFromPath returns the description from frontmatter, or falls back to first content line
func getDescriptionFromPath(path string) string {
	contentBytes, err := readPrefix(path, metaReadLimit)
	if err != nil {
		return ""
	}
	return descriptionFrom(string(contentBytes))
}

// ParseFrontmatter extracts YAML frontmatter from content.
// Returns the parsed frontmatter, the remaining content after frontmatter, and any error.
func ParseFrontmatter(content string) (*Frontmatter, string, error) {
	// Check for frontmatter delimiter
	if !strings.HasPrefix(content, "---\n") && !strings.HasPrefix(content, "---\r\n") {
		return nil, content, nil
	}

	// Find the closing delimiter
	rest := content[4:] // Skip opening "---\n"
	endIndex := strings.Index(rest, "\n---\n")
	if endIndex == -1 {
		// Try with \r\n
		endIndex = strings.Index(rest, "\r\n---\r\n")
		if endIndex == -1 {
			// Also check for end-of-content case
			if strings.HasSuffix(rest, "\n---") {
				endIndex = len(rest) - 4
			} else {
				return nil, content, nil
			}
		}
	}

	frontmatterYAML := rest[:endIndex]
	remaining := strings.TrimPrefix(rest[endIndex:], "\n---\n")
	remaining = strings.TrimPrefix(remaining, "\r\n---\r\n")
	remaining = strings.TrimPrefix(remaining, "\n---")

	var fm Frontmatter
	if err := yaml.Unmarshal([]byte(frontmatterYAML), &fm); err != nil {
		return nil, content, fmt.Errorf("invalid frontmatter YAML: %w", err)
	}

	return &fm, strings.TrimPrefix(remaining, "\n"), nil
}

// ReadNoteWithMeta reads a single note with full metadata including labels
func (s *Service) ReadNoteWithMeta(filename string) (*Note, error) {
	path := s.GetNotePath(filename)
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(contentBytes)
	fm, body, err := ParseFrontmatter(content)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(path)
	var modTime time.Time
	if err == nil {
		modTime = info.ModTime()
	}

	var labels []string
	var description string
	if fm != nil {
		labels = fm.Labels
		description = fm.Description
	}
	if description == "" {
		description = firstNonEmptyLine(body)
	}

	n := &Note{
		Filename:    filepath.Base(path),
		Path:        path,
		Content:     content,
		Description: description,
		ModTime:     modTime,
		Labels:      labels,
	}

	// Populate linked tasks if DB is available
	if s.db != nil {
		tasks, err := s.db.GetTasksForNote(n.Filename)
		if err == nil {
			n.Tasks = tasks
		}
	}

	return n, nil
}

// ReadAllNotes reads all notes with full content and metadata
func (s *Service) ReadAllNotes() ([]Note, error) {
	entries, err := os.ReadDir(s.notesDir)
	if err != nil {
		return nil, err
	}

	var notes []Note
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			note, err := s.ReadNoteWithMeta(e.Name())
			if err != nil {
				continue // Skip notes that can't be read
			}
			notes = append(notes, *note)
		}
	}
	return notes, nil
}

// ListNoteNames returns notes with only filenames and paths, without reading
// file contents. This is much faster than ListNotesWithMeta for use cases that
// only need to display note names (e.g. TUI list).
func (s *Service) ListNoteNames() ([]Note, error) {
	entries, err := os.ReadDir(s.notesDir)
	if err != nil {
		return nil, err
	}

	var notes []Note
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			notes = append(notes, Note{
				Filename: e.Name(),
				Path:     filepath.Join(s.notesDir, e.Name()),
			})
		}
	}
	return notes, nil
}

// ListNotesWithMeta lists notes with optional content inclusion and label metadata
func (s *Service) ListNotesWithMeta(includeContent bool) ([]Note, error) {
	entries, err := os.ReadDir(s.notesDir)
	if err != nil {
		return nil, err
	}

	var notes []Note
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			path := filepath.Join(s.notesDir, e.Name())
			var contentBytes []byte
			var rerr error
			if includeContent {
				contentBytes, rerr = os.ReadFile(path)
			} else {
				contentBytes, rerr = readPrefix(path, metaReadLimit)
			}
			if rerr != nil {
				notes = append(notes, Note{
					Filename:    e.Name(),
					Path:        path,
					Description: "(read error: " + rerr.Error() + ")",
				})
				continue
			}

			content := string(contentBytes)
			fm, body, perr := ParseFrontmatter(content)
			if perr != nil {
				fm = nil
				body = content
			}

			info, ierr := e.Info()
			var modTime time.Time
			if ierr == nil {
				modTime = info.ModTime()
			}

			var labels []string
			var description string
			if fm != nil {
				labels = fm.Labels
				description = fm.Description
			}
			if description == "" {
				description = firstNonEmptyLine(body)
			}

			note := Note{
				Filename:    e.Name(),
				Path:        path,
				Description: description,
				ModTime:     modTime,
				Labels:      labels,
			}

			if includeContent {
				note.Content = content
			}

			notes = append(notes, note)
		}
	}
	return notes, nil
}

// MergeNotes combines multiple notes into a single output note.
// Labels from all source notes are deduplicated and combined.
// Content is concatenated with --- separators.
func (s *Service) MergeNotes(filenames []string, outputFilename string) (*Note, error) {
	if len(filenames) < 2 {
		return nil, fmt.Errorf("at least 2 notes required for merge")
	}

	var allLabels []string
	labelSet := make(map[string]bool)
	var contentParts []string

	for _, filename := range filenames {
		note, err := s.ReadNoteWithMeta(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to read note %s: %w", filename, err)
		}

		// Collect unique labels
		for _, label := range note.Labels {
			if !labelSet[label] {
				labelSet[label] = true
				allLabels = append(allLabels, label)
			}
		}

		// Strip frontmatter from content for merging
		_, bodyContent, _ := ParseFrontmatter(note.Content)
		contentParts = append(contentParts, strings.TrimSpace(bodyContent))
	}

	// Build merged content
	var mergedContent strings.Builder

	// Add frontmatter if there are labels
	if len(allLabels) > 0 {
		mergedContent.WriteString("---\nlabels:\n")
		for _, label := range allLabels {
			mergedContent.WriteString("  - ")
			mergedContent.WriteString(label)
			mergedContent.WriteString("\n")
		}
		mergedContent.WriteString("---\n\n")
	}

	// Add content with separators
	mergedContent.WriteString(strings.Join(contentParts, "\n\n---\n\n"))

	// Write the merged note
	if err := s.WriteNote(outputFilename, mergedContent.String()); err != nil {
		return nil, fmt.Errorf("failed to write merged note: %w", err)
	}

	return s.ReadNoteWithMeta(outputFilename)
}

// extractFirstLine extracts the first meaningful line from content, skipping frontmatter
func extractFirstLine(content string) string {
	_, body, _ := ParseFrontmatter(content)
	return firstNonEmptyLine(body)
}

// firstNonEmptyLine returns the first non-empty line of body with any
// leading markdown heading marks stripped.
func firstNonEmptyLine(body string) string {
	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			return strings.TrimLeft(line, "# ")
		}
	}
	return ""
}
