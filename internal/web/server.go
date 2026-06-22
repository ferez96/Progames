package web

import (
	"expvar"
	"html/template"
	"net/http"

	"github.com/go-chi/chi/v5"

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
	Enqueue(userAgentID, systemAgentID int64) (int64, error)
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
		templates:  newTemplates(),
	}
}

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(requestLogger)
	r.Mount("/", s.feRoutes())
	r.Mount("/api", s.apiRoutes())
	return r
}

func (s *Server) feRoutes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", s.home)
	r.Get("/signup", s.signupForm)
	r.Get("/login", s.loginForm)
	r.Group(func(r chi.Router) {
		r.Use(s.auth.Require)
		r.Get("/practice", s.practice)
		r.Get("/matches/{id}", s.matchSummary)
		r.Get("/matches/{id}/logs", s.matchLogs)
		r.Get("/matches/{id}/replay", s.matchReplay)
	})
	return r
}

func (s *Server) apiRoutes() http.Handler {
	r := chi.NewRouter()
	r.Method("GET", "/debug/vars", expvar.Handler())
	r.Post("/signup", s.signup)
	r.Post("/login", s.login)
	r.Group(func(r chi.Router) {
		r.Use(s.auth.Require)
		r.Post("/logout", s.logout)
		r.Post("/practice/run", s.runPractice)
	})
	return r
}
