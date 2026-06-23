package service

import (
	"context"
	"fmt"

	"progames/internal/store"
	"progames/internal/submission"
)

type practiceStore interface {
	SystemAgents() ([]store.Agent, error)
	ListUserMatches(userID int64) ([]store.Match, error)
	AgentByID(id int64) (store.Agent, error)
}

type submitter interface {
	Submit(userID int64, code string) (submission.Result, error)
}

type matchQueue interface {
	Enqueue(ctx context.Context, job MatchJob) (int64, error)
}

// MatchEntry is a match together with its two agents.
type MatchEntry struct {
	Match  Match
	AgentA Agent
	AgentB Agent
}

// ── Request / Response ────────────────────────────────────────────────────────

type GetPracticeDataRequest struct {
	UserID int64
}

type GetPracticeDataResponse struct {
	Opponents []Agent
	Matches   []MatchEntry
}

type RunMatchRequest struct {
	UserID     int64
	Code       string
	OpponentID int64
}

type RunMatchResponse struct {
	MatchID int64
}

// ── Service ───────────────────────────────────────────────────────────────────

type PracticeService struct {
	store practiceStore
	sub   submitter
	queue matchQueue
}

func NewPractice(st practiceStore, sub submitter, queue matchQueue) *PracticeService {
	return &PracticeService{store: st, sub: sub, queue: queue}
}

func (s *PracticeService) GetPracticeData(req GetPracticeDataRequest) (GetPracticeDataResponse, error) {
	rawOpponents, err := s.store.SystemAgents()
	if err != nil {
		return GetPracticeDataResponse{}, err
	}
	opponents := make([]Agent, len(rawOpponents))
	for i, a := range rawOpponents {
		opponents[i] = agentFrom(a)
	}
	matches, _ := s.store.ListUserMatches(req.UserID)
	entries := make([]MatchEntry, 0, len(matches))
	for _, m := range matches {
		agentA, _ := s.store.AgentByID(m.AgentAID)
		agentB, _ := s.store.AgentByID(m.AgentBID)
		entries = append(entries, MatchEntry{Match: matchFrom(m), AgentA: agentFrom(agentA), AgentB: agentFrom(agentB)})
	}
	return GetPracticeDataResponse{Opponents: opponents, Matches: entries}, nil
}

func (s *PracticeService) RunMatch(ctx context.Context, req RunMatchRequest) (RunMatchResponse, error) {
	res, err := s.sub.Submit(req.UserID, req.Code)
	if err != nil {
		return RunMatchResponse{}, err
	}
	if res.Status != "compiled" {
		return RunMatchResponse{}, fmt.Errorf("%s\n%s", res.Message, res.Output)
	}
	opponent, err := s.store.AgentByID(req.OpponentID)
	if err != nil || opponent.Type != "system" {
		return RunMatchResponse{}, fmt.Errorf("invalid system opponent")
	}
	matchID, err := s.queue.Enqueue(ctx, MatchJob{UserAgentID: res.AgentID, SystemAgentID: req.OpponentID})
	if err != nil && matchID == 0 {
		return RunMatchResponse{}, err
	}
	return RunMatchResponse{MatchID: matchID}, nil
}
