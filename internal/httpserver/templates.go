package httpserver

import (
	"embed"
	"fmt"
	"html/template"
	"io"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

// templates holds one standalone template per server-rendered page. Only the
// binary download's error page is server-rendered; the rest of the UI is the
// React SPA.
type templates struct {
	pages map[string]*template.Template
}

func parseTemplates() (*templates, error) {
	t := &templates{pages: make(map[string]*template.Template)}
	for _, page := range []string{"error"} {
		parsed, err := template.ParseFS(templateFS, "templates/"+page+".html")
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", page, err)
		}
		t.pages[page] = parsed
	}
	return t, nil
}

func (t *templates) render(w io.Writer, page string, data any) error {
	tpl, ok := t.pages[page]
	if !ok {
		return fmt.Errorf("unknown page %q", page)
	}
	return tpl.Execute(w, data)
}
