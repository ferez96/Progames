package auth

import (
	"context"
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

func (s *Service) SignIn(email, password string) (store.User, string, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if s.isLocked(email) {
		_, _ = verifyPassword(password, dummyHash, dummySalt)
		return store.User{}, "", "", errors.New("invalid email or password")
	}
	user, err := s.store.UserByEmail(email)
	if err != nil {
		_, _ = verifyPassword(password, dummyHash, dummySalt)
		s.recordFailure(email)
		return store.User{}, "", "", errors.New("invalid email or password")
	}
	ok, err := verifyPassword(password, user.PasswordHash, user.PasswordSalt)
	if err != nil || !ok {
		s.recordFailure(email)
		return store.User{}, "", "", errors.New("invalid email or password")
	}
	s.clearFailures(email)
	sessionID := uuid.NewString()
	csrf := uuid.NewString()
	if err := s.store.CreateSession(sessionID, user.ID, csrf, time.Now().UTC().Add(s.sessionTTL)); err != nil {
		return store.User{}, "", "", err
	}
	return user, sessionID, csrf, nil
}

func (s *Service) SignOut(sessionID string) error {
	return s.store.DeleteSession(sessionID)
}

func (s *Service) UserFromRequest(r *http.Request) (store.User, store.Session, error) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return store.User{}, store.Session{}, err
	}
	session, err := s.store.SessionByID(cookie.Value)
	if err != nil {
		return store.User{}, store.Session{}, err
	}
	user, err := s.store.UserByID(session.UserID)
	return user, session, err
}

func (s *Service) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, session, err := s.UserFromRequest(r)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey, user)
		ctx = context.WithValue(ctx, contextKey("session"), session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func CurrentUser(r *http.Request) (store.User, bool) {
	user, ok := r.Context().Value(userContextKey).(store.User)
	return user, ok
}

func CurrentSession(r *http.Request) (store.Session, bool) {
	session, ok := r.Context().Value(contextKey("session")).(store.Session)
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
