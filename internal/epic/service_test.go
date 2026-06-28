package epic

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/lucas-tremaroli/pace/internal/storage"
)

// memRepo is an in-memory EpicRepository used only by tests.
type memRepo struct {
	records map[string]storage.EpicRecord
}

func newMemRepo() *memRepo {
	return &memRepo{records: map[string]storage.EpicRecord{}}
}

func (m *memRepo) Close() error { return nil }

func (m *memRepo) CreateEpic(id, title, summary string, status int) error {
	if _, exists := m.records[id]; exists {
		return errors.New("duplicate id")
	}
	m.records[id] = storage.EpicRecord{
		ID:        id,
		Title:     title,
		Summary:   summary,
		Status:    status,
		CreatedAt: "2026-06-28T00:00:00Z",
	}
	return nil
}

func (m *memRepo) GetAllEpics() ([]storage.EpicRecord, error) {
	out := make([]storage.EpicRecord, 0, len(m.records))
	for _, r := range m.records {
		out = append(out, r)
	}
	return out, nil
}

func (m *memRepo) GetEpicByID(id string) (*storage.EpicRecord, error) {
	r, ok := m.records[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return &r, nil
}

func (m *memRepo) UpdateEpic(id, title, summary string, status int) error {
	r, ok := m.records[id]
	if !ok {
		return storage.ErrNotFound
	}
	r.Title = title
	r.Summary = summary
	r.Status = status
	m.records[id] = r
	return nil
}

func (m *memRepo) DeleteEpic(id string) error {
	if _, ok := m.records[id]; !ok {
		return storage.ErrNotFound
	}
	delete(m.records, id)
	return nil
}

func TestService_CreateAndGet(t *testing.T) {
	svc := NewServiceWithRepository(newMemRepo())
	e := NewEpic("epic-aaa", Planning, "spec the thing", "rough notes")
	if err := svc.CreateEpic(e); err != nil {
		t.Fatalf("CreateEpic: %v", err)
	}

	got, err := svc.GetEpicByID("epic-aaa")
	if err != nil {
		t.Fatalf("GetEpicByID: %v", err)
	}
	if got.Title() != "spec the thing" || got.Summary() != "rough notes" || got.Status() != Planning {
		t.Errorf("unexpected epic loaded: %+v", got.ToJSON())
	}
}

func TestService_CreateRejectsInvalid(t *testing.T) {
	svc := NewServiceWithRepository(newMemRepo())
	if err := svc.CreateEpic(NewEpic("epic-1", Planning, "", "")); err != ErrEmptyTitle {
		t.Errorf("expected ErrEmptyTitle, got %v", err)
	}
	bad := NewEpic("epic-1", Planning, "title", "")
	bad.status = Status(99)
	if err := svc.CreateEpic(bad); err != ErrInvalidStatus {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}
}

func TestService_Update(t *testing.T) {
	svc := NewServiceWithRepository(newMemRepo())
	e := NewEpic("epic-bbb", Planning, "v1", "")
	if err := svc.CreateEpic(e); err != nil {
		t.Fatalf("seed: %v", err)
	}

	e.SetTitle("v2")
	if err := e.SetStatus(Active); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	if err := svc.UpdateEpic(e); err != nil {
		t.Fatalf("UpdateEpic: %v", err)
	}

	got, err := svc.GetEpicByID("epic-bbb")
	if err != nil {
		t.Fatalf("GetEpicByID: %v", err)
	}
	if got.Title() != "v2" || got.Status() != Active {
		t.Errorf("update did not persist: %+v", got.ToJSON())
	}
}

func TestService_UpdateMissing(t *testing.T) {
	svc := NewServiceWithRepository(newMemRepo())
	e := NewEpic("epic-missing", Planning, "title", "")
	if err := svc.UpdateEpic(e); err != ErrEpicNotFound {
		t.Errorf("expected ErrEpicNotFound, got %v", err)
	}
}

func TestService_DeleteAndMissing(t *testing.T) {
	svc := NewServiceWithRepository(newMemRepo())
	if err := svc.CreateEpic(NewEpic("epic-ccc", Planning, "doomed", "")); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := svc.DeleteEpic("epic-ccc"); err != nil {
		t.Fatalf("DeleteEpic: %v", err)
	}
	if _, err := svc.GetEpicByID("epic-ccc"); err != ErrEpicNotFound {
		t.Errorf("expected ErrEpicNotFound after delete, got %v", err)
	}
	if err := svc.DeleteEpic("epic-ccc"); err != ErrEpicNotFound {
		t.Errorf("expected ErrEpicNotFound on second delete, got %v", err)
	}
}

func TestService_LoadAll(t *testing.T) {
	svc := NewServiceWithRepository(newMemRepo())
	for _, id := range []string{"epic-1", "epic-2", "epic-3"} {
		if err := svc.CreateEpic(NewEpic(id, Planning, "t-"+id, "")); err != nil {
			t.Fatalf("seed %s: %v", id, err)
		}
	}
	all, err := svc.LoadAllEpics()
	if err != nil {
		t.Fatalf("LoadAllEpics: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 epics, got %d", len(all))
	}
}
