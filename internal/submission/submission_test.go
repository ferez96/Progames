package submission_test

import (
	"path/filepath"
	"testing"
	"time"

	"progames/internal/config"
	"progames/internal/store"
	"progames/internal/submission"
)

func TestSubmitBuildsValidGoSource(t *testing.T) {
	t.Parallel()

	st := newStore(t)
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Errorf("close store: %v", err)
		}
	})
	userID, err := st.CreateUser("User", "user@example.com", "hash", "salt")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	svc := submission.New(st, testConfig(t), nil)
	result, err := svc.Submit(userID, "package main\nfunc main() {}\n")
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if result.Status != "compiled" {
		t.Fatalf("expected compiled, got %q output=%s", result.Status, result.Output)
	}
	if result.AgentID == 0 {
		t.Fatal("expected user agent to be created")
	}
}

func TestSubmitRejectsInvalidSource(t *testing.T) {
	t.Parallel()

	st := newStore(t)
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Errorf("close store: %v", err)
		}
	})
	userID, err := st.CreateUser("User", "invalid@example.com", "hash", "salt")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	svc := submission.New(st, testConfig(t), nil)
	result, err := svc.Submit(userID, "package main\n")
	if err == nil {
		t.Fatalf("expected validation error, got result=%+v", result)
	}
}

func newStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(testConfig(t))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return st
}

func testConfig(t *testing.T) config.Config {
	t.Helper()
	base := t.TempDir()
	return config.Config{
		DBPath:             filepath.Join(base, "progames.db"),
		ArtifactDir:        filepath.Join(base, "artifacts"),
		MaxSourceBytes:     256 * 1024,
		PerMoveTimeout:     time.Second,
		MaxStdoutLineBytes: 64 * 1024,
		MaxLogBytes:        1024 * 1024,
		SessionTTL:         time.Hour,
	}
}
