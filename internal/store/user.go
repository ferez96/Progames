package store

import (
	"strings"
	"time"
)

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
