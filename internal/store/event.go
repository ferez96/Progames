package store

import (
	"database/sql"
	"time"
)

type Event struct {
	ID        int64         `db:"id"`
	MatchID   int64         `db:"match_id"`
	GameID    sql.NullInt64 `db:"game_id"`
	Seq       int64         `db:"seq"`
	Type      string        `db:"type"`
	Payload   string        `db:"payload"`
	CreatedAt time.Time     `db:"created_at"`
}

func (s *Store) NextEventSeq(matchID int64) (int64, error) {
	var seq sql.NullInt64
	if err := s.DB.Get(&seq, `SELECT MAX(seq) FROM events WHERE match_id = ?`, matchID); err != nil {
		return 0, err
	}
	if !seq.Valid {
		return 1, nil
	}
	return seq.Int64 + 1, nil
}

func (s *Store) AppendEvent(matchID int64, gameID sql.NullInt64, typ, payload string) error {
	seq, err := s.NextEventSeq(matchID)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(`INSERT OR IGNORE INTO events (match_id, game_id, seq, type, payload) VALUES (?, ?, ?, ?, ?)`,
		matchID, gameID, seq, typ, payload)
	return err
}

func (s *Store) ListEvents(matchID int64) ([]Event, error) {
	var events []Event
	err := s.DB.Select(&events, `SELECT * FROM events WHERE match_id = ? ORDER BY seq`, matchID)
	return events, err
}
