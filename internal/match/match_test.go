package match_test

import (
	"path/filepath"
	"testing"
	"time"

	"progames/internal/config"
	"progames/internal/events"
	matchsvc "progames/internal/match"
	"progames/internal/store"
	"progames/internal/submission"
)

func TestRunPracticeCreatesEventsMovesAndLog(t *testing.T) {
	t.Parallel()

	cfg := testConfig(t)
	st, err := store.Open(cfg)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Errorf("close store: %v", err)
		}
	})
	userID, err := st.CreateUser("User", "player@example.com", "hash", "salt")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	submit := submission.New(st, cfg, nil)
	result, err := submit.Submit(userID, botSource)
	if err != nil {
		t.Fatalf("submit bot: %v", err)
	}
	if result.Status != "compiled" {
		t.Fatalf("expected compiled, got %q output=%s", result.Status, result.Output)
	}
	systemAgents, err := st.SystemAgents()
	if err != nil {
		t.Fatalf("system agents: %v", err)
	}
	if len(systemAgents) == 0 {
		t.Fatal("expected system agent")
	}

	matches := matchsvc.New(st, events.New(st), cfg, nil)
	matchID, err := matches.RunPractice(result.AgentID, systemAgents[0].ID)
	if err != nil {
		t.Fatalf("run practice: %v", err)
	}
	match, err := st.MatchByID(matchID)
	if err != nil {
		t.Fatalf("match by id: %v", err)
	}
	if match.Status != "completed" {
		t.Fatalf("expected completed match, got %q", match.Status)
	}
	events, err := st.ListEvents(matchID)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected events")
	}
	games, err := st.ListGames(matchID)
	if err != nil {
		t.Fatalf("games: %v", err)
	}
	if len(games) < 2 {
		t.Fatalf("expected at least two games, got %d", len(games))
	}
	moves, err := st.ListMoves(games[0].ID)
	if err != nil {
		t.Fatalf("moves: %v", err)
	}
	if len(moves) == 0 {
		t.Fatal("expected projected moves")
	}
	if _, err := st.ExecutionLog(matchID); err != nil {
		t.Fatalf("execution log: %v", err)
	}
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

const botSource = `package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	in := bufio.NewScanner(os.Stdin)
	for in.Scan() {
		state := in.Text()
		for i, cell := range state {
			if cell == '.' {
				fmt.Printf("%d,%d\n", i%8+1, i/8+1)
				break
			}
		}
	}
}
`
