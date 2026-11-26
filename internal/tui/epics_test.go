package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lucas-tremaroli/pace/internal/epic"
	"github.com/lucas-tremaroli/pace/internal/task"
)

type fakeEpicSvc struct {
	created, updated []epic.Epic
}

func (f *fakeEpicSvc) LoadAllEpics() ([]epic.Epic, error) { return nil, nil }
func (f *fakeEpicSvc) GenerateEpicID() string             { return "epic-new" }
func (f *fakeEpicSvc) CreateEpic(e epic.Epic) error       { f.created = append(f.created, e); return nil }
func (f *fakeEpicSvc) UpdateEpic(e epic.Epic) error       { f.updated = append(f.updated, e); return nil }
func (f *fakeEpicSvc) DeleteEpic(string) error            { return nil }

type fakeTaskSvc struct{}

func (fakeTaskSvc) LoadAllTasks() ([]task.Task, error) { return nil, nil }

func TestSaveEditedEpic(t *testing.T) {
	base := epic.NewEpic("epic-abc", epic.Planning, "old", "")
	edited := epic.Format(base)
	edited = edited[:len("---\n")] + "title: new title\nstatus: active\n" + edited[len("---\ntitle: old\nsummary: \"\"\nstatus: planning\n"):]

	for _, create := range []bool{false, true} {
		svc := &fakeEpicSvc{}
		path := filepath.Join(t.TempDir(), "e.md")
		if err := os.WriteFile(path, []byte(edited), 0o600); err != nil {
			t.Fatal(err)
		}
		tab := newEpics(svc, fakeTaskSvc{})
		msg := tab.saveEditedCmd(epicEditedMsg{path: path, base: base, create: create})()
		if _, ok := msg.(epicsMutatedMsg); !ok {
			t.Fatalf("create=%v: want epicsMutatedMsg, got %#v", create, msg)
		}
		got := svc.updated
		if create {
			got = svc.created
		}
		if len(got) != 1 || got[0].Title() != "new title" || got[0].Status() != epic.Active || got[0].ID() != "epic-abc" {
			t.Errorf("create=%v: saved %+v", create, got)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("create=%v: temp file should be removed after save", create)
		}
	}
}

func TestSaveEditedEpicKeepsFileOnParseError(t *testing.T) {
	svc := &fakeEpicSvc{}
	path := filepath.Join(t.TempDir(), "e.md")
	if err := os.WriteFile(path, []byte("garbage"), 0o600); err != nil {
		t.Fatal(err)
	}
	tab := newEpics(svc, fakeTaskSvc{})
	msg := tab.saveEditedCmd(epicEditedMsg{path: path, base: epic.NewEpic("epic-abc", epic.Planning, "t", "")})()
	if _, ok := msg.(epicLoadedMsg); !ok {
		t.Fatalf("want epicLoadedMsg error, got %#v", msg)
	}
	if len(svc.updated)+len(svc.created) != 0 {
		t.Error("nothing should be saved on a parse error")
	}
	if _, err := os.Stat(path); err != nil {
		t.Error("rejected edit must be kept on disk")
	}
}
