package match

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

	dockertypes "github.com/docker/docker/api/types/build"
	dockerimage "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"go.uber.org/zap"

	"progames/internal/config"
	"progames/internal/events"
	"progames/internal/obs"
	"progames/internal/runner"
	"progames/internal/store"
	"progames/pkg/engine/caro"
)

type Service struct {
	store     *store.Store
	events    *events.Store
	cfg       config.Config
	dockerCli *client.Client
}

func New(st *store.Store, ev *events.Store, cfg config.Config, dockerCli *client.Client) *Service {
	return &Service{store: st, events: ev, cfg: cfg, dockerCli: dockerCli}
}

func (s *Service) RunPractice(userAgentID, systemAgentID int64) (int64, error) {
	p, err := s.prepare(userAgentID, systemAgentID)
	if err != nil {
		return 0, err
	}
	s.run(p)
	return p.matchID, nil
}

type pendingMatch struct {
	matchID   int64
	agentA    store.Agent
	agentB    store.Agent
	startedAt time.Time
}

func (s *Service) prepare(userAgentID, systemAgentID int64) (pendingMatch, error) {
	userAgent, err := s.store.AgentByID(userAgentID)
	if err != nil {
		return pendingMatch{}, err
	}
	systemAgent, err := s.store.AgentByID(systemAgentID)
	if err != nil {
		return pendingMatch{}, err
	}
	if userAgent.Type != "user" || systemAgent.Type != "system" {
		return pendingMatch{}, fmt.Errorf("practice requires one user agent and one system agent")
	}
	matchID, err := s.store.CreateMatch(userAgentID, systemAgentID)
	if err != nil {
		return pendingMatch{}, err
	}
	startedAt := time.Now().UTC()
	if err := s.store.StartMatch(matchID); err != nil {
		return pendingMatch{}, err
	}
	if err := s.events.Append(matchID, sql.NullInt64{}, "match.started", map[string]any{
		"agent_a_id": userAgentID,
		"agent_b_id": systemAgentID,
	}); err != nil {
		return pendingMatch{}, err
	}
	return pendingMatch{matchID: matchID, agentA: userAgent, agentB: systemAgent, startedAt: startedAt}, nil
}

func (s *Service) run(p pendingMatch) {
	winner, runErr := s.runMatchAttempts(p.matchID, p.agentA, p.agentB)
	if runErr != nil {
		_ = s.events.Append(p.matchID, sql.NullInt64{}, "match.failed", map[string]any{"error": runErr.Error()})
		_ = s.store.FailMatch(p.matchID, runErr.Error(), p.startedAt)
		_ = s.events.RenderExecutionLog(p.matchID, s.cfg.MaxLogBytes)
		obs.MatchesFailed.Add(1)
		zap.L().Warn("match.failed",
			zap.Int64("match_id", p.matchID),
			zap.Int64("dur_ms", time.Since(p.startedAt).Milliseconds()),
			zap.Error(runErr),
		)
		return
	}
	draw := !winner.Valid
	_ = s.events.Append(p.matchID, sql.NullInt64{}, "match.completed", map[string]any{
		"winner_agent_id": nullableInt(winner),
		"draw":            draw,
	})
	_ = s.store.CompleteMatch(p.matchID, winner, p.startedAt)
	_ = s.events.RenderExecutionLog(p.matchID, s.cfg.MaxLogBytes)
	obs.MatchesCompleted.Add(1)
	fields := []zap.Field{
		zap.Int64("match_id", p.matchID),
		zap.Int64("dur_ms", time.Since(p.startedAt).Milliseconds()),
		zap.Bool("draw", draw),
	}
	if winner.Valid {
		fields = append(fields, zap.Int64("winner_agent_id", winner.Int64))
	}
	zap.L().Info("match.completed", fields...)
}

func (s *Service) runMatchAttempts(matchID int64, agentA, agentB store.Agent) (sql.NullInt64, error) {
	runners, imageTags, err := s.startRunners(agentA, agentB)
	if err != nil {
		return sql.NullInt64{}, err
	}
	defer func() {
		for agentID, r := range runners {
			_ = s.store.UpsertAgentLog(matchID, agentID, r.Stderr(), false)
			_ = r.Close()
		}
		if s.dockerCli != nil {
			ctx := context.Background()
			for _, tag := range imageTags {
				_, _ = s.dockerCli.ImageRemove(ctx, tag, dockerimage.RemoveOptions{Force: true})
			}
		}
	}()
	for agentID, r := range runners {
		_ = s.events.Append(matchID, sql.NullInt64{}, "bot.started", map[string]any{
			"agent_id": agentID,
			"kind":     fmt.Sprintf("%T", r),
		})
	}

	for attempt := 0; attempt < 6; attempt++ {
		outcome, err := s.runTwoGames(matchID, agentA, agentB, runners, attempt)
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

func (s *Service) runTwoGames(matchID int64, agentA, agentB store.Agent, runners map[int64]runner.AgentRunner, attempt int) (attemptOutcome, error) {
	wins := map[int64]int{agentA.ID: 0, agentB.ID: 0}
	samples := map[int64][]int64{agentA.ID: {}, agentB.ID: {}}
	orders := [][2]store.Agent{{agentA, agentB}, {agentB, agentA}}
	for idx, order := range orders {
		result, err := s.runGame(matchID, order[0], order[1], runners, attempt, idx+1, samples)
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

func (s *Service) runGame(matchID int64, first, second store.Agent, runners map[int64]runner.AgentRunner, attempt, gameNumber int, samples map[int64][]int64) (gameResult, error) {
	start := time.Now()
	gameID, err := s.store.CreateGame(matchID, fmt.Sprintf("%d", first.ID), fmt.Sprintf("%d", second.ID))
	if err != nil {
		return gameResult{}, err
	}
	gameIDNull := sql.NullInt64{Int64: gameID, Valid: true}
	_ = s.events.Append(matchID, gameIDNull, "game.started", map[string]any{
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
		_ = s.events.Append(matchID, gameIDNull, "turn.state_sent", map[string]any{
			"game_number": gameNumber,
			"turn":        turn,
			"agent_id":    agentID,
			"state":       state,
		})
		move, err := r.Move(state, s.cfg.PerMoveTimeout)
		if err != nil {
			reason := "crash"
			eventType := "turn.crash"
			if strings.Contains(err.Error(), "timeout") {
				reason = "timeout"
				eventType = "turn.timeout"
			}
			_ = s.events.Append(matchID, gameIDNull, eventType, map[string]any{
				"game_number": gameNumber,
				"turn":        turn,
				"agent_id":    agentID,
				"reason":      reason,
			})
			_ = s.events.ProjectMove(gameID, turn, agentID, reason, map[string]any{}, false, sql.NullInt64{})
			winner := otherAgent(agentID, first.ID, second.ID)
			return s.finishGame(matchID, gameID, gameIDNull, gameNumber, winner, first.ID, start, game.MoveCount())
		}
		pos, parseErr := parseMove(move.RawLine)
		if parseErr != nil {
			_ = s.events.Append(matchID, gameIDNull, "turn.move_rejected", map[string]any{
				"game_number": gameNumber,
				"turn":        turn,
				"agent_id":    agentID,
				"reason":      "invalid_format",
				"raw":         move.RawLine,
			})
			_ = s.events.ProjectMove(gameID, turn, agentID, "invalid", map[string]any{"raw": move.RawLine}, false, sql.NullInt64{})
			winner := otherAgent(agentID, first.ID, second.ID)
			return s.finishGame(matchID, gameID, gameIDNull, gameNumber, winner, first.ID, start, game.MoveCount())
		}
		if err := game.ApplyMove(pos); err != nil {
			_ = s.events.Append(matchID, gameIDNull, "turn.move_rejected", map[string]any{
				"game_number": gameNumber,
				"turn":        turn,
				"agent_id":    agentID,
				"reason":      err.Error(),
				"raw":         move.RawLine,
			})
			_ = s.events.ProjectMove(gameID, turn, agentID, "invalid", map[string]any{"raw": move.RawLine}, false, sql.NullInt64{})
			winner := otherAgent(agentID, first.ID, second.ID)
			return s.finishGame(matchID, gameID, gameIDNull, gameNumber, winner, first.ID, start, game.MoveCount())
		}
		samples[agentID] = append(samples[agentID], move.DurationMS)
		payload := map[string]any{"x": pos.X, "y": pos.Y}
		_ = s.events.Append(matchID, gameIDNull, "turn.move_accepted", map[string]any{
			"game_number": gameNumber,
			"turn":        turn,
			"agent_id":    agentID,
			"move":        payload,
			"duration_ms": move.DurationMS,
		})
		_ = s.events.ProjectMove(gameID, turn, agentID, "place", payload, true, sql.NullInt64{Int64: move.DurationMS, Valid: true})
		_ = agentByID // keep the domain mapping close to turn handling for future policy checks.
	}
	if game.Result() == "draw" {
		return s.finishGame(matchID, gameID, gameIDNull, gameNumber, sql.NullInt64{}, first.ID, start, game.MoveCount())
	}
	winner, _ := strconv.ParseInt(game.Result(), 10, 64)
	return s.finishGame(matchID, gameID, gameIDNull, gameNumber, sql.NullInt64{Int64: winner, Valid: true}, first.ID, start, game.MoveCount())
}

func (s *Service) finishGame(matchID, gameID int64, gameIDNull sql.NullInt64, gameNumber int, winner sql.NullInt64, playerAID int64, started time.Time, moveCount int) (gameResult, error) {
	result := "draw"
	if winner.Valid {
		if winner.Int64 == playerAID {
			result = "player_a_win"
		} else {
			result = "player_b_win"
		}
	}
	duration := time.Since(started).Milliseconds()
	if err := s.store.FinishGame(gameID, result, duration, int64(moveCount)); err != nil {
		return gameResult{}, err
	}
	if err := s.events.Append(matchID, gameIDNull, "game.ended", map[string]any{
		"game_number":     gameNumber,
		"result":          result,
		"winner_agent_id": nullableInt(winner),
	}); err != nil {
		return gameResult{}, err
	}
	return gameResult{Winner: winner}, nil
}

func (s *Service) startRunners(agentA, agentB store.Agent) (map[int64]runner.AgentRunner, []string, error) {
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
			sub, err := s.store.SubmissionByID(agent.SubmissionID.Int64)
			if err != nil {
				return nil, nil, err
			}
			if s.dockerCli != nil {
				imageTag := fmt.Sprintf("%s:%d", s.cfg.DockerImagePrefix, sub.ID)
				if err := buildImage(context.Background(), s.dockerCli, sub.BinaryPath.String, imageTag); err != nil {
					return nil, nil, fmt.Errorf("build image for submission %d: %w", sub.ID, err)
				}
				imageTags = append(imageTags, imageTag)
				r = runner.NewContainer(s.dockerCli, imageTag, s.cfg.MaxStdoutLineBytes)
			} else {
				if !sub.BinaryPath.Valid {
					return nil, nil, fmt.Errorf("submission %d has no binary", sub.ID)
				}
				r = runner.NewProcess(sub.BinaryPath.String, s.cfg.MaxStdoutLineBytes)
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
	resp, err := cli.ImageBuild(ctx, &buf, dockertypes.ImageBuildOptions{
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
