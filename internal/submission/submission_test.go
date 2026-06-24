package submission_test

import (
	"context"
	"fmt"
	"testing"

	"progames/internal/artifact"
	"progames/internal/sandbox"
	"progames/internal/submission"
	"progames/internal/testhelper"
)

// failBuilder satisfies submission.Builder but always returns a build error.
// Used to exercise validation paths without requiring Docker.
type failBuilder struct{}

func (failBuilder) Build(_ context.Context, _ artifact.ID) (artifact.ID, string, error) {
	return "", "", fmt.Errorf("stub: build not available")
}

func TestSubmitBuildsValidGoSource(t *testing.T) {
	t.Parallel()

	cfg := testhelper.TestConfig(t)
	cli := testhelper.NewDockerClient(t)
	st := testhelper.NewStore(t, cfg)

	userID, err := st.CreateUser("User", "user@example.com", "hash", "salt")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	repo := artifact.NewLocalRepository(cfg.ArtifactDir)
	svc := submission.New(st, cfg, sandbox.NewCompiler(cli, cfg.GoBuilderImage, repo), repo)
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

	cfg := testhelper.TestConfig(t)
	st := testhelper.NewStore(t, cfg)
	userID, err := st.CreateUser("User", "invalid@example.com", "hash", "salt")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	repo := artifact.NewLocalRepository(cfg.ArtifactDir)
	svc := submission.New(st, cfg, failBuilder{}, repo)
	result, err := svc.Submit(context.Background(), userID, "package main\n")
	if err == nil {
		t.Fatalf("expected validation error, got result=%+v", result)
	}
}
