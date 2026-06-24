package matchexec_test

import (
	"context"
	"testing"

	"progames/internal/artifact"
	"progames/internal/events"
	"progames/internal/matchexec"
	"progames/internal/sandbox"
	"progames/internal/service"
	"progames/internal/submission"
	"progames/internal/testhelper"
)

func TestRunPracticeCreatesEventsMovesAndLog(t *testing.T) {
	t.Parallel()

	cfg := testhelper.TestConfig(t)
	cli := testhelper.NewDockerClient(t)
	st := testhelper.NewStore(t, cfg)

	userID, err := st.CreateUser("User", "player@example.com", "hash", "salt")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	repo := artifact.NewLocalRepository(cfg.ArtifactDir)
	submit := submission.New(st, cfg, sandbox.NewCompiler(cli, cfg.GoBuilderImage, repo), repo)
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

	proc := matchexec.NewProcessor(st, events.New(st), cfg, cli, repo)
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
	evs, err := st.ListEvents(matchID)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	if len(evs) == 0 {
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
