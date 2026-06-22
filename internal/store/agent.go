package store

import (
	"database/sql"
	"time"
)

type Agent struct {
	ID           int64         `db:"id"`
	UserID       int64         `db:"user_id"`
	SubmissionID sql.NullInt64 `db:"submission_id"`
	Name         string        `db:"name"`
	Type         string        `db:"type"`
	Status       string        `db:"status"`
	CreatedAt    time.Time     `db:"created_at"`
}

func (s *Store) CreateAgent(userID, submissionID int64, name string) (int64, error) {
	res, err := s.DB.Exec(`INSERT INTO agents (user_id, submission_id, name, type, status) VALUES (?, ?, ?, 'user', 'active')`,
		userID, submissionID, name)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) AgentByID(id int64) (Agent, error) {
	var agent Agent
	err := s.DB.Get(&agent, `SELECT * FROM agents WHERE id = ?`, id)
	return agent, err
}

func (s *Store) SystemAgents() ([]Agent, error) {
	var agents []Agent
	err := s.DB.Select(&agents, `SELECT * FROM agents WHERE type = 'system' AND status = 'active' ORDER BY id`)
	return agents, err
}

func (s *Store) UserOwnsAgent(userID, agentID int64) (bool, error) {
	var count int
	err := s.DB.Get(&count, `SELECT COUNT(*) FROM agents WHERE id = ? AND user_id = ?`, agentID, userID)
	return count > 0, err
}
