package epic

import "github.com/lucas-tremaroli/pace/internal/storage"

// EpicRepository abstracts storage operations the service needs.
// *storage.DB satisfies this interface; alternative implementations (e.g.
// in-memory) can be used in tests.
type EpicRepository interface {
	Close() error
	CreateEpic(rec storage.EpicRecord) error
	GetAllEpics() ([]storage.EpicRecord, error)
	GetEpicByID(id string) (*storage.EpicRecord, error)
	UpdateEpic(rec storage.EpicRecord) error
	DeleteEpic(id string) error
	ClearTaskEpic(epicID string) error
}
