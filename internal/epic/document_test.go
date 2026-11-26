package epic

import "testing"

func TestDocumentRoundTrip(t *testing.T) {
	e := NewEpic("epic-abc", Active, "Auth rework", "move to sessions")
	e.SetCurrentState("jwt everywhere")
	e.SetTargetState("server sessions\n\n- with ## inside text")
	e.SetFreeform("misc")
	e.SetLabels([]string{"design", "backend"})

	got, err := Parse(Format(e), NewEpic("epic-abc", Planning, "x", ""))
	if err != nil {
		t.Fatal(err)
	}
	if got.ID() != "epic-abc" || got.Title() != "Auth rework" || got.Status() != Active || got.Summary() != "move to sessions" {
		t.Errorf("metadata lost: %+v", got)
	}
	if got.Spec() != e.Spec() {
		t.Errorf("spec changed:\n got %+v\nwant %+v", got.Spec(), e.Spec())
	}
	if l := got.Labels(); len(l) != 2 || l[0] != "design" || l[1] != "backend" {
		t.Errorf("labels = %v", l)
	}
}

func TestParseErrorsAndPreamble(t *testing.T) {
	base := NewEpic("epic-abc", Planning, "t", "")
	if _, err := Parse("no frontmatter", base); err == nil {
		t.Error("want error for missing frontmatter")
	}
	if _, err := Parse("---\ntitle: \"\"\nstatus: planning\n---\n", base); err == nil {
		t.Error("want error for empty title")
	}
	if _, err := Parse("---\ntitle: a\nstatus: nope\n---\n", base); err == nil {
		t.Error("want error for bad status")
	}
	got, err := Parse("---\ntitle: a\nstatus: planning\n---\n\nstray text\n\n## Target state\n\nx\n", base)
	if err != nil {
		t.Fatal(err)
	}
	if got.Spec().Freeform != "stray text" || got.Spec().TargetState != "x" {
		t.Errorf("preamble handling wrong: %+v", got.Spec())
	}
}
