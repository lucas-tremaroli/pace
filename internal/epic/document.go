package epic

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// sectionHeadings maps the editable document's "## " headings to spec fields.
// Order is the order sections are written.
var sectionHeadings = []struct {
	name string
	get  func(*Spec) *string
}{
	{"Current state", func(s *Spec) *string { return &s.CurrentState }},
	{"Target state", func(s *Spec) *string { return &s.TargetState }},
	{"Constraints", func(s *Spec) *string { return &s.Constraints }},
	{"Exclusions", func(s *Spec) *string { return &s.Exclusions }},
	{"Notes", func(s *Spec) *string { return &s.Freeform }},
}

type docFrontmatter struct {
	Title   string   `yaml:"title"`
	Summary string   `yaml:"summary"`
	Status  string   `yaml:"status"`
	Labels  []string `yaml:"labels"`
}

// Format renders an epic as a markdown document for editing in $EDITOR:
// YAML frontmatter for title/summary/status/labels, then one "## " section
// per spec field. Empty sections are kept so there is somewhere to type.
func Format(e Epic) string {
	fm, _ := yaml.Marshal(docFrontmatter{
		Title: e.title, Summary: e.summary, Status: e.status.String(), Labels: e.labels,
	})
	spec := e.spec
	var b strings.Builder
	b.WriteString("---\n" + string(fm) + "---\n")
	for _, sec := range sectionHeadings {
		b.WriteString("\n## " + sec.name + "\n\n")
		if body := strings.TrimSpace(*sec.get(&spec)); body != "" {
			b.WriteString(body + "\n")
		}
	}
	return b.String()
}

// Parse applies an edited document onto base, keeping its ID and creation
// time. Text before the first known heading is appended to the Notes section
// so nothing the user typed is dropped.
func Parse(content string, base Epic) (Epic, error) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	rest, ok := strings.CutPrefix(content, "---\n")
	if !ok {
		return base, fmt.Errorf("missing frontmatter (expected a leading ---)")
	}
	yml, body, ok := strings.Cut(rest, "\n---")
	if !ok {
		return base, fmt.Errorf("unterminated frontmatter (expected a closing ---)")
	}
	var fm docFrontmatter
	if err := yaml.Unmarshal([]byte(yml), &fm); err != nil {
		return base, fmt.Errorf("invalid frontmatter: %w", err)
	}
	status, err := ParseStatus(fm.Status)
	if err != nil {
		return base, err
	}

	var spec Spec
	var preamble []string
	current := (*string)(nil)
	var buf []string
	flush := func() {
		text := strings.TrimSpace(strings.Join(buf, "\n"))
		if current != nil {
			*current = text
		} else if text != "" {
			preamble = append(preamble, text)
		}
		buf = nil
	}
	for _, line := range strings.Split(body, "\n") {
		if name, ok := strings.CutPrefix(line, "## "); ok {
			if sec := findSection(strings.TrimSpace(name)); sec != nil {
				flush()
				current = sec.get(&spec)
				continue
			}
		}
		buf = append(buf, line)
	}
	flush()
	if len(preamble) > 0 {
		spec.Freeform = strings.TrimSpace(strings.Join(append(preamble, spec.Freeform), "\n\n"))
	}

	e := base
	e.title, e.summary, e.status, e.spec = fm.Title, fm.Summary, status, spec
	e.SetLabels(fm.Labels)
	return e, e.Validate()
}

func findSection(name string) *struct {
	name string
	get  func(*Spec) *string
} {
	for i := range sectionHeadings {
		if strings.EqualFold(sectionHeadings[i].name, name) {
			return &sectionHeadings[i]
		}
	}
	return nil
}
