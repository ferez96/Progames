package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"

	"progames/internal/config"
	"progames/internal/store"
)

const SessionCookieName = "progames_session"

type contextKey string

const userContextKey contextKey = "user"

// User and Session are the minimal auth-layer identity types.
// They carry only what downstream layers need — no password hashes or db tags.
type User struct {
	ID   int64
	Name string
}

type Session struct {
	CSRFToken string
}

type Service struct {
	store       *store.Store
	sessionTTL  time.Duration
	forceSecure bool
	failures    map[string]failureState
	mu          sync.Mutex
}

type failureState struct {
	Count     int
	LockedTil time.Time
}

func New(st *store.Store, cfg config.Config) *Service {
	return &Service{
		store:       st,
		sessionTTL:  cfg.SessionTTL,
		forceSecure: cfg.ForceSecureCookie,
		failures:    map[string]failureState{},
	}
}

func (s *Service) SignUp(name, email, password string) (int64, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))
	if name == "" || email == "" || password == "" {
		return 0, errors.New("name, email, and password are required")
	}
	hash, salt, err := hashPassword(password)
	if err != nil {
		return 0, err
	}
	return s.store.CreateUser(name, email, hash, salt)
}

func (s *Service) SignIn(email, password string) (User, string, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if s.isLocked(email) {
		_, _ = verifyPassword(password, dummyHash, dummySalt)
		return User{}, "", "", errors.New("invalid email or password")
	}
	user, err := s.store.UserByEmail(email)
	if err != nil {
		_, _ = verifyPassword(password, dummyHash, dummySalt)
		s.recordFailure(email)
		return User{}, "", "", errors.New("invalid email or password")
	}
	ok, err := verifyPassword(password, user.PasswordHash, user.PasswordSalt)
	if err != nil || !ok {
		s.recordFailure(email)
		return User{}, "", "", errors.New("invalid email or password")
	}
	s.clearFailures(email)
	sessionID := uuid.NewString()
	csrf := uuid.NewString()
	if err := s.store.CreateSession(sessionID, user.ID, csrf, time.Now().UTC().Add(s.sessionTTL)); err != nil {
		return User{}, "", "", err
	}
	return User{ID: user.ID, Name: user.Name}, sessionID, csrf, nil
}

func (s *Service) SignOut(sessionID string) error {
	return s.store.DeleteSession(sessionID)
}

func (s *Service) UserFromRequest(r *http.Request) (User, Session, error) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return User{}, Session{}, err
	}
	session, err := s.store.SessionByID(cookie.Value)
	if err != nil {
		return User{}, Session{}, err
	}
	user, err := s.store.UserByID(session.UserID)
	if err != nil {
		return User{}, Session{}, err
	}
	return User{ID: user.ID, Name: user.Name}, Session{CSRFToken: session.CSRFToken}, nil
}

func CurrentUser(r *http.Request) (User, bool) {
	user, ok := r.Context().Value(userContextKey).(User)
	return user, ok
}

func CurrentSession(r *http.Request) (Session, bool) {
	session, ok := r.Context().Value(contextKey("session")).(Session)
	return session, ok
}

func (s *Service) SetSessionCookie(w http.ResponseWriter, r *http.Request, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   s.forceSecure || r.TLS != nil,
		Expires:  time.Now().UTC().Add(s.sessionTTL),
	})
}

func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
		MaxAge:   -1,
	})
}

func ValidateCSRF(r *http.Request) bool {
	session, ok := CurrentSession(r)
	if !ok {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(session.CSRFToken), []byte(r.FormValue("csrf_token"))) == 1
}

func (s *Service) isLocked(email string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.failures[email]
	return state.LockedTil.After(time.Now())
}

func (s *Service) recordFailure(email string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.failures[email]
	state.Count++
	if state.Count >= 5 {
		state.LockedTil = time.Now().Add(30 * time.Second)
	}
	s.failures[email] = state
}

func (s *Service) clearFailures(email string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.failures, email)
}

func hashPassword(password string) (string, string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", "", err
	}
	hash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 2, 32)
	return base64.RawStdEncoding.EncodeToString(hash), base64.RawStdEncoding.EncodeToString(salt), nil
}

func verifyPassword(password, encodedHash, encodedSalt string) (bool, error) {
	if encodedHash == "*" {
		return false, sql.ErrNoRows
	}
	salt, err := base64.RawStdEncoding.DecodeString(encodedSalt)
	if err != nil {
		return false, err
	}
	want, err := base64.RawStdEncoding.DecodeString(encodedHash)
	if err != nil {
		return false, err
	}
	got := argon2.IDKey([]byte(password), salt, 3, 64*1024, 2, 32)
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

var (
	dummySalt = base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef"))
	dummyHash = base64.RawStdEncoding.EncodeToString(argon2.IDKey([]byte("dummy"), []byte("0123456789abcdef"), 3, 64*1024, 2, 32))
)
