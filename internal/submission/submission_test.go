package submission_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/moby/moby/client"

	"progames/internal/config"
	"progames/internal/store"
	"progames/internal/submission"
)

func TestSubmitBuildsValidGoSource(t *testing.T) {
	if testing.Short() {
		t.Skip("requires Docker")
	}
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
	svc := submission.New(st, testConfig(t), newDockerClient(t))
	result, err := svc.Submit(context.Background(), userID, "package main\nfunc main() {}\n")
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
	result, err := svc.Submit(context.Background(), userID, "package main\n")
	if err == nil {
		t.Fatalf("expected validation error, got result=%+v", result)
	}
}

func newDockerClient(t *testing.T) *client.Client {
	t.Helper()
	cli, err := client.New(client.FromEnv)
	if err != nil {
		t.Skipf("docker unavailable: %v", err)
	}
	if _, err := cli.Ping(context.Background(), client.PingOptions{}); err != nil {
		_ = cli.Close()
		t.Skipf("docker daemon unreachable: %v", err)
	}
	t.Cleanup(func() { _ = cli.Close() })
	return cli
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
		GoBuilderImage:     "golang:1.26",
	}
}
