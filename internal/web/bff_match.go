package web

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"

	"progames/internal/service"
)

// ── Stone mark codes (wire format to the JS board renderer) ──────────────────

const (
	markBlack = "B"
	markWhite = "W"
)

// ── Result translation ────────────────────────────────────────────────────────

func resultLabel(r string) string {
	switch r {
	case service.ResultPlayerAWin:
		return "Black wins"
	case service.ResultPlayerBWin:
		return "White wins"
	case service.ResultDraw:
		return "Draw"
	default:
		return "Failed"
	}
}

func resultBadgeClass(r string) string {
	switch r {
	case service.ResultPlayerAWin:
		return "badge-blackwins"
	case service.ResultPlayerBWin:
		return "badge-whitewins"
	case service.ResultDraw:
		return "badge-draw"
	default:
		return "badge-failed"
	}
}

func resultHasTrophy(r string) bool {
	return r == service.ResultPlayerAWin || r == service.ResultPlayerBWin
}

// ── BFF page DTOs ─────────────────────────────────────────────────────────────

type summaryPageData struct {
	Match      service.Match
	AgentA     service.Agent
	AgentB     service.Agent
	Outcome    string // "win" | "loss" | "draw" | "failed" | ""
	WinnerName string
	Games      []gameRow
}

type logsPageData struct {
	MatchID int64
	Content string
	HasLog  bool
}

type gamesPageData struct {
	MatchID     int64
	Games       []gameRow
	InitialGame *gameView
}

// ── BFF display types ─────────────────────────────────────────────────────────

type gameRow struct {
	ID               int64
	ResultLabel      string
	ResultBadgeClass string
	BlackPlayer      string
	WhitePlayer      string
	MoveCount        int64
	DurationMS       int64
}

type gameView struct {
	ID               int64
	ResultLabel      string
	ResultBadgeClass string
	ResultHasTrophy  bool
	BlackPlayer      string
	WhitePlayer      string
	Moves            []moveView
	MovesJS          template.JS
}

type moveView struct {
	Num      int
	Player   string
	Pos      string
	Accepted bool
	DurMS    sql.NullInt64
}

// ── Builders ──────────────────────────────────────────────────────────────────

func toSummaryPage(m service.GetMatchResponse, g service.ListGamesResponse) summaryPageData {
	names := agentNameMap(m.AgentA, m.AgentB)
	rows := make([]gameRow, len(g.Games))
	for i, game := range g.Games {
		rows[i] = toGameRow(game, names)
	}
	return summaryPageData{
		Match:      m.Match,
		AgentA:     m.AgentA,
		AgentB:     m.AgentB,
		Outcome:    matchOutcome(m.Match, m.AgentA),
		WinnerName: matchWinnerName(m.Match, m.AgentA, m.AgentB),
		Games:      rows,
	}
}

func toGameRow(g service.Game, agentNames map[int64]string) gameRow {
	return gameRow{
		ID:               g.ID,
		ResultLabel:      resultLabel(g.Result),
		ResultBadgeClass: resultBadgeClass(g.Result),
		BlackPlayer:      agentNames[g.BlackAgent],
		WhitePlayer:      agentNames[g.WhiteAgent],
		MoveCount:        g.MoveCount,
		DurationMS:       g.DurationMS,
	}
}

func toGameView(d service.GetGameResponse) gameView {
	type movePayload struct {
		X int `json:"x"`
		Y int `json:"y"`
	}
	type jsMove struct {
		X      int    `json:"x"`
		Y      int    `json:"y"`
		Mark   string `json:"mark"`
		Player string `json:"player"`
	}

	names := agentNameMap(d.AgentA, d.AgentB)
	markFor := map[int64]string{
		d.Game.BlackAgent: markBlack,
		d.Game.WhiteAgent: markWhite,
	}

	var mvs []moveView
	var jsMoves []jsMove
	for i, m := range d.Moves {
		var p movePayload
		_ = json.Unmarshal([]byte(m.ActionPayload), &p)
		pos := ""
		if p.X > 0 && p.Y > 0 {
			pos = fmt.Sprintf("%d,%d", p.X, p.Y)
		}
		durMS := sql.NullInt64{}
		if m.DurationMS != nil {
			durMS = sql.NullInt64{Valid: true, Int64: *m.DurationMS}
		}
		mvs = append(mvs, moveView{
			Num:      i + 1,
			Player:   names[m.AgentID],
			Pos:      pos,
			Accepted: m.Accepted,
			DurMS:    durMS,
		})
		jsMoves = append(jsMoves, jsMove{
			X:      p.X,
			Y:      p.Y,
			Mark:   markFor[m.AgentID],
			Player: names[m.AgentID],
		})
	}
	raw, _ := json.Marshal(jsMoves)

	return gameView{
		ID:               d.Game.ID,
		ResultLabel:      resultLabel(d.Game.Result),
		ResultBadgeClass: resultBadgeClass(d.Game.Result),
		ResultHasTrophy:  resultHasTrophy(d.Game.Result),
		BlackPlayer:      names[d.Game.BlackAgent],
		WhitePlayer:      names[d.Game.WhiteAgent],
		Moves:            mvs,
		MovesJS:          template.JS(raw),
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func agentNameMap(agentA, agentB service.Agent) map[int64]string {
	return map[int64]string{agentA.ID: agentA.Name, agentB.ID: agentB.Name}
}

func matchOutcome(m service.Match, agentA service.Agent) string {
	switch m.Status {
	case "completed":
		if m.WinnerAgent == nil {
			return "draw"
		}
		if *m.WinnerAgent == agentA.ID {
			return "win"
		}
		return "loss"
	case "failed":
		return "failed"
	}
	return ""
}

func matchWinnerName(m service.Match, agentA, agentB service.Agent) string {
	if m.WinnerAgent == nil {
		return "Draw"
	}
	if *m.WinnerAgent == agentA.ID {
		return agentA.Name
	}
	return agentB.Name
}
