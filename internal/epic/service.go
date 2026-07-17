package epic

import (
	"database/sql"
	"errors"

	"github.com/lucas-tremaroli/pace/internal/storage"
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
	epic.setCreatedAt(r.CreatedAt)
	return epic
}

// DeleteEpic removes an epic by ID, first unlinking any tasks that point at it.
func (s *Service) DeleteEpic(id string) error {
	if err := s.db.ClearTaskEpic(id); err != nil {
		return err
	}
	err := s.db.DeleteEpic(id)
	if errors.Is(err, storage.ErrNotFound) {
		return ErrEpicNotFound
	}
	return err
}

// LoadEpicsByStatus returns every epic in the given lifecycle state.
func (s *Service) LoadEpicsByStatus(status Status) ([]Epic, error) {
	epics, err := s.LoadAllEpics()
	if err != nil {
		return nil, err
	}
	var out []Epic
	for _, e := range epics {
		if e.Status() == status {
			out = append(out, e)
		}
	}
	return out, nil
}
