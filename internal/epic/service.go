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
	return s.db.CreateEpic(e.ID(), e.Title(), e.Summary(), int(e.Status()))
}

// UpdateEpic persists changes to an existing epic.
func (s *Service) UpdateEpic(e Epic) error {
	if err := e.Validate(); err != nil {
		return err
	}
	err := s.db.UpdateEpic(e.ID(), e.Title(), e.Summary(), int(e.Status()))
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
	epic := NewEpic(record.ID, Status(record.Status), record.Title, record.Summary)
	epic.setCreatedAt(record.CreatedAt)
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
		epic := NewEpic(r.ID, Status(r.Status), r.Title, r.Summary)
		epic.setCreatedAt(r.CreatedAt)
		epics = append(epics, epic)
	}
	return epics, nil
}

// DeleteEpic removes an epic by ID.
func (s *Service) DeleteEpic(id string) error {
	err := s.db.DeleteEpic(id)
	if errors.Is(err, storage.ErrNotFound) {
		return ErrEpicNotFound
	}
	return err
}
