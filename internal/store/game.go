package store

import (
	"database/sql"
	"time"
)

type Game struct {
	ID         int64     `db:"id"`
	MatchID    int64     `db:"match_id"`
	PlayerA    string    `db:"player_a"`
	PlayerB    string    `db:"player_b"`
	Result     string    `db:"result"`
	DurationMS int64     `db:"duration_ms"`
	MoveCount  int64     `db:"move_count"`
	CreatedAt  time.Time `db:"created_at"`
}

type Move struct {
	ID            int64         `db:"id"`
	GameID        int64         `db:"game_id"`
	Seq           int64         `db:"seq"`
	AgentID       int64         `db:"agent_id"`
	ActionType    string        `db:"action_type"`
	ActionPayload string        `db:"action_payload"`
	Accepted      bool          `db:"accepted"`
	DurationMS    sql.NullInt64 `db:"duration_ms"`
	CreatedAt     time.Time     `db:"created_at"`
}

type AgentLog struct {
	ID        int64     `db:"id"`
	MatchID   int64     `db:"match_id"`
	AgentID   int64     `db:"agent_id"`
	Content   string    `db:"content"`
	Truncated bool      `db:"truncated"`
	CreatedAt time.Time `db:"created_at"`
}

func (s *Store) CreateGame(matchID int64, playerA, playerB string) (int64, error) {
	res, err := s.DB.Exec(`INSERT INTO games (match_id, player_a, player_b) VALUES (?, ?, ?)`, matchID, playerA, playerB)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) FinishGame(id int64, result string, durationMS, moveCount int64) error {
	_, err := s.DB.Exec(`UPDATE games SET result = ?, duration_ms = ?, move_count = ? WHERE id = ?`, result, durationMS, moveCount, id)
	return err
}

func (s *Store) ListGames(matchID int64) ([]Game, error) {
	var games []Game
	err := s.DB.Select(&games, `SELECT * FROM games WHERE match_id = ? ORDER BY id`, matchID)
	return games, err
}

func (s *Store) InsertMove(gameID, seq, agentID int64, actionType, payload string, accepted bool, durationMS sql.NullInt64) error {
	_, err := s.DB.Exec(`INSERT OR IGNORE INTO moves (game_id, seq, agent_id, action_type, action_payload, accepted, duration_ms)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, gameID, seq, agentID, actionType, payload, accepted, durationMS)
	return err
}

func (s *Store) ListMoves(gameID int64) ([]Move, error) {
	var moves []Move
	err := s.DB.Select(&moves, `SELECT * FROM moves WHERE game_id = ? ORDER BY seq`, gameID)
	return moves, err
}

func (s *Store) UpsertAgentLog(matchID, agentID int64, content string, truncated bool) error {
	_, err := s.DB.Exec(`INSERT INTO agent_logs (match_id, agent_id, content, truncated)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(match_id, agent_id) DO UPDATE SET content = agent_logs.content || excluded.content, truncated = excluded.truncated`,
		matchID, agentID, content, truncated)
	return err
}

func (s *Store) AgentLogs(matchID int64) ([]AgentLog, error) {
	var logs []AgentLog
	err := s.DB.Select(&logs, `SELECT * FROM agent_logs WHERE match_id = ? ORDER BY agent_id`, matchID)
	return logs, err
}

func (s *Store) UpsertExecutionLog(matchID int64, content string, truncated bool) error {
	_, err := s.DB.Exec(`INSERT INTO execution_logs (match_id, content, truncated)
		VALUES (?, ?, ?)
		ON CONFLICT(match_id) DO UPDATE SET content = excluded.content, truncated = excluded.truncated`, matchID, content, truncated)
	return err
}

func (s *Store) ExecutionLog(matchID int64) (string, error) {
	var content string
	err := s.DB.Get(&content, `SELECT content FROM execution_logs WHERE match_id = ?`, matchID)
	return content, err
}
