package service

import (
	"database/sql"
	"errors"

	"progames/internal/store"
)

var ErrNotFound = errors.New("not found")

type matchStore interface {
	MatchByID(id int64) (store.Match, error)
	AgentByID(id int64) (store.Agent, error)
	ExecutionLog(matchID int64) (string, error)
	UserCanViewMatch(userID, matchID int64) (bool, error)
}

// ── Request / Response ────────────────────────────────────────────────────────

type GetMatchRequest struct {
	UserID  int64
	MatchID int64
}

type GetMatchResponse struct {
	Match  Match
	AgentA Agent
	AgentB Agent
}

type GetExecutionLogRequest struct {
	UserID  int64
	MatchID int64
}

type GetExecutionLogResponse struct {
	Content string // empty when no log exists
}

// ── Service ───────────────────────────────────────────────────────────────────

type MatchService struct {
	store matchStore
}

func NewMatch(st matchStore) *MatchService {
	return &MatchService{store: st}
}

func (s *MatchService) authorized(userID, matchID int64) error {
	ok, err := s.store.UserCanViewMatch(userID, matchID)
	if err != nil || !ok {
		return ErrNotFound
	}
	return nil
}

func (s *MatchService) GetMatch(req GetMatchRequest) (GetMatchResponse, error) {
	if err := s.authorized(req.UserID, req.MatchID); err != nil {
		return GetMatchResponse{}, err
	}
	m, err := s.store.MatchByID(req.MatchID)
	if err != nil {
		return GetMatchResponse{}, ErrNotFound
	}
	agentA, _ := s.store.AgentByID(m.AgentAID)
	agentB, _ := s.store.AgentByID(m.AgentBID)
	return GetMatchResponse{Match: matchFrom(m), AgentA: agentFrom(agentA), AgentB: agentFrom(agentB)}, nil
}

func (s *MatchService) GetExecutionLog(req GetExecutionLogRequest) (GetExecutionLogResponse, error) {
	if err := s.authorized(req.UserID, req.MatchID); err != nil {
		return GetExecutionLogResponse{}, err
	}
	content, err := s.store.ExecutionLog(req.MatchID)
	if errors.Is(err, sql.ErrNoRows) {
		return GetExecutionLogResponse{}, nil
	}
	return GetExecutionLogResponse{Content: content}, err
}
