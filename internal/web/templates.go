package web

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

// templates holds one fully-parsed template set per page, each composed with the
// shared base layout. Parsing per-page avoids "content"/"title" block clashes.
type templates struct {
	pages map[string]*template.Template
}

func parseTemplates() (*templates, error) {
	t := &templates{pages: make(map[string]*template.Template)}
	for _, page := range []string{"index", "setup", "share", "share_password", "login", "verify", "error", "shares"} {
		parsed, err := template.ParseFS(templateFS, "templates/base.html", "templates/"+page+".html")
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
	return tpl.ExecuteTemplate(w, "base.html", data)
}
