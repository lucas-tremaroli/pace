package epic

import "testing"

func TestValidate_EmptyTitle(t *testing.T) {
	e := NewEpic("epic-1", Planning, "", "summary")
	if err := e.Validate(); err != ErrEmptyTitle {
		t.Errorf("expected ErrEmptyTitle, got %v", err)
	}
}

func TestValidate_InvalidStatus(t *testing.T) {
	e := Epic{id: "epic-1", status: Status(-1), title: "valid title"}
	if err := e.Validate(); err != ErrInvalidStatus {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}

	e.status = Status(99)
	if err := e.Validate(); err != ErrInvalidStatus {
		t.Errorf("expected ErrInvalidStatus for status 99, got %v", err)
	}
}

func TestValidate_Success(t *testing.T) {
	for _, s := range []Status{Planning, Active, Done} {
		e := NewEpic("epic-1", s, "title", "")
		if err := e.Validate(); err != nil {
			t.Errorf("expected nil error for status %v, got %v", s, err)
		}
	}
}

func TestSetStatus(t *testing.T) {
	e := NewEpic("epic-1", Planning, "title", "")
	if err := e.SetStatus(Active); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if e.Status() != Active {
		t.Errorf("expected status Active, got %v", e.Status())
	}
	if err := e.SetStatus(Status(99)); err != ErrInvalidStatus {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}
}

func TestParseStatus(t *testing.T) {
	tests := []struct {
		in      string
		want    Status
		wantErr bool
	}{
		{"planning", Planning, false},
		{"active", Active, false},
		{"done", Done, false},
		{"bogus", 0, true},
	}
	for _, tc := range tests {
		got, err := ParseStatus(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseStatus(%q): expected error, got nil", tc.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseStatus(%q): unexpected error %v", tc.in, err)
		}
		if got != tc.want {
			t.Errorf("ParseStatus(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestToJSON(t *testing.T) {
	e := NewEpic("epic-1", Active, "title", "summary")
	j := e.ToJSON()
	if j.ID != "epic-1" || j.Title != "title" || j.Summary != "summary" || j.Status != "active" {
		t.Errorf("unexpected JSON form: %+v", j)
	}
}

func TestGenerateID(t *testing.T) {
	id := GenerateID()
	if got, want := id[:len(IDPrefix)+1], IDPrefix+"-"; got != want {
		t.Errorf("expected prefix %q, got %q", want, got)
	}
	if len(id) != len(IDPrefix)+1+IDLength {
		t.Errorf("expected length %d, got %d (%s)", len(IDPrefix)+1+IDLength, len(id), id)
	}
}
