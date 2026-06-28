package epic

import "fmt"

// Status represents the lifecycle stage of an epic.
type Status int

const (
	Planning Status = iota
	Active
	Done
)

// String returns the string representation of a status.
func (s Status) String() string {
	switch s {
	case Planning:
		return "planning"
	case Active:
		return "active"
	case Done:
		return "done"
	default:
		return "unknown"
	}
}

// ParseStatus parses a string into a Status value.
func ParseStatus(s string) (Status, error) {
	switch s {
	case "planning":
		return Planning, nil
	case "active":
		return Active, nil
	case "done":
		return Done, nil
	default:
		return 0, fmt.Errorf("invalid status: %s (valid: planning, active, done)", s)
	}
}

// Spec captures the design statement for an epic. Each section is markdown.
// Freeform is a fallback for users/agents that don't want the structured form;
// it is used alongside, not instead of, the structured sections.
type Spec struct {
	CurrentState string `json:"current_state,omitempty"`
	TargetState  string `json:"target_state,omitempty"`
	Constraints  string `json:"constraints,omitempty"`
	Exclusions   string `json:"exclusions,omitempty"`
	Freeform     string `json:"freeform,omitempty"`
}

// Epic is the in-memory representation of an epic record.
type Epic struct {
	id        string
	status    Status
	title     string
	summary   string
	spec      Spec
	createdAt string
}

// EpicJSON is the JSON-serializable form of an Epic.
type EpicJSON struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Summary   string `json:"summary,omitempty"`
	Status    string `json:"status"`
	Spec      Spec   `json:"spec"`
	CreatedAt string `json:"created_at,omitempty"`
}

// NewEpic constructs a new epic.
func NewEpic(id string, status Status, title, summary string) Epic {
	return Epic{
		id:      id,
		status:  status,
		title:   title,
		summary: summary,
	}
}

func (e Epic) ID() string        { return e.id }
func (e Epic) Title() string     { return e.title }
func (e Epic) Summary() string   { return e.summary }
func (e Epic) Status() Status    { return e.status }
func (e Epic) Spec() Spec        { return e.spec }
func (e Epic) CreatedAt() string { return e.createdAt }

// SetTitle updates the title.
func (e *Epic) SetTitle(title string) { e.title = title }

// SetSummary updates the summary.
func (e *Epic) SetSummary(summary string) { e.summary = summary }

// SetSpec replaces the entire spec.
func (e *Epic) SetSpec(spec Spec) { e.spec = spec }

// SetCurrentState updates only the current-state section.
func (e *Epic) SetCurrentState(s string) { e.spec.CurrentState = s }

// SetTargetState updates only the target-state section.
func (e *Epic) SetTargetState(s string) { e.spec.TargetState = s }

// SetConstraints updates only the constraints section.
func (e *Epic) SetConstraints(s string) { e.spec.Constraints = s }

// SetExclusions updates only the exclusions section.
func (e *Epic) SetExclusions(s string) { e.spec.Exclusions = s }

// SetFreeform updates only the freeform fallback section.
func (e *Epic) SetFreeform(s string) { e.spec.Freeform = s }

// SetStatus updates the status with validation.
func (e *Epic) SetStatus(s Status) error {
	if s < Planning || s > Done {
		return ErrInvalidStatus
	}
	e.status = s
	return nil
}

// setCreatedAt is used by the service layer when hydrating from storage.
func (e *Epic) setCreatedAt(t string) { e.createdAt = t }

// Validate checks that the epic has valid field values.
func (e Epic) Validate() error {
	if e.id == "" {
		return ErrEmptyID
	}
	if e.title == "" {
		return ErrEmptyTitle
	}
	if e.status < Planning || e.status > Done {
		return ErrInvalidStatus
	}
	return nil
}

// ToJSON converts an Epic to its JSON-serializable form.
func (e Epic) ToJSON() EpicJSON {
	return EpicJSON{
		ID:        e.id,
		Title:     e.title,
		Summary:   e.summary,
		Status:    e.status.String(),
		Spec:      e.spec,
		CreatedAt: e.createdAt,
	}
}
