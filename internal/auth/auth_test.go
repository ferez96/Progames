package auth_test

import (
	"path/filepath"
	"testing"
	"time"

	"progames/internal/auth"
	"progames/internal/config"
	"progames/internal/store"
)

func TestSignUpAndSignInCreatesSession(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		DBPath:      filepath.Join(t.TempDir(), "progames.db"),
		ArtifactDir: filepath.Join(t.TempDir(), "artifacts"),
		SessionTTL:  time.Hour,
	}
	st, err := store.Open(cfg)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()
	svc := auth.New(st, cfg)
	if _, err := svc.SignUp("User", "auth@example.com", "secret"); err != nil {
		t.Fatalf("signup: %v", err)
	}
	user, sessionID, csrf, err := svc.SignIn("auth@example.com", "secret")
	if err != nil {
		t.Fatalf("signin: %v", err)
	}
	if user.ID == 0 || sessionID == "" || csrf == "" {
		t.Fatalf("expected user/session/csrf, got user=%+v session=%q csrf=%q", user, sessionID, csrf)
	}
}
