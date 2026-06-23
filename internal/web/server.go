package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	fe *Frontend
}

func New(authSvc authService, practiceSvc practiceService, matchSvc matchService, gameSvc gameService) *Server {
	return &Server{fe: newFrontend(authSvc, practiceSvc, matchSvc, gameSvc)}
}

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(requestLogger)
	r.Mount("/", s.fe.pageRoutes())
	r.Mount("/api", s.fe.apiRoutes())
	return r
}
