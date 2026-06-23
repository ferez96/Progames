package store

import (
	"database/sql"
	"time"
)

type Match struct {
	ID            int64         `db:"id"`
	TournamentID  sql.NullInt64 `db:"tournament_id"`
	AgentAID      int64         `db:"agent_a_id"`
	AgentBID      int64         `db:"agent_b_id"`
	Status        string        `db:"status"`
	WinnerAgentID sql.NullInt64 `db:"winner_agent_id"`
	ErrorMsg      string        `db:"error_msg"`
	StartedAt     sql.NullTime  `db:"started_at"`
	EndedAt       sql.NullTime  `db:"ended_at"`
	DurationMS    sql.NullInt64 `db:"duration_ms"`
}

func (s *Store) CreateMatch(agentAID, agentBID int64) (int64, error) {
	res, err := s.DB.Exec(`INSERT INTO matches (agent_a_id, agent_b_id, status) VALUES (?, ?, 'queued')`, agentAID, agentBID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) StartMatch(id int64, startedAt time.Time) error {
	_, err := s.DB.Exec(`UPDATE matches SET status = 'running', started_at = ?
		WHERE id = ? AND status = 'queued'`, startedAt, id)
	return err
}

func (s *Store) CompleteMatch(id int64, winnerAgentID sql.NullInt64, endedAt time.Time, durationMS int64) error {
	_, err := s.DB.Exec(`UPDATE matches SET status = 'completed', winner_agent_id = ?, ended_at = ?, duration_ms = ?
		WHERE id = ? AND status = 'running'`, winnerAgentID, endedAt, durationMS, id)
	return err
}

func (s *Store) FailMatch(id int64, msg string, endedAt time.Time, durationMS int64) error {
	_, err := s.DB.Exec(`UPDATE matches SET status = 'failed', error_msg = ?, ended_at = ?, duration_ms = ?
		WHERE id = ? AND status NOT IN ('completed', 'failed')`, msg, endedAt, durationMS, id)
	return err
}

func (s *Store) MatchByID(id int64) (Match, error) {
	var match Match
	err := s.DB.Get(&match, `SELECT * FROM matches WHERE id = ?`, id)
	return match, err
}

func (s *Store) ListUserMatches(userID int64) ([]Match, error) {
	var matches []Match
	err := s.DB.Select(&matches, `
		SELECT * FROM matches
		WHERE agent_a_id IN (SELECT id FROM agents WHERE user_id = ?)
		   OR agent_b_id IN (SELECT id FROM agents WHERE user_id = ?)
		ORDER BY id DESC LIMIT 10`, userID, userID)
	return matches, err
}

func (s *Store) UserCanViewMatch(userID, matchID int64) (bool, error) {
	var count int
	err := s.DB.Get(&count, `SELECT COUNT(*)
		FROM matches m
		JOIN agents a ON a.id = m.agent_a_id
		WHERE m.id = ? AND a.user_id = ?`, matchID, userID)
	return count > 0, err
}
