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
		"fmtDuration":   fmtDuration,
		"fmtGameResult": fmtGameResult,
		"inc":           func(i int) int { return i + 1 },
	}).ParseFS(templateFS, "templates/*.html"))
}

func fmtGameResult(r string) string {
	switch r {
	case "player_a_win":
		return "X wins"
	case "player_b_win":
		return "O wins"
	case "draw":
		return "Draw"
	default:
		return r
	}
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

func (s *Server) render(w http.ResponseWriter, r *http.Request, title, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if isHTMX(r) {
		if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	var content bytes.Buffer
	if err := s.templates.ExecuteTemplate(&content, name, data); err != nil {
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
	if err := s.templates.ExecuteTemplate(w, "layout", page); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
