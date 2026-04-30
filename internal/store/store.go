package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"progames/internal/config"
)

type Store struct {
	DB          *sqlx.DB
	ArtifactDir string
}

type User struct {
	ID           int64     `db:"id"`
	Name         string    `db:"name"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
	PasswordSalt string    `db:"password_salt"`
	CreatedAt    time.Time `db:"created_at"`
}

type Session struct {
	ID        string    `db:"id"`
	UserID    int64     `db:"user_id"`
	CSRFToken string    `db:"csrf_token"`
	CreatedAt time.Time `db:"created_at"`
	ExpiresAt time.Time `db:"expires_at"`
}

type SourceCode struct {
	ID          string    `db:"id"`
	Language    string    `db:"language"`
	StorageType string    `db:"storage_type"`
	StorageKey  string    `db:"storage_key"`
	Size        int64     `db:"size"`
	CreatedAt   time.Time `db:"created_at"`
}

type Submission struct {
	ID            int64          `db:"id"`
	UserID        int64          `db:"user_id"`
	SourceCodeID  string         `db:"source_code_id"`
	Status        string         `db:"status"`
	Msg           string         `db:"msg"`
	CompileOutput string         `db:"compile_output"`
	BinaryPath    sql.NullString `db:"binary_path"`
	CreatedAt     time.Time      `db:"created_at"`
}

type Agent struct {
	ID           int64         `db:"id"`
	UserID       int64         `db:"user_id"`
	SubmissionID sql.NullInt64 `db:"submission_id"`
	Name         string        `db:"name"`
	Type         string        `db:"type"`
	Status       string        `db:"status"`
	CreatedAt    time.Time     `db:"created_at"`
}

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

type Event struct {
	ID        int64         `db:"id"`
	MatchID   int64         `db:"match_id"`
	GameID    sql.NullInt64 `db:"game_id"`
	Seq       int64         `db:"seq"`
	Type      string        `db:"type"`
	Payload   string        `db:"payload"`
	CreatedAt time.Time     `db:"created_at"`
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

func Open(cfg config.Config) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(cfg.ArtifactDir, "sources"), 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(cfg.ArtifactDir, "bins"), 0o755); err != nil {
		return nil, err
	}
	db, err := sqlx.Open("sqlite", cfg.DBPath)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{DB: db, ArtifactDir: cfg.ArtifactDir}
	if err := s.Init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	if s == nil || s.DB == nil {
		return nil
	}
	return s.DB.Close()
}

func (s *Store) Init() error {
	for _, stmt := range schemaStatements {
		if _, err := s.DB.Exec(stmt); err != nil {
			return err
		}
	}
	_, _ = s.DB.Exec(`DELETE FROM sessions WHERE expires_at <= datetime('now')`)
	return s.SeedSystemAgent()
}

func (s *Store) SeedSystemAgent() error {
	const email = "system@progames.internal"
	var userID int64
	err := s.DB.Get(&userID, `SELECT id FROM users WHERE email = ?`, email)
	if errors.Is(err, sql.ErrNoRows) {
		res, err := s.DB.Exec(`INSERT INTO users (name, email, password_hash, password_salt) VALUES (?, ?, ?, ?)`,
			"System", email, "*", "")
		if err != nil {
			return err
		}
		userID, err = res.LastInsertId()
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	var count int
	if err := s.DB.Get(&count, `SELECT COUNT(*) FROM agents WHERE type = 'system' AND status = 'active'`); err != nil {
		return err
	}
	if count == 0 {
		_, err = s.DB.Exec(`INSERT INTO agents (user_id, submission_id, name, type, status) VALUES (?, NULL, ?, 'system', 'active')`,
			userID, "First Empty Cell")
		return err
	}
	return nil
}

func (s *Store) CreateUser(name, email, hash, salt string) (int64, error) {
	res, err := s.DB.Exec(`INSERT INTO users (name, email, password_hash, password_salt) VALUES (?, ?, ?, ?)`,
		name, strings.ToLower(email), hash, salt)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) UserByEmail(email string) (User, error) {
	var user User
	err := s.DB.Get(&user, `SELECT * FROM users WHERE email = ?`, strings.ToLower(email))
	return user, err
}

func (s *Store) UserByID(id int64) (User, error) {
	var user User
	err := s.DB.Get(&user, `SELECT * FROM users WHERE id = ?`, id)
	return user, err
}

func (s *Store) CreateSession(id string, userID int64, csrf string, expiresAt time.Time) error {
	_, err := s.DB.Exec(`INSERT INTO sessions (id, user_id, csrf_token, expires_at) VALUES (?, ?, ?, ?)`,
		id, userID, csrf, expiresAt.UTC())
	return err
}

func (s *Store) SessionByID(id string) (Session, error) {
	var session Session
	err := s.DB.Get(&session, `SELECT * FROM sessions WHERE id = ? AND expires_at > datetime('now')`, id)
	return session, err
}

func (s *Store) DeleteSession(id string) error {
	_, err := s.DB.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	return err
}

func (s *Store) CreateSourceCode(id, storageKey string, size int64) error {
	_, err := s.DB.Exec(`INSERT INTO source_codes (id, language, storage_type, storage_key, size) VALUES (?, 'Go', 'local', ?, ?)`,
		id, storageKey, size)
	return err
}

func (s *Store) CreateSubmission(userID int64, sourceID string) (int64, error) {
	res, err := s.DB.Exec(`INSERT INTO submissions (user_id, source_code_id, status) VALUES (?, ?, 'pending')`, userID, sourceID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) UpdateSubmissionBuild(id int64, status, msg, output, binaryPath string) error {
	_, err := s.DB.Exec(`UPDATE submissions SET status = ?, msg = ?, compile_output = ?, binary_path = ? WHERE id = ?`,
		status, msg, output, nullString(binaryPath), id)
	return err
}

func (s *Store) SubmissionByID(id int64) (Submission, error) {
	var sub Submission
	err := s.DB.Get(&sub, `SELECT * FROM submissions WHERE id = ?`, id)
	return sub, err
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

func (s *Store) CreateMatch(agentAID, agentBID int64) (int64, error) {
	res, err := s.DB.Exec(`INSERT INTO matches (agent_a_id, agent_b_id, status) VALUES (?, ?, 'queued')`, agentAID, agentBID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) StartMatch(id int64) error {
	_, err := s.DB.Exec(`UPDATE matches SET status = 'running', started_at = ? WHERE id = ?`, time.Now().UTC(), id)
	return err
}

func (s *Store) CompleteMatch(id int64, winnerAgentID sql.NullInt64, startedAt time.Time) error {
	endedAt := time.Now().UTC()
	duration := endedAt.Sub(startedAt).Milliseconds()
	_, err := s.DB.Exec(`UPDATE matches SET status = 'completed', winner_agent_id = ?, ended_at = ?, duration_ms = ? WHERE id = ?`,
		winnerAgentID, endedAt, duration, id)
	return err
}

func (s *Store) FailMatch(id int64, msg string, startedAt time.Time) error {
	endedAt := time.Now().UTC()
	duration := endedAt.Sub(startedAt).Milliseconds()
	_, err := s.DB.Exec(`UPDATE matches SET status = 'failed', error_msg = ?, ended_at = ?, duration_ms = ? WHERE id = ?`,
		msg, endedAt, duration, id)
	return err
}

func (s *Store) MatchByID(id int64) (Match, error) {
	var match Match
	err := s.DB.Get(&match, `SELECT * FROM matches WHERE id = ?`, id)
	return match, err
}

func (s *Store) UserCanViewMatch(userID, matchID int64) (bool, error) {
	var count int
	err := s.DB.Get(&count, `SELECT COUNT(*)
		FROM matches m
		JOIN agents a ON a.id = m.agent_a_id
		WHERE m.id = ? AND a.user_id = ?`, matchID, userID)
	return count > 0, err
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

func nullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func (s *Store) SourcePath(sourceID string) string {
	base, _ := filepath.Abs(s.ArtifactDir)
	return filepath.Join(base, "sources", sourceID, "main.go")
}

func (s *Store) BinaryPath(submissionID int64) string {
	base, _ := filepath.Abs(s.ArtifactDir)
	name := "bot"
	if runtime.GOOS == "windows" { // more robust than os.Args[0]
		name += ".exe"
	}
	return filepath.Join(base, "bins", fmt.Sprintf("%d", submissionID), name)
}

var schemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		password_salt TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
	)`,
	`CREATE TABLE IF NOT EXISTS source_codes (
		id TEXT PRIMARY KEY,
		language TEXT NOT NULL,
		storage_type TEXT NOT NULL,
		storage_key TEXT NOT NULL,
		size INTEGER NOT NULL,
		created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
	)`,
	`CREATE TABLE IF NOT EXISTS submissions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		source_code_id TEXT NOT NULL,
		status TEXT NOT NULL,
		msg TEXT NOT NULL DEFAULT '',
		compile_output TEXT NOT NULL DEFAULT '',
		binary_path TEXT,
		created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
	)`,
	`CREATE TABLE IF NOT EXISTS tournaments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		status TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
	)`,
	`CREATE TABLE IF NOT EXISTS tournament_entries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tournament_id INTEGER NOT NULL,
		agent_id INTEGER NOT NULL,
		seed INTEGER NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS agents (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		submission_id INTEGER,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		status TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
	)`,
	`CREATE TABLE IF NOT EXISTS matches (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tournament_id INTEGER,
		agent_a_id INTEGER NOT NULL,
		agent_b_id INTEGER NOT NULL,
		status TEXT NOT NULL,
		winner_agent_id INTEGER,
		error_msg TEXT NOT NULL DEFAULT '',
		started_at DATETIME,
		ended_at DATETIME,
		duration_ms INTEGER
	)`,
	`CREATE TABLE IF NOT EXISTS games (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		match_id INTEGER NOT NULL,
		player_a TEXT NOT NULL,
		player_b TEXT NOT NULL,
		result TEXT NOT NULL DEFAULT '',
		duration_ms INTEGER NOT NULL DEFAULT 0,
		move_count INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
	)`,
	`CREATE TABLE IF NOT EXISTS moves (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		game_id INTEGER NOT NULL,
		seq INTEGER NOT NULL,
		agent_id INTEGER NOT NULL,
		action_type TEXT NOT NULL,
		action_payload TEXT NOT NULL,
		accepted INTEGER NOT NULL,
		duration_ms INTEGER,
		created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
		UNIQUE(game_id, seq)
	)`,
	`CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		csrf_token TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
		expires_at DATETIME NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions (user_id)`,
	`CREATE TABLE IF NOT EXISTS events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		match_id INTEGER NOT NULL,
		game_id INTEGER,
		seq INTEGER NOT NULL,
		type TEXT NOT NULL,
		payload TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
		UNIQUE(match_id, seq)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_events_match_seq ON events (match_id, seq)`,
	`CREATE TABLE IF NOT EXISTS agent_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		match_id INTEGER NOT NULL,
		agent_id INTEGER NOT NULL,
		content TEXT NOT NULL DEFAULT '',
		truncated INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
		UNIQUE(match_id, agent_id)
	)`,
	`CREATE TABLE IF NOT EXISTS execution_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		match_id INTEGER NOT NULL UNIQUE,
		content TEXT NOT NULL,
		truncated INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
	)`,
}
