package epic

import "github.com/lucas-tremaroli/pace/internal/storage"

// EpicRepository abstracts storage operations the service needs.
// *storage.DB satisfies this interface; alternative implementations (e.g.
// in-memory) can be used in tests.
type EpicRepository interface {
	Close() error
	CreateEpic(id, title, summary string, status int) error
	GetAllEpics() ([]storage.EpicRecord, error)
	GetEpicByID(id string) (*storage.EpicRecord, error)
	UpdateEpic(id, title, summary string, status int) error
	DeleteEpic(id string) error
}
