package web

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/http"

	"progames/internal/auth"
)

//go:embed templates/*.html
var templateFS embed.FS

type viewData struct {
	Title         string
	Content       template.HTML
	Authenticated bool
	CSRF          string
	Data          any
}

func newTemplates() *template.Template {
	return template.Must(template.New("").Funcs(template.FuncMap{
		"fmtDuration": fmtDuration,
		"inc":         func(i int) int { return i + 1 },
	}).ParseFS(templateFS, "templates/*.html"))
}

func fmtDuration(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	s := float64(ms) / 1000
	if s < 60 {
		return fmt.Sprintf("%.1fs", s)
	}
	return fmt.Sprintf("%dm %ds", int(s)/60, int(s)%60)
}

func (fe *Frontend) render(w http.ResponseWriter, r *http.Request, title, name string, data any) {
	fe.renderStatus(w, r, http.StatusOK, title, name, data)
}

func (fe *Frontend) renderStatus(w http.ResponseWriter, r *http.Request, status int, title, name string, data any) {
	if isHTMX(r) {
		var buf bytes.Buffer
		if err := fe.templates.ExecuteTemplate(&buf, name, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)
		_, _ = w.Write(buf.Bytes())
		return
	}
	var content bytes.Buffer
	if err := fe.templates.ExecuteTemplate(&content, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	session, hasSession := auth.CurrentSession(r)
	page := viewData{
		Title:         title,
		Content:       template.HTML(content.String()),
		Authenticated: hasSession,
		Data:          data,
	}
	if hasSession {
		page.CSRF = session.CSRFToken
	}
	var layout bytes.Buffer
	if err := fe.templates.ExecuteTemplate(&layout, "layout", page); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(layout.Bytes())
}
