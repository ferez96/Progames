package store_test

import (
	"path/filepath"
	"testing"

	"progames/internal/config"
	"progames/internal/store"
)

func TestOpenInitializesSchemaAndSeedsSystemAgent(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		DBPath:      filepath.Join(t.TempDir(), "progames.db"),
		ArtifactDir: filepath.Join(t.TempDir(), "artifacts"),
	}
	st, err := store.Open(cfg)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	agents, err := st.SystemAgents()
	if err != nil {
		t.Fatalf("list system agents: %v", err)
	}
	if len(agents) == 0 {
		t.Fatal("expected at least one seeded system agent")
	}
	if agents[0].Type != "system" {
		t.Fatalf("expected system agent, got %q", agents[0].Type)
	}

	if err := st.Init(); err != nil {
		t.Fatalf("idempotent init: %v", err)
	}
}
