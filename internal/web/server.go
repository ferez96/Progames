package web

import (
	"database/sql"
	"expvar"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"progames/internal/auth"
	"progames/internal/obs"
	"progames/internal/store"
	"progames/internal/submission"
)

type authService interface {
	Require(next http.Handler) http.Handler
	UserFromRequest(r *http.Request) (store.User, store.Session, error)
	SignUp(name, email, password string) (int64, error)
	SignIn(email, password string) (store.User, string, string, error)
	SignOut(sessionID string) error
	SetSessionCookie(w http.ResponseWriter, r *http.Request, sessionID string)
}

type storeService interface {
	SystemAgents() ([]store.Agent, error)
	AgentByID(id int64) (store.Agent, error)
	MatchByID(id int64) (store.Match, error)
	ListGames(matchID int64) ([]store.Game, error)
	ExecutionLog(matchID int64) (string, error)
	ListMoves(gameID int64) ([]store.Move, error)
	UserCanViewMatch(userID, matchID int64) (bool, error)
}

type submissionService interface {
	Submit(userID int64, code string) (submission.Result, error)
}

type matchService interface {
	RunPractice(userAgentID, systemAgentID int64) (int64, error)
}

type Server struct {
	auth       authService
	store      storeService
	submission submissionService
	match      matchService
	templates  *template.Template
}

func New(st storeService, authSvc authService, subSvc submissionService, matchSvc matchService) *Server {
	return &Server{
		auth:       authSvc,
		store:      st,
		submission: subSvc,
		match:      matchSvc,
		templates:  template.Must(template.New("pages").Parse(pages)),
	}
}

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(requestLogger)
	r.Get("/", s.home)
	r.Get("/signup", s.signupForm)
	r.Post("/signup", s.signup)
	r.Get("/login", s.loginForm)
	r.Post("/login", s.login)
	r.Method("GET", "/debug/vars", expvar.Handler())
	r.Group(func(r chi.Router) {
		r.Use(s.auth.Require)
		r.Get("/practice", s.practice)
		r.Post("/practice/run", s.runPractice)
		r.Get("/matches/{id}", s.matchSummary)
		r.Get("/matches/{id}/logs", s.matchLogs)
		r.Get("/matches/{id}/replay", s.matchReplay)
		r.Post("/logout", s.logout)
	})
	return r
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sr := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sr, r)
		zap.L().Info("http",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", sr.status),
			zap.Int64("dur_ms", time.Since(start).Milliseconds()),
		)
	})
}

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	if _, _, err := s.auth.UserFromRequest(r); err == nil {
		http.Redirect(w, r, "/practice", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) signupForm(w http.ResponseWriter, r *http.Request) {
	s.render(w, "signup", nil)
}

func (s *Server) signup(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.render(w, "signup", map[string]any{"Error": err.Error()})
		return
	}
	if _, err := s.auth.SignUp(r.FormValue("name"), r.FormValue("email"), r.FormValue("password")); err != nil {
		s.render(w, "signup", map[string]any{"Error": err.Error()})
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) loginForm(w http.ResponseWriter, r *http.Request) {
	s.render(w, "login", nil)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.render(w, "login", map[string]any{"Error": err.Error()})
		return
	}
	_, sessionID, _, err := s.auth.SignIn(r.FormValue("email"), r.FormValue("password"))
	if err != nil {
		obs.LoginsFailure.Add(1)
		s.render(w, "login", map[string]any{"Error": "invalid email or password"})
		return
	}
	obs.LoginsSuccess.Add(1)
	s.auth.SetSessionCookie(w, r, sessionID)
	http.Redirect(w, r, "/practice", http.StatusSeeOther)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if !auth.ValidateCSRF(r) {
		http.Error(w, "invalid csrf token", http.StatusForbidden)
		return
	}
	cookie, err := r.Cookie(auth.SessionCookieName)
	if err == nil {
		_ = s.auth.SignOut(cookie.Value)
	}
	auth.ClearSessionCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) practice(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.CurrentSession(r)
	agents, err := s.store.SystemAgents()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.render(w, "practice", map[string]any{
		"CSRF":   session.CSRFToken,
		"Agents": agents,
		"Code":   defaultCode,
	})
}

func (s *Server) runPractice(w http.ResponseWriter, r *http.Request) {
	if !auth.ValidateCSRF(r) {
		http.Error(w, "invalid csrf token", http.StatusForbidden)
		return
	}
	user, _ := auth.CurrentUser(r)
	code, err := readCode(r)
	if err != nil {
		s.practiceError(w, r, err.Error())
		return
	}
	res, err := s.submission.Submit(user.ID, code)
	if err != nil {
		s.practiceError(w, r, err.Error())
		return
	}
	if res.Status != "compiled" {
		s.practiceError(w, r, fmt.Sprintf("%s\n%s", res.Message, res.Output))
		return
	}
	opponentID, err := strconv.ParseInt(r.FormValue("opponent_agent_id"), 10, 64)
	if err != nil {
		s.practiceError(w, r, "select a system opponent")
		return
	}
	opponent, err := s.store.AgentByID(opponentID)
	if err != nil || opponent.Type != "system" {
		s.practiceError(w, r, "invalid system opponent")
		return
	}
	matchID, err := s.match.RunPractice(res.AgentID, opponentID)
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/matches/%d", matchID), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/matches/%d", matchID), http.StatusSeeOther)
}

func (s *Server) practiceError(w http.ResponseWriter, r *http.Request, msg string) {
	session, _ := auth.CurrentSession(r)
	agents, _ := s.store.SystemAgents()
	s.render(w, "practice", map[string]any{
		"CSRF":   session.CSRFToken,
		"Agents": agents,
		"Code":   defaultCode,
		"Error":  msg,
	})
}

func (s *Server) matchSummary(w http.ResponseWriter, r *http.Request) {
	matchID, ok := s.authorizedMatch(w, r)
	if !ok {
		return
	}
	m, err := s.store.MatchByID(matchID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	games, _ := s.store.ListGames(matchID)
	s.render(w, "match", map[string]any{"Match": m, "Games": games})
}

func (s *Server) matchLogs(w http.ResponseWriter, r *http.Request) {
	matchID, ok := s.authorizedMatch(w, r)
	if !ok {
		return
	}
	content, err := s.store.ExecutionLog(matchID)
	if err != nil {
		if err == sql.ErrNoRows {
			content = "No execution log available."
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	s.render(w, "logs", map[string]any{"MatchID": matchID, "Content": content})
}

func (s *Server) matchReplay(w http.ResponseWriter, r *http.Request) {
	matchID, ok := s.authorizedMatch(w, r)
	if !ok {
		return
	}
	games, err := s.store.ListGames(matchID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type gameReplay struct {
		Game  store.Game
		Moves []store.Move
	}
	var replay []gameReplay
	for _, game := range games {
		moves, _ := s.store.ListMoves(game.ID)
		replay = append(replay, gameReplay{Game: game, Moves: moves})
	}
	s.render(w, "replay", map[string]any{"MatchID": matchID, "Replay": replay})
}

func (s *Server) authorizedMatch(w http.ResponseWriter, r *http.Request) (int64, bool) {
	user, _ := auth.CurrentUser(r)
	matchID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid match id", http.StatusBadRequest)
		return 0, false
	}
	ok, err := s.store.UserCanViewMatch(user.ID, matchID)
	if err != nil || !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return 0, false
	}
	return matchID, true
}

func readCode(r *http.Request) (string, error) {
	if err := r.ParseMultipartForm(1 << 20); err == nil {
		file, _, err := r.FormFile("source_file")
		if err == nil {
			defer file.Close()
			raw, err := io.ReadAll(file)
			return string(raw), err
		}
	}
	if err := r.ParseForm(); err != nil {
		return "", err
	}
	return r.FormValue("source"), nil
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

const defaultCode = `package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	in := bufio.NewScanner(os.Stdin)
	for in.Scan() {
		state := in.Text()
		for i, cell := range state {
			if cell == '.' {
				fmt.Printf("%d,%d\n", i%8+1, i/8+1)
				os.Stdout.Sync()
				break
			}
		}
		_ = strings.TrimSpace(state)
	}
}
`

const pages = `
{{define "login"}}<!doctype html><title>Login</title><h1>Login</h1>{{with .Error}}<pre>{{.}}</pre>{{end}}<form method="post"><input name="email" placeholder="email"><input name="password" type="password" placeholder="password"><button>Login</button></form><a href="/signup">Sign up</a>{{end}}
{{define "signup"}}<!doctype html><title>Sign up</title><h1>Sign up</h1>{{with .Error}}<pre>{{.}}</pre>{{end}}<form method="post"><input name="name" placeholder="name"><input name="email" placeholder="email"><input name="password" type="password" placeholder="password"><button>Sign up</button></form><a href="/login">Login</a>{{end}}
{{define "practice"}}<!doctype html><title>Practice</title><h1>Practice</h1>{{with .Error}}<pre>{{.}}</pre>{{end}}<form method="post" action="/practice/run" enctype="multipart/form-data"><input type="hidden" name="csrf_token" value="{{.CSRF}}"><p>Opponent: <select name="opponent_agent_id">{{range .Agents}}<option value="{{.ID}}">{{.Name}}</option>{{end}}</select></p><p><textarea name="source" rows="24" cols="90">{{.Code}}</textarea></p><p>or upload main.go: <input type="file" name="source_file"></p><button>Build and run practice</button></form><form method="post" action="/logout"><input type="hidden" name="csrf_token" value="{{.CSRF}}"><button>Logout</button></form>{{end}}
{{define "match"}}<!doctype html><title>Match</title><h1>Match #{{.Match.ID}}</h1><p>Status: {{.Match.Status}}</p><p>Winner agent: {{if .Match.WinnerAgentID.Valid}}{{.Match.WinnerAgentID.Int64}}{{else}}draw/none{{end}}</p><p><a href="/matches/{{.Match.ID}}/logs">Logs</a> | <a href="/matches/{{.Match.ID}}/replay">Replay</a> | <a href="/practice">New practice</a></p><h2>Games</h2><ul>{{range .Games}}<li>Game #{{.ID}} result={{.Result}} moves={{.MoveCount}} duration={{.DurationMS}}ms</li>{{end}}</ul>{{end}}
{{define "logs"}}<!doctype html><title>Logs</title><h1>Logs for match #{{.MatchID}}</h1><pre>{{.Content}}</pre><a href="/matches/{{.MatchID}}">Back</a>{{end}}
{{define "replay"}}<!doctype html><title>Replay</title><h1>Replay for match #{{.MatchID}}</h1>{{range .Replay}}<h2>Game #{{.Game.ID}} ({{.Game.Result}})</h2><ol>{{range .Moves}}<li>seq={{.Seq}} agent={{.AgentID}} type={{.ActionType}} accepted={{.Accepted}} payload={{.ActionPayload}} duration={{if .DurationMS.Valid}}{{.DurationMS.Int64}}ms{{end}}</li>{{end}}</ol>{{end}}<a href="/matches/{{.MatchID}}">Back</a>{{end}}
`
