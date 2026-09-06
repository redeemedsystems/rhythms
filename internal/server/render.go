// Package server wires HTTP routes, request handlers, and HTML rendering
// together for the Rhythms application.
package server

import (
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
)

var funcMap = template.FuncMap{
	"join":        strings.Join,
	"streakBadge": streakBadge,
	"isLast":      func(i, n int) bool { return i == n-1 },
}

// Base carries the fields every full-page template needs; page-specific data
// structs embed it so {{.CSRFToken}} etc. resolve via Go's promoted-field
// reflection even though the concrete value is page-specific.
type Base struct {
	LoggedIn  bool
	Nav       string
	CSRFToken string
	Flash     string
}

type Renderer struct {
	pages    map[string]*template.Template
	partials *template.Template
}

func NewRenderer(fsys fs.FS) (*Renderer, error) {
	pageFiles, err := fs.Glob(fsys, "templates/pages/*.html")
	if err != nil {
		return nil, err
	}

	pages := make(map[string]*template.Template, len(pageFiles))
	for _, pf := range pageFiles {
		t, err := template.New("layout").Funcs(funcMap).ParseFS(fsys, "templates/layout.html", pf, "templates/partials/*.html")
		if err != nil {
			return nil, err
		}
		pages[filepath.Base(pf)] = t
	}

	partials, err := template.New("partials").Funcs(funcMap).ParseFS(fsys, "templates/partials/*.html")
	if err != nil {
		return nil, err
	}

	return &Renderer{pages: pages, partials: partials}, nil
}

// Page renders the named page (e.g. "today.html") inside the shared layout.
func (rn *Renderer) Page(w http.ResponseWriter, name string, data any) {
	t, ok := rn.pages[name]
	if !ok {
		http.Error(w, "template not found: "+name, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "layout", data); err != nil {
		slog.Error("render page", "template", name, "err", err)
	}
}

// Partial renders a single named partial (an htmx-swappable fragment) with
// no layout wrapper.
func (rn *Renderer) Partial(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := rn.partials.ExecuteTemplate(w, name, data); err != nil {
		slog.Error("render partial", "template", name, "err", err)
	}
}
