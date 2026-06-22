package store

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
