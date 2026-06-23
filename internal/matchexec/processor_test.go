package matchexec_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/moby/moby/client"

	"progames/internal/config"
	"progames/internal/events"
	"progames/internal/matchexec"
	"progames/internal/service"
	"progames/internal/store"
	"progames/internal/submission"
)

func TestRunPracticeCreatesEventsMovesAndLog(t *testing.T) {
	if testing.Short() {
		t.Skip("requires Docker")
	}
	t.Parallel()

	cfg := testConfig(t)
	cli := newDockerClient(t)

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
	submit := submission.New(st, cfg, cli)
	result, err := submit.Submit(context.Background(), userID, botSource)
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

	proc := matchexec.NewProcessor(st, events.New(st), cfg, cli)
	matchID, err := proc.Process(service.MatchJob{UserAgentID: result.AgentID, SystemAgentID: systemAgents[0].ID})
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
		DockerImagePrefix:  "progames/bot",
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
