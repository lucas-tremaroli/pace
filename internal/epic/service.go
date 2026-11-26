package epic

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/lucas-tremaroli/pace/v2/internal/storage"
)

// Service handles epic business logic and storage operations.
type Service struct {
	db EpicRepository
}

// NewService creates a new epic service backed by the default SQLite storage.
func NewService() (*Service, error) {
	db, err := storage.NewDB()
	if err != nil {
		return nil, err
	}
	return &Service{db: db}, nil
}

// NewServiceWithRepository constructs a service with a caller-provided
// repository (useful for tests).
func NewServiceWithRepository(repo EpicRepository) *Service {
	return &Service{db: repo}
}

// Close releases the underlying storage handle.
func (s *Service) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// GenerateEpicID returns a new unique epic ID.
func (s *Service) GenerateEpicID() string {
	return GenerateID()
}

// CreateEpic persists a new epic after validating it.
func (s *Service) CreateEpic(e Epic) error {
	if err := e.Validate(); err != nil {
		return err
	}
	return s.db.CreateEpic(toRecord(e))
}

// UpdateEpic persists changes to an existing epic.
func (s *Service) UpdateEpic(e Epic) error {
	if err := e.Validate(); err != nil {
		return err
	}
	err := s.db.UpdateEpic(toRecord(e))
	if errors.Is(err, storage.ErrNotFound) {
		return ErrEpicNotFound
	}
	return err
}

// GetEpicByID retrieves a single epic.
func (s *Service) GetEpicByID(id string) (*Epic, error) {
	record, err := s.db.GetEpicByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEpicNotFound
		}
		return nil, err
	}
	epic := fromRecord(*record)
	return &epic, nil
}

// LoadAllEpics returns every epic in storage.
func (s *Service) LoadAllEpics() ([]Epic, error) {
	records, err := s.db.GetAllEpics()
	if err != nil {
		return nil, err
	}
	epics := make([]Epic, 0, len(records))
	for _, r := range records {
		epics = append(epics, fromRecord(r))
	}
	return epics, nil
}

func toRecord(e Epic) storage.EpicRecord {
	spec := e.Spec()
	return storage.EpicRecord{
		ID:           e.ID(),
		Title:        e.Title(),
		Summary:      e.Summary(),
		Status:       int(e.Status()),
		CurrentState: spec.CurrentState,
		TargetState:  spec.TargetState,
		Constraints:  spec.Constraints,
		Exclusions:   spec.Exclusions,
		Freeform:     spec.Freeform,
		Labels:       encodeLabels(e.Labels()),
	}
}

func fromRecord(r storage.EpicRecord) Epic {
	epic := NewEpic(r.ID, Status(r.Status), r.Title, r.Summary)
	epic.SetSpec(Spec{
		CurrentState: r.CurrentState,
		TargetState:  r.TargetState,
		Constraints:  r.Constraints,
		Exclusions:   r.Exclusions,
		Freeform:     r.Freeform,
	})
	epic.SetLabels(decodeLabels(r.Labels))
	epic.setCreatedAt(r.CreatedAt)
	return epic
}

// DeleteEpic removes an epic by ID. Storage atomically unlinks any tasks that
// pointed at it in the same transaction.
func (s *Service) DeleteEpic(id string) error {
	err := s.db.DeleteEpic(id)
	if errors.Is(err, storage.ErrNotFound) {
		return ErrEpicNotFound
	}
	return err
}

// LoadOpenEpics returns epics that still need work: planning or active.
func (s *Service) LoadOpenEpics() ([]Epic, error) {
	epics, err := s.LoadAllEpics()
	if err != nil {
		return nil, err
	}
	var out []Epic
	for _, e := range epics {
		if e.Status() != Done {
			out = append(out, e)
		}
	}
	return out, nil
}

func encodeLabels(labels []string) string {
	if len(labels) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(labels) // []string never fails to marshal
	return string(b)
}

func decodeLabels(s string) []string {
	var labels []string
	_ = json.Unmarshal([]byte(s), &labels) // empty/invalid column reads as no labels
	return labels
}

// MergeEpics combines two or more epics into a new one titled title. Each
// spec section is the sources' sections joined with "---" separators, labels
// are the deduplicated union, and the new status is the furthest-along of the
// sources. Tasks move to the new epic and the sources are deleted.
func (s *Service) MergeEpics(ids []string, title string) (*Epic, error) {
	if len(ids) < 2 {
		return nil, fmt.Errorf("at least 2 epics required for merge")
	}
	if title == "" {
		return nil, ErrEmptyTitle
	}
	seen := make(map[string]bool, len(ids))
	var parts [5][]string
	var labels, summaries []string
	status := Planning
	for _, id := range ids {
		if seen[id] {
			return nil, fmt.Errorf("duplicate epic %s", id)
		}
		seen[id] = true
		e, err := s.GetEpicByID(id)
		if err != nil {
			return nil, fmt.Errorf("epic %s: %w", id, err)
		}
		if e.Status() > status {
			status = e.Status()
		}
		labels = append(labels, e.Labels()...)
		if e.Summary() != "" {
			summaries = append(summaries, e.Summary())
		}
		spec := e.Spec()
		for i, sec := range sectionHeadings {
			if body := strings.TrimSpace(*sec.get(&spec)); body != "" {
				parts[i] = append(parts[i], body)
			}
		}
	}

	merged := NewEpic(s.GenerateEpicID(), status, title, strings.Join(summaries, " / "))
	var spec Spec
	for i, sec := range sectionHeadings {
		*sec.get(&spec) = strings.Join(parts[i], "\n\n---\n\n")
	}
	merged.SetSpec(spec)
	merged.SetLabels(labels)
	if err := merged.Validate(); err != nil {
		return nil, err
	}
	if err := s.db.MergeEpics(toRecord(merged), ids); err != nil {
		return nil, err
	}
	return s.GetEpicByID(merged.ID())
}
