package store

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"progames/internal/config"
)

type Store struct {
	DB          *sqlx.DB
	ArtifactDir string
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

func nullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}
