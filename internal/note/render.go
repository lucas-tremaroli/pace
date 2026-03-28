package note

import (
	"sync"

	"github.com/charmbracelet/glamour"
)

var (
	rendererMu         sync.Mutex
	rendererCache      *glamour.TermRenderer
	rendererCacheWidth int
)

func getRenderer(width int) *glamour.TermRenderer {
	rendererMu.Lock()
	defer rendererMu.Unlock()
	if rendererCache != nil && rendererCacheWidth == width {
		return rendererCache
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil
	}
	rendererCache = r
	rendererCacheWidth = width
	return r
}

func RenderMarkdown(content string) string {
	return RenderMarkdownWithWidth(content, 100)
}

func RenderMarkdownWithWidth(content string, width int) string {
	_, body, _ := ParseFrontmatter(content)

	r := getRenderer(width)
	if r == nil {
		return body
	}

	rendered, err := r.Render(body)
	if err != nil {
		return body
	}

	return rendered
}
