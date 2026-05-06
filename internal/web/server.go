package web

import (
	"bytes"
	"database/sql"
	"embed"
	"encoding/json"
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
	ListUserMatches(userID int64) ([]store.Match, error)
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

type viewData struct {
	Title         string
	Content       template.HTML
	Authenticated bool
	CSRF          string
	Data          any
}

//go:embed templates/*.html
var templateFS embed.FS

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

func New(st storeService, authSvc authService, subSvc submissionService, matchSvc matchService) *Server {
	tmpl := template.Must(template.New("").Funcs(template.FuncMap{
		"fmtDuration":   fmtDuration,
		"fmtGameResult": fmtGameResult,
		"inc":           func(i int) int { return i + 1 },
	}).ParseFS(templateFS, "templates/*.html"))
	return &Server{
		auth:       authSvc,
		store:      st,
		submission: subSvc,
		match:      matchSvc,
		templates:  tmpl,
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
	s.render(w, r, "Sign Up", "signup", nil)
}

func (s *Server) signup(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.render(w, r, "Sign Up", "signup", map[string]any{"Error": err.Error()})
		return
	}
	if _, err := s.auth.SignUp(r.FormValue("name"), r.FormValue("email"), r.FormValue("password")); err != nil {
		s.render(w, r, "Sign Up", "signup", map[string]any{"Error": err.Error()})
		return
	}
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/login")
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) loginForm(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, "Login", "login", nil)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.render(w, r, "Login", "login", map[string]any{"Error": err.Error()})
		return
	}
	_, sessionID, _, err := s.auth.SignIn(r.FormValue("email"), r.FormValue("password"))
	if err != nil {
		obs.LoginsFailure.Add(1)
		s.render(w, r, "Login", "login", map[string]any{"Error": "invalid email or password"})
		return
	}
	obs.LoginsSuccess.Add(1)
	s.auth.SetSessionCookie(w, r, sessionID)
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/practice")
		return
	}
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
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/login")
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

type matchRow struct {
	ID      int64
	Status  string
	Outcome string // "Win" | "Loss" | "Draw" | "–"
	OppName string
	DurMS   sql.NullInt64
}

func (s *Server) practice(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.CurrentSession(r)
	user, _ := auth.CurrentUser(r)
	agents, err := s.store.SystemAgents()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	matches, _ := s.store.ListUserMatches(user.ID)
	var rows []matchRow
	for _, m := range matches {
		agentA, _ := s.store.AgentByID(m.AgentAID)
		agentB, _ := s.store.AgentByID(m.AgentBID)
		userAgentID := agentA.ID
		oppName := agentB.Name
		if agentB.UserID == user.ID {
			userAgentID = agentB.ID
			oppName = agentA.Name
		}
		outcome := "–"
		if m.Status == "completed" {
			if !m.WinnerAgentID.Valid {
				outcome = "Draw"
			} else if m.WinnerAgentID.Int64 == userAgentID {
				outcome = "Win"
			} else {
				outcome = "Loss"
			}
		} else if m.Status == "failed" {
			outcome = "Failed"
		}
		rows = append(rows, matchRow{ID: m.ID, Status: m.Status, Outcome: outcome, OppName: oppName, DurMS: m.DurationMS})
	}
	s.render(w, r, "Practice", "practice", map[string]any{
		"CSRF":    session.CSRFToken,
		"Agents":  agents,
		"Code":    defaultCode,
		"Matches": rows,
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
	target := fmt.Sprintf("/matches/%d", matchID)
	if err != nil {
		if isHTMX(r) {
			w.Header().Set("HX-Redirect", target)
			return
		}
		http.Redirect(w, r, target, http.StatusSeeOther)
		return
	}
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", target)
		return
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (s *Server) practiceError(w http.ResponseWriter, r *http.Request, msg string) {
	session, _ := auth.CurrentSession(r)
	agents, _ := s.store.SystemAgents()
	s.render(w, r, "Practice", "practice", map[string]any{
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
	agentA, _ := s.store.AgentByID(m.AgentAID)
	agentB, _ := s.store.AgentByID(m.AgentBID)
	winnerName := "Draw"
	if m.WinnerAgentID.Valid {
		if m.WinnerAgentID.Int64 == agentA.ID {
			winnerName = agentA.Name
		} else {
			winnerName = agentB.Name
		}
	}
	outcome := ""
	if m.Status == "completed" {
		if !m.WinnerAgentID.Valid {
			outcome = "draw"
		} else if m.WinnerAgentID.Int64 == agentA.ID {
			outcome = "win"
		} else {
			outcome = "loss"
		}
	} else if m.Status == "failed" {
		outcome = "failed"
	}
	games, _ := s.store.ListGames(matchID)
	s.render(w, r, fmt.Sprintf("Match #%d", m.ID), "match", map[string]any{
		"Match":      m,
		"AgentA":     agentA,
		"AgentB":     agentB,
		"WinnerName": winnerName,
		"Outcome":    outcome,
		"Games":      games,
	})
}

func (s *Server) matchLogs(w http.ResponseWriter, r *http.Request) {
	matchID, ok := s.authorizedMatch(w, r)
	if !ok {
		return
	}
	content, err := s.store.ExecutionLog(matchID)
	hasLog := true
	if err != nil {
		if err == sql.ErrNoRows {
			hasLog = false
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	s.render(w, r, fmt.Sprintf("Match #%d Logs", matchID), "logs", map[string]any{
		"MatchID": matchID,
		"Content": content,
		"HasLog":  hasLog,
	})
}

type movePayload struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type moveView struct {
	Num      int
	Player   string
	Pos      string
	Accepted bool
	DurMS    sql.NullInt64
}

type gameReplayView struct {
	ID      int64
	Result  string
	Moves   []moveView
	Board   [8][8]string
	MovesJS template.JS
}

func buildReplayView(game store.Game, moves []store.Move, agentNames map[int64]string) gameReplayView {
	type jsMove struct {
		X      int    `json:"x"`
		Y      int    `json:"y"`
		Mark   string `json:"mark"`
		Player string `json:"player"`
	}
	var board [8][8]string
	var mvs []moveView
	var jsMoves []jsMove
	// first player in this game gets X, second gets O
	markFor := map[string]string{game.PlayerA: "X", game.PlayerB: "O"}
	for i, m := range moves {
		var p movePayload
		_ = json.Unmarshal([]byte(m.ActionPayload), &p)
		name := agentNames[m.AgentID]
		pos := ""
		mark := markFor[fmt.Sprintf("%d", m.AgentID)]
		if p.X > 0 && p.Y > 0 {
			pos = fmt.Sprintf("%d,%d", p.X, p.Y)
			if m.Accepted && p.X >= 1 && p.X <= 8 && p.Y >= 1 && p.Y <= 8 {
				board[p.Y-1][p.X-1] = mark
			}
		}
		mvs = append(mvs, moveView{Num: i + 1, Player: name, Pos: pos, Accepted: m.Accepted, DurMS: m.DurationMS})
		jsMoves = append(jsMoves, jsMove{X: p.X, Y: p.Y, Mark: mark, Player: name})
	}
	raw, _ := json.Marshal(jsMoves)
	return gameReplayView{ID: game.ID, Result: game.Result, Moves: mvs, Board: board, MovesJS: template.JS(raw)}
}

func (s *Server) matchReplay(w http.ResponseWriter, r *http.Request) {
	matchID, ok := s.authorizedMatch(w, r)
	if !ok {
		return
	}
	m, err := s.store.MatchByID(matchID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	agentA, _ := s.store.AgentByID(m.AgentAID)
	agentB, _ := s.store.AgentByID(m.AgentBID)
	agentNames := map[int64]string{agentA.ID: agentA.Name, agentB.ID: agentB.Name}
	games, err := s.store.ListGames(matchID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var replay []gameReplayView
	for _, game := range games {
		moves, _ := s.store.ListMoves(game.ID)
		replay = append(replay, buildReplayView(game, moves, agentNames))
	}
	s.render(w, r, fmt.Sprintf("Match #%d Replay", matchID), "replay", map[string]any{"MatchID": matchID, "Replay": replay})
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
			raw, readErr := io.ReadAll(file)
			closeErr := file.Close()
			if readErr != nil {
				return "", readErr
			}
			if closeErr != nil {
				return "", closeErr
			}
			return string(raw), nil
		}
	}
	if err := r.ParseForm(); err != nil {
		return "", err
	}
	return r.FormValue("source"), nil
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

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
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
