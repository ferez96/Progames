package web

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"progames/internal/auth"
	"progames/internal/store"
)

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
	switch m.Status {
	case "completed":
		if !m.WinnerAgentID.Valid {
			outcome = "draw"
		} else if m.WinnerAgentID.Int64 == agentA.ID {
			outcome = "win"
		} else {
			outcome = "loss"
		}
	case "failed":
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
		moves, err := s.store.ListMoves(game.ID)
		if err != nil {
			zap.L().Error("failed to list moves for game", zap.Int64("game_id", game.ID), zap.Error(err))
			continue
		}
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
