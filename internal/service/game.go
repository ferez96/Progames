package service

import (
	"progames/internal/store"
)

type gameStore interface {
	MatchByID(id int64) (store.Match, error)
	AgentByID(id int64) (store.Agent, error)
	GameByID(id int64) (store.Game, error)
	ListGames(matchID int64) ([]store.Game, error)
	ListMoves(gameID int64) ([]store.Move, error)
	UserCanViewMatch(userID, matchID int64) (bool, error)
}

// ── Request / Response ────────────────────────────────────────────────────────

type ListGamesRequest struct {
	UserID  int64
	MatchID int64
}

type ListGamesResponse struct {
	AgentA Agent
	AgentB Agent
	Games  []Game
}

type GetGameRequest struct {
	UserID  int64
	MatchID int64
	GameID  int64
}

type GetGameResponse struct {
	AgentA Agent
	AgentB Agent
	Game   Game
	Moves  []Move
}

// ── Service ───────────────────────────────────────────────────────────────────

type GameService struct {
	store gameStore
}

func NewGame(st gameStore) *GameService {
	return &GameService{store: st}
}

func (s *GameService) authorized(userID, matchID int64) error {
	ok, err := s.store.UserCanViewMatch(userID, matchID)
	if err != nil || !ok {
		return ErrNotFound
	}
	return nil
}

func (s *GameService) ListGames(req ListGamesRequest) (ListGamesResponse, error) {
	if err := s.authorized(req.UserID, req.MatchID); err != nil {
		return ListGamesResponse{}, err
	}
	m, err := s.store.MatchByID(req.MatchID)
	if err != nil {
		return ListGamesResponse{}, ErrNotFound
	}
	agentA, _ := s.store.AgentByID(m.AgentAID)
	agentB, _ := s.store.AgentByID(m.AgentBID)
	rawGames, err := s.store.ListGames(req.MatchID)
	if err != nil {
		return ListGamesResponse{}, err
	}
	games := make([]Game, len(rawGames))
	for i, g := range rawGames {
		games[i] = gameFrom(g)
	}
	return ListGamesResponse{AgentA: agentFrom(agentA), AgentB: agentFrom(agentB), Games: games}, nil
}

func (s *GameService) GetGame(req GetGameRequest) (GetGameResponse, error) {
	if err := s.authorized(req.UserID, req.MatchID); err != nil {
		return GetGameResponse{}, err
	}
	game, err := s.store.GameByID(req.GameID)
	if err != nil || game.MatchID != req.MatchID {
		return GetGameResponse{}, ErrNotFound
	}
	m, err := s.store.MatchByID(req.MatchID)
	if err != nil {
		return GetGameResponse{}, ErrNotFound
	}
	agentA, _ := s.store.AgentByID(m.AgentAID)
	agentB, _ := s.store.AgentByID(m.AgentBID)
	rawMoves, err := s.store.ListMoves(req.GameID)
	if err != nil {
		return GetGameResponse{}, err
	}
	moves := make([]Move, len(rawMoves))
	for i, mv := range rawMoves {
		moves[i] = moveFrom(mv)
	}
	return GetGameResponse{
		AgentA: agentFrom(agentA), AgentB: agentFrom(agentB),
		Game: gameFrom(game), Moves: moves,
	}, nil
}
