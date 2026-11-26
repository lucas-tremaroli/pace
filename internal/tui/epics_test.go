package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lucas-tremaroli/pace/internal/epic"
	"github.com/lucas-tremaroli/pace/internal/task"
)

type fakeEpicSvc struct {
	created, updated []epic.Epic
	deleteErr        error
}

func (f *fakeEpicSvc) LoadAllEpics() ([]epic.Epic, error) { return nil, nil }
func (f *fakeEpicSvc) GenerateEpicID() string             { return "epic-new" }
func (f *fakeEpicSvc) CreateEpic(e epic.Epic) error       { f.created = append(f.created, e); return nil }
func (f *fakeEpicSvc) UpdateEpic(e epic.Epic) error       { f.updated = append(f.updated, e); return nil }
func (f *fakeEpicSvc) DeleteEpic(string) error            { return f.deleteErr }

type fakeTaskSvc struct{ loads int }

func (f *fakeTaskSvc) LoadAllTasks() ([]task.Task, error) { f.loads++; return nil, nil }

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
		tab := newEpics(svc, &fakeTaskSvc{})
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
	tab := newEpics(svc, &fakeTaskSvc{})
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

func TestDeleteErrorIsShownNotSwallowed(t *testing.T) {
	svc := &fakeEpicSvc{deleteErr: errors.New("database is locked")}
	tab := newEpics(svc, &fakeTaskSvc{})
	msg := tab.deleteCmd("epic-abc")()
	loaded, ok := msg.(epicLoadedMsg)
	if !ok {
		t.Fatalf("a failed delete must not look like success, got %#v", msg)
	}
	if !strings.Contains(loaded.content, "database is locked") {
		t.Errorf("error not shown: %q", loaded.content)
	}

	svc.deleteErr = nil
	if _, ok := tab.deleteCmd("epic-abc")().(epicsMutatedMsg); !ok {
		t.Error("a successful delete should refresh the list")
	}
}

// Tasks change on the Kanban tab without the epics tab hearing about it, so
// every Enter must reload; only a load already in flight is skipped.
func TestEnterAlwaysReloadsEpicDetail(t *testing.T) {
	tasks := &fakeTaskSvc{}
	tab := newEpics(&fakeEpicSvc{}, tasks)
	tab.list.SetItems([]list.Item{epicItem{Epic: epic.NewEpic("epic-abc", epic.Active, "x", "")}})

	for i := 1; i <= 2; i++ {
		cmd := tab.loadCmd()
		if cmd == nil {
			t.Fatalf("enter #%d did not start a load", i)
		}
		msg := runLoad(t, cmd)
		tab.Update(msg)
		if tasks.loads != i {
			t.Errorf("after enter #%d tasks were loaded %d times, want %d", i, tasks.loads, i)
		}
	}

	// A second Enter while the first is still loading is a no-op.
	tab.loadCmd()
	if again := tab.loadCmd(); again != nil {
		t.Error("enter during an in-flight load should be ignored")
	}
}

// runLoad executes a loadCmd (a batch of spinner tick + read) and returns the
// epicLoadedMsg the read produces.
func runLoad(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	batch, ok := cmd().(tea.BatchMsg)
	if !ok {
		t.Fatal("loadCmd should be a batch of tick + read")
	}
	for _, c := range batch {
		if c == nil {
			continue
		}
		if m := c(); m != nil {
			if _, ok := m.(epicLoadedMsg); ok {
				return m
			}
		}
	}
	t.Fatal("loadCmd produced no epicLoadedMsg")
	return nil
}
