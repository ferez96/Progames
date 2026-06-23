package web

import (
	"database/sql"

	"progames/internal/service"
)

// ── BFF page DTO ──────────────────────────────────────────────────────────────

type practicePageData struct {
	CSRF    string
	Agents  []service.Agent
	Code    string
	Matches []practiceMatchRow
	Error   string
}

type practiceMatchRow struct {
	ID      int64
	Status  string
	Outcome string // "Win" | "Loss" | "Draw" | "Failed" | "–"
	OppName string
	DurMS   sql.NullInt64
}

// ── Builders ──────────────────────────────────────────────────────────────────

func toPracticeMatchRow(e service.MatchEntry, userID int64) practiceMatchRow {
	userAgentID := e.AgentA.ID
	oppName := e.AgentB.Name
	if e.AgentB.UserID == userID {
		userAgentID = e.AgentB.ID
		oppName = e.AgentA.Name
	}
	outcome := "–"
	switch e.Match.Status {
	case "completed":
		if e.Match.WinnerAgent == nil {
			outcome = "Draw"
		} else if *e.Match.WinnerAgent == userAgentID {
			outcome = "Win"
		} else {
			outcome = "Loss"
		}
	case "failed":
		outcome = "Failed"
	}
	durMS := sql.NullInt64{}
	if e.Match.DurationMS > 0 {
		durMS = sql.NullInt64{Valid: true, Int64: e.Match.DurationMS}
	}
	return practiceMatchRow{
		ID:      e.Match.ID,
		Status:  e.Match.Status,
		Outcome: outcome,
		OppName: oppName,
		DurMS:   durMS,
	}
}

func toPracticePage(resp service.GetPracticeDataResponse, userID int64, csrf, code, errMsg string) practicePageData {
	rows := make([]practiceMatchRow, len(resp.Matches))
	for i, e := range resp.Matches {
		rows[i] = toPracticeMatchRow(e, userID)
	}
	return practicePageData{
		CSRF:    csrf,
		Agents:  resp.Opponents,
		Code:    code,
		Matches: rows,
		Error:   errMsg,
	}
}
