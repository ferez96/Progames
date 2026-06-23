package web

import (
	"expvar"
	"html/template"
	"net/http"

	"context"

	"github.com/go-chi/chi/v5"

	"progames/internal/auth"
	"progames/internal/service"
)

type authService interface {
	UserFromRequest(r *http.Request) (auth.User, auth.Session, error)
	SignUp(name, email, password string) (int64, error)
	SignIn(email, password string) (auth.User, string, string, error)
	SignOut(sessionID string) error
	SetSessionCookie(w http.ResponseWriter, r *http.Request, sessionID string)
}

type practiceService interface {
	GetPracticeData(req service.GetPracticeDataRequest) (service.GetPracticeDataResponse, error)
	RunMatch(ctx context.Context, req service.RunMatchRequest) (service.RunMatchResponse, error)
}

type matchService interface {
	GetMatch(req service.GetMatchRequest) (service.GetMatchResponse, error)
	GetExecutionLog(req service.GetExecutionLogRequest) (service.GetExecutionLogResponse, error)
}

type gameService interface {
	ListGames(req service.ListGamesRequest) (service.ListGamesResponse, error)
	GetGame(req service.GetGameRequest) (service.GetGameResponse, error)
}

type Frontend struct {
	authSvc     authService
	practiceSvc practiceService
	matchSvc    matchService
	gameSvc     gameService
	templates   *template.Template
}

func newFrontend(auth authService, practice practiceService, match matchService, game gameService) *Frontend {
	return &Frontend{
		authSvc:     auth,
		practiceSvc: practice,
		matchSvc:    match,
		gameSvc:     game,
		templates:   newTemplates(),
	}
}

func (fe *Frontend) pageRoutes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", fe.home)
	r.Get("/signup", fe.signupForm)
	r.Get("/login", fe.loginForm)
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireLogin(fe.authSvc))
		r.Get("/practice", fe.practice)
		r.Get("/matches/{id}", fe.matchSummary)
		r.Get("/matches/{id}/logs", fe.matchLogs)
		r.Get("/matches/{id}/games", fe.matchGames)
		r.Get("/matches/{id}/games/{gid}/replay", fe.matchGameReplay)
	})
	return r
}

func (fe *Frontend) apiRoutes() http.Handler {
	r := chi.NewRouter()
	r.Method("GET", "/debug/vars", expvar.Handler())
	r.Post("/signup", fe.signup)
	r.Post("/login", fe.login)
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireLogin(fe.authSvc))
		r.Post("/logout", fe.logout)
		r.Post("/practice/run", fe.runPractice)
	})
	return r
}
