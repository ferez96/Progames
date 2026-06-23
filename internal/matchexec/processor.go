package matchexec

import (
	"archive/tar"
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/moby/moby/client"
	"go.uber.org/zap"

	"progames/internal/config"
	"progames/internal/events"
	"progames/internal/obs"
	"progames/internal/runner"
	"progames/internal/service"
	"progames/internal/store"
	"progames/pkg/engine/caro"
)

// preparedJob is the internal state after the match is created in the DB.
// It carries everything needed to execute the games without further DB reads.
type preparedJob struct {
	matchID   int64
	agentA    store.Agent
	agentB    store.Agent
	startedAt time.Time
}

type matchStore interface {
	AgentByID(id int64) (store.Agent, error)
	SubmissionByID(id int64) (store.Submission, error)
	CreateMatch(agentAID, agentBID int64) (int64, error)
	StartMatch(id int64, startedAt time.Time) error
	FailMatch(id int64, msg string, endedAt time.Time, durationMS int64) error
	CompleteMatch(id int64, winnerAgentID sql.NullInt64, endedAt time.Time, durationMS int64) error
	CreateGame(matchID int64, playerA, playerB string) (int64, error)
	FinishGame(id int64, result string, durationMS, moveCount int64) error
	UpsertAgentLog(matchID, agentID int64, content string, truncated bool) error
}

// Processor handles match execution: DB setup and game loop.
type Processor struct {
	store     matchStore
	events    *events.Store
	cfg       config.Config
	dockerCli *client.Client
}

func NewProcessor(st matchStore, ev *events.Store, cfg config.Config, dockerCli *client.Client) *Processor {
	return &Processor{store: st, events: ev, cfg: cfg, dockerCli: dockerCli}
}

// Process runs a job synchronously: prepares the match in the DB then executes all games.
// Used directly when no queue is needed (e.g. tests). The queue calls prepare/execute separately.
func (p *Processor) Process(job service.MatchJob) (int64, error) {
	pj, err := p.prepare(job)
	if err != nil {
		return 0, err
	}
	p.execute(pj)
	return pj.matchID, nil
}

func (p *Processor) prepare(job service.MatchJob) (preparedJob, error) {
	userAgent, err := p.store.AgentByID(job.UserAgentID)
	if err != nil {
		return preparedJob{}, err
	}
	systemAgent, err := p.store.AgentByID(job.SystemAgentID)
	if err != nil {
		return preparedJob{}, err
	}
	if userAgent.Type != "user" || systemAgent.Type != "system" {
		return preparedJob{}, fmt.Errorf("practice requires one user agent and one system agent")
	}
	matchID, err := p.store.CreateMatch(job.UserAgentID, job.SystemAgentID)
	if err != nil {
		return preparedJob{}, err
	}
	startedAt := time.Now().UTC()
	if err := p.store.StartMatch(matchID, startedAt); err != nil {
		return preparedJob{}, err
	}
	if err := p.events.Append(matchID, sql.NullInt64{}, "match.started", map[string]any{
		"agent_a_id": job.UserAgentID,
		"agent_b_id": job.SystemAgentID,
	}); err != nil {
		return preparedJob{}, err
	}
	return preparedJob{matchID: matchID, agentA: userAgent, agentB: systemAgent, startedAt: startedAt}, nil
}

func (p *Processor) execute(pj preparedJob) {
	winner, runErr := p.runMatchAttempts(pj.matchID, pj.agentA, pj.agentB)
	endedAt := time.Now().UTC()
	durationMS := endedAt.Sub(pj.startedAt).Milliseconds()
	if runErr != nil {
		_ = p.events.Append(pj.matchID, sql.NullInt64{}, "match.failed", map[string]any{"error": runErr.Error()})
		_ = p.store.FailMatch(pj.matchID, runErr.Error(), endedAt, durationMS)
		_ = p.events.RenderExecutionLog(pj.matchID, p.cfg.MaxLogBytes)
		obs.MatchesFailed.Add(1)
		zap.L().Warn("match.failed",
			zap.Int64("match_id", pj.matchID),
			zap.Int64("dur_ms", durationMS),
			zap.Error(runErr),
		)
		return
	}
	draw := !winner.Valid
	_ = p.events.Append(pj.matchID, sql.NullInt64{}, "match.completed", map[string]any{
		"winner_agent_id": nullableInt(winner),
		"draw":            draw,
	})
	_ = p.store.CompleteMatch(pj.matchID, winner, endedAt, durationMS)
	_ = p.events.RenderExecutionLog(pj.matchID, p.cfg.MaxLogBytes)
	obs.MatchesCompleted.Add(1)
	fields := []zap.Field{
		zap.Int64("match_id", pj.matchID),
		zap.Int64("dur_ms", durationMS),
		zap.Bool("draw", draw),
	}
	if winner.Valid {
		fields = append(fields, zap.Int64("winner_agent_id", winner.Int64))
	}
	zap.L().Info("match.completed", fields...)
}

func (p *Processor) runMatchAttempts(matchID int64, agentA, agentB store.Agent) (sql.NullInt64, error) {
	runners, imageTags, err := p.startRunners(agentA, agentB)
	if err != nil {
		return sql.NullInt64{}, err
	}
	defer func() {
		for agentID, r := range runners {
			_ = p.store.UpsertAgentLog(matchID, agentID, r.Stderr(), false)
			_ = r.Close()
		}
		if p.dockerCli != nil {
			ctx := context.Background()
			for _, tag := range imageTags {
				_, _ = p.dockerCli.ImageRemove(ctx, tag, client.ImageRemoveOptions{Force: true})
			}
		}
	}()
	for agentID, r := range runners {
		_ = p.events.Append(matchID, sql.NullInt64{}, "bot.started", map[string]any{
			"agent_id": agentID,
			"kind":     fmt.Sprintf("%T", r),
		})
	}

	for attempt := 0; attempt < 6; attempt++ {
		outcome, err := p.runTwoGames(matchID, agentA, agentB, runners, attempt)
		if err != nil {
			return sql.NullInt64{}, err
		}
		if outcome.Winner.Valid {
			return outcome.Winner, nil
		}
	}
	return sql.NullInt64{}, nil
}

type attemptOutcome struct {
	Winner sql.NullInt64
}

func (p *Processor) runTwoGames(matchID int64, agentA, agentB store.Agent, runners map[int64]runner.AgentRunner, attempt int) (attemptOutcome, error) {
	wins := map[int64]int{agentA.ID: 0, agentB.ID: 0}
	samples := map[int64][]int64{agentA.ID: {}, agentB.ID: {}}
	orders := [][2]store.Agent{{agentA, agentB}, {agentB, agentA}}
	for idx, order := range orders {
		result, err := p.runGame(matchID, order[0], order[1], runners, attempt, idx+1, samples)
		if err != nil {
			return attemptOutcome{}, err
		}
		if result.Winner.Valid {
			wins[result.Winner.Int64]++
		}
	}
	if wins[agentA.ID] > wins[agentB.ID] {
		return attemptOutcome{Winner: sql.NullInt64{Int64: agentA.ID, Valid: true}}, nil
	}
	if wins[agentB.ID] > wins[agentA.ID] {
		return attemptOutcome{Winner: sql.NullInt64{Int64: agentB.ID, Valid: true}}, nil
	}
	sumA, cntA, okA := sumCount(samples[agentA.ID])
	sumB, cntB, okB := sumCount(samples[agentB.ID])
	if okA && okB {
		if sumA*cntB < sumB*cntA {
			return attemptOutcome{Winner: sql.NullInt64{Int64: agentA.ID, Valid: true}}, nil
		}
		if sumB*cntA < sumA*cntB {
			return attemptOutcome{Winner: sql.NullInt64{Int64: agentB.ID, Valid: true}}, nil
		}
	}
	if okA && !okB {
		return attemptOutcome{Winner: sql.NullInt64{Int64: agentA.ID, Valid: true}}, nil
	}
	if okB && !okA {
		return attemptOutcome{Winner: sql.NullInt64{Int64: agentB.ID, Valid: true}}, nil
	}
	return attemptOutcome{}, nil
}

type gameResult struct {
	Winner sql.NullInt64
}

func (p *Processor) runGame(matchID int64, first, second store.Agent, runners map[int64]runner.AgentRunner, attempt, gameNumber int, samples map[int64][]int64) (gameResult, error) {
	start := time.Now()
	gameID, err := p.store.CreateGame(matchID, fmt.Sprintf("%d", first.ID), fmt.Sprintf("%d", second.ID))
	if err != nil {
		return gameResult{}, err
	}
	gameIDNull := sql.NullInt64{Int64: gameID, Valid: true}
	_ = p.events.Append(matchID, gameIDNull, "game.started", map[string]any{
		"attempt":           attempt,
		"game_number":       gameNumber,
		"player_a_agent_id": first.ID,
		"player_b_agent_id": second.ID,
	})
	game := caro.NewGame([]string{fmt.Sprintf("%d", first.ID), fmt.Sprintf("%d", second.ID)})
	turn := int64(0)
	agentByID := map[int64]store.Agent{first.ID: first, second.ID: second}
	for !game.IsOver() {
		turn++
		agentID, _ := strconv.ParseInt(game.CurrentPlayer(), 10, 64)
		r := runners[agentID]
		state := game.Snapshot()
		_ = p.events.Append(matchID, gameIDNull, "turn.state_sent", map[string]any{
			"game_number": gameNumber,
			"turn":        turn,
			"agent_id":    agentID,
			"state":       state,
		})
		move, err := r.Move(state, p.cfg.PerMoveTimeout)
		if err != nil {
			reason := "crash"
			eventType := "turn.crash"
			if strings.Contains(err.Error(), "timeout") {
				reason = "timeout"
				eventType = "turn.timeout"
			}
			_ = p.events.Append(matchID, gameIDNull, eventType, map[string]any{
				"game_number": gameNumber,
				"turn":        turn,
				"agent_id":    agentID,
				"reason":      reason,
			})
			_ = p.events.ProjectMove(gameID, turn, agentID, reason, map[string]any{}, false, sql.NullInt64{})
			winner := otherAgent(agentID, first.ID, second.ID)
			return p.finishGame(matchID, gameID, gameIDNull, gameNumber, winner, first.ID, start, game.MoveCount())
		}
		pos, parseErr := parseMove(move.RawLine)
		if parseErr != nil {
			_ = p.events.Append(matchID, gameIDNull, "turn.move_rejected", map[string]any{
				"game_number": gameNumber,
				"turn":        turn,
				"agent_id":    agentID,
				"reason":      "invalid_format",
				"raw":         move.RawLine,
			})
			_ = p.events.ProjectMove(gameID, turn, agentID, "invalid", map[string]any{"raw": move.RawLine}, false, sql.NullInt64{})
			winner := otherAgent(agentID, first.ID, second.ID)
			return p.finishGame(matchID, gameID, gameIDNull, gameNumber, winner, first.ID, start, game.MoveCount())
		}
		if err := game.ApplyMove(pos); err != nil {
			_ = p.events.Append(matchID, gameIDNull, "turn.move_rejected", map[string]any{
				"game_number": gameNumber,
				"turn":        turn,
				"agent_id":    agentID,
				"reason":      err.Error(),
				"raw":         move.RawLine,
			})
			_ = p.events.ProjectMove(gameID, turn, agentID, "invalid", map[string]any{"raw": move.RawLine}, false, sql.NullInt64{})
			winner := otherAgent(agentID, first.ID, second.ID)
			return p.finishGame(matchID, gameID, gameIDNull, gameNumber, winner, first.ID, start, game.MoveCount())
		}
		samples[agentID] = append(samples[agentID], move.DurationMS)
		payload := map[string]any{"x": pos.X, "y": pos.Y}
		_ = p.events.Append(matchID, gameIDNull, "turn.move_accepted", map[string]any{
			"game_number": gameNumber,
			"turn":        turn,
			"agent_id":    agentID,
			"move":        payload,
			"duration_ms": move.DurationMS,
		})
		_ = p.events.ProjectMove(gameID, turn, agentID, "place", payload, true, sql.NullInt64{Int64: move.DurationMS, Valid: true})
		_ = agentByID // keep the domain mapping close to turn handling for future policy checks.
	}
	if game.Result() == "draw" {
		return p.finishGame(matchID, gameID, gameIDNull, gameNumber, sql.NullInt64{}, first.ID, start, game.MoveCount())
	}
	winner, _ := strconv.ParseInt(game.Result(), 10, 64)
	return p.finishGame(matchID, gameID, gameIDNull, gameNumber, sql.NullInt64{Int64: winner, Valid: true}, first.ID, start, game.MoveCount())
}

func (p *Processor) finishGame(matchID, gameID int64, gameIDNull sql.NullInt64, gameNumber int, winner sql.NullInt64, playerAID int64, started time.Time, moveCount int) (gameResult, error) {
	result := service.ResultDraw
	if winner.Valid {
		if winner.Int64 == playerAID {
			result = service.ResultPlayerAWin
		} else {
			result = service.ResultPlayerBWin
		}
	}
	duration := time.Since(started).Milliseconds()
	if err := p.store.FinishGame(gameID, result, duration, int64(moveCount)); err != nil {
		return gameResult{}, err
	}
	if err := p.events.Append(matchID, gameIDNull, "game.ended", map[string]any{
		"game_number":     gameNumber,
		"result":          result,
		"winner_agent_id": nullableInt(winner),
	}); err != nil {
		return gameResult{}, err
	}
	return gameResult{Winner: winner}, nil
}

func (p *Processor) startRunners(agentA, agentB store.Agent) (map[int64]runner.AgentRunner, []string, error) {
	result := map[int64]runner.AgentRunner{}
	var imageTags []string
	for _, agent := range []store.Agent{agentA, agentB} {
		var r runner.AgentRunner
		if agent.Type == "system" {
			r = runner.NewSystemAgent()
		} else {
			if !agent.SubmissionID.Valid {
				return nil, nil, fmt.Errorf("user agent %d has no submission", agent.ID)
			}
			sub, err := p.store.SubmissionByID(agent.SubmissionID.Int64)
			if err != nil {
				return nil, nil, err
			}
			if p.dockerCli != nil {
				imageTag := fmt.Sprintf("%s:%d", p.cfg.DockerImagePrefix, sub.ID)
				if err := buildImage(context.Background(), p.dockerCli, sub.BinaryPath.String, imageTag); err != nil {
					return nil, nil, fmt.Errorf("build image for submission %d: %w", sub.ID, err)
				}
				imageTags = append(imageTags, imageTag)
				r = runner.NewContainer(p.dockerCli, imageTag, p.cfg.MaxStdoutLineBytes, p.cfg.BotMemoryBytes, p.cfg.BotNanoCPUs)
			} else {
				if !sub.BinaryPath.Valid {
					return nil, nil, fmt.Errorf("submission %d has no binary", sub.ID)
				}
				r = runner.NewProcess(sub.BinaryPath.String, p.cfg.MaxStdoutLineBytes)
			}
		}
		if err := r.Start(); err != nil {
			return nil, nil, err
		}
		result[agent.ID] = r
	}
	return result, imageTags, nil
}

func parseMove(raw string) (caro.Position, error) {
	parts := strings.Split(raw, ",")
	if len(parts) != 2 {
		return caro.Position{}, fmt.Errorf("invalid move format")
	}
	x, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return caro.Position{}, err
	}
	y, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return caro.Position{}, err
	}
	return caro.Position{X: x, Y: y}, nil
}

func otherAgent(agentID, a, b int64) sql.NullInt64 {
	if agentID == a {
		return sql.NullInt64{Int64: b, Valid: true}
	}
	return sql.NullInt64{Int64: a, Valid: true}
}

func sumCount(values []int64) (int64, int64, bool) {
	if len(values) == 0 {
		return 0, 0, false
	}
	var sum int64
	for _, v := range values {
		sum += v
	}
	return sum, int64(len(values)), true
}

func buildImage(ctx context.Context, cli *client.Client, binaryPath, imageTag string) error {
	binary, err := os.ReadFile(binaryPath)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	dockerfile := "FROM scratch\nCOPY bot /bot\nENTRYPOINT [\"/bot\"]\n"
	_ = tw.WriteHeader(&tar.Header{Name: "Dockerfile", Size: int64(len(dockerfile)), Mode: 0o644})
	_, _ = io.WriteString(tw, dockerfile)
	_ = tw.WriteHeader(&tar.Header{Name: "bot", Size: int64(len(binary)), Mode: 0o755})
	_, _ = tw.Write(binary)
	_ = tw.Close()
	resp, err := cli.ImageBuild(ctx, &buf, client.ImageBuildOptions{
		Tags:   []string{imageTag},
		Remove: true,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(io.Discard, resp.Body)
	return err
}

func nullableInt(value sql.NullInt64) any {
	if !value.Valid {
		return nil
	}
	return value.Int64
}
