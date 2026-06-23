package service

import (
	"strconv"

	"progames/internal/store"
)

// ── Queue messages ────────────────────────────────────────────────────────────

// MatchJob is the input message submitted to the match queue.
// Defined here so the service layer can reference it without importing the match package.
type MatchJob struct {
	UserAgentID   int64
	SystemAgentID int64
}

// ── Game result codes ─────────────────────────────────────────────────────────
// Written by match runner, read by the BFF translation layer.
// Both sides must use these constants so a rename is caught at compile time.
const (
	ResultPlayerAWin = "player_a_win"
	ResultPlayerBWin = "player_b_win"
	ResultDraw       = "draw"
)

// ── Service domain types ──────────────────────────────────────────────────────
// These are the service layer's own representations of domain concepts.
// They are independent of storage layout (no db tags, no sql.Null* types,
// no storage-specific string encodings like PlayerA/PlayerB).

type Agent struct {
	ID     int64
	Name   string
	UserID int64
	Type   string // "user" | "system"
}

type Match struct {
	ID          int64
	Status      string
	WinnerAgent *int64 // nil when no winner determined
	ErrorMsg    string
	DurationMS  int64 // 0 when not measured
}

type Game struct {
	ID         int64
	MatchID    int64
	BlackAgent int64 // agent ID of first mover
	WhiteAgent int64 // agent ID of second mover
	Result     string
	MoveCount  int64
	DurationMS int64
}

type Move struct {
	Seq           int64
	AgentID       int64
	ActionPayload string
	Accepted      bool
	DurationMS    *int64 // nil when not measured
}

// ── Store → service converters ────────────────────────────────────────────────

func agentFrom(a store.Agent) Agent {
	return Agent{ID: a.ID, Name: a.Name, UserID: a.UserID, Type: a.Type}
}

func matchFrom(m store.Match) Match {
	sm := Match{
		ID:       m.ID,
		Status:   m.Status,
		ErrorMsg: m.ErrorMsg,
	}
	if m.WinnerAgentID.Valid {
		sm.WinnerAgent = &m.WinnerAgentID.Int64
	}
	if m.DurationMS.Valid {
		sm.DurationMS = m.DurationMS.Int64
	}
	return sm
}

func gameFrom(g store.Game) Game {
	return Game{
		ID:         g.ID,
		MatchID:    g.MatchID,
		BlackAgent: parseAgentID(g.PlayerA),
		WhiteAgent: parseAgentID(g.PlayerB),
		Result:     g.Result,
		MoveCount:  g.MoveCount,
		DurationMS: g.DurationMS,
	}
}

func moveFrom(m store.Move) Move {
	mv := Move{
		Seq:           m.Seq,
		AgentID:       m.AgentID,
		ActionPayload: m.ActionPayload,
		Accepted:      m.Accepted,
	}
	if m.DurationMS.Valid {
		mv.DurationMS = &m.DurationMS.Int64
	}
	return mv
}

func parseAgentID(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}
