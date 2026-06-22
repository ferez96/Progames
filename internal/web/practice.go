package web

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"progames/internal/auth"
)

type matchRow struct {
	ID      int64
	Status  string
	Outcome string // "Win" | "Loss" | "Draw" | "–"
	OppName string
	DurMS   sql.NullInt64
}

func (s *Server) practice(w http.ResponseWriter, r *http.Request) {
	session, _ := auth.CurrentSession(r)
	user, _ := auth.CurrentUser(r)
	agents, err := s.store.SystemAgents()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	matches, _ := s.store.ListUserMatches(user.ID)
	var rows []matchRow
	for _, m := range matches {
		agentA, _ := s.store.AgentByID(m.AgentAID)
		agentB, _ := s.store.AgentByID(m.AgentBID)
		userAgentID := agentA.ID
		oppName := agentB.Name
		if agentB.UserID == user.ID {
			userAgentID = agentB.ID
			oppName = agentA.Name
		}
		outcome := "–"
		switch m.Status {
		case "completed":
			if !m.WinnerAgentID.Valid {
				outcome = "Draw"
			} else if m.WinnerAgentID.Int64 == userAgentID {
				outcome = "Win"
			} else {
				outcome = "Loss"
			}
		case "failed":
			outcome = "Failed"
		}
		rows = append(rows, matchRow{ID: m.ID, Status: m.Status, Outcome: outcome, OppName: oppName, DurMS: m.DurationMS})
	}
	s.render(w, r, "Practice", "practice", map[string]any{
		"CSRF":    session.CSRFToken,
		"Agents":  agents,
		"Code":    defaultCode,
		"Matches": rows,
	})
}

func (s *Server) runPractice(w http.ResponseWriter, r *http.Request) {
	if !auth.ValidateCSRF(r) {
		http.Error(w, "invalid csrf token", http.StatusForbidden)
		return
	}
	user, _ := auth.CurrentUser(r)
	code, err := readCode(r)
	if err != nil {
		s.practiceError(w, r, err.Error())
		return
	}
	res, err := s.submission.Submit(user.ID, code)
	if err != nil {
		s.practiceError(w, r, err.Error())
		return
	}
	if res.Status != "compiled" {
		s.practiceError(w, r, fmt.Sprintf("%s\n%s", res.Message, res.Output))
		return
	}
	opponentID, err := strconv.ParseInt(r.FormValue("opponent_agent_id"), 10, 64)
	if err != nil {
		s.practiceError(w, r, "select a system opponent")
		return
	}
	opponent, err := s.store.AgentByID(opponentID)
	if err != nil || opponent.Type != "system" {
		s.practiceError(w, r, "invalid system opponent")
		return
	}
	matchID, err := s.match.Enqueue(res.AgentID, opponentID)
	if err != nil && matchID == 0 {
		s.practiceError(w, r, err.Error())
		return
	}
	target := fmt.Sprintf("/matches/%d", matchID)
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", target)
		return
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (s *Server) practiceError(w http.ResponseWriter, r *http.Request, msg string) {
	session, _ := auth.CurrentSession(r)
	agents, _ := s.store.SystemAgents()
	s.render(w, r, "Practice", "practice", map[string]any{
		"CSRF":   session.CSRFToken,
		"Agents": agents,
		"Code":   defaultCode,
		"Error":  msg,
	})
}

func readCode(r *http.Request) (string, error) {
	if err := r.ParseMultipartForm(1 << 20); err == nil {
		file, _, err := r.FormFile("source_file")
		if err == nil {
			raw, readErr := io.ReadAll(file)
			closeErr := file.Close()
			if readErr != nil {
				return "", readErr
			}
			if closeErr != nil {
				return "", closeErr
			}
			return string(raw), nil
		}
	}
	if err := r.ParseForm(); err != nil {
		return "", err
	}
	return r.FormValue("source"), nil
}

const defaultCode = `package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	in := bufio.NewScanner(os.Stdin)
	for in.Scan() {
		state := in.Text()
		for i, cell := range state {
			if cell == '.' {
				fmt.Printf("%d,%d\n", i%8+1, i/8+1)
				os.Stdout.Sync()
				break
			}
		}
		_ = strings.TrimSpace(state)
	}
}
`
