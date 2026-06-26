package submission

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"progames/internal/artifact"
	"progames/internal/config"
	"progames/internal/obs"
	"progames/internal/pkg/agentname"
	"progames/internal/store"
)

type Builder interface {
	Build(ctx context.Context, sourceID artifact.ID) (artifactID artifact.ID, output string, err error)
}

type Service struct {
	store   *store.Store
	cfg     config.Config
	builder Builder
	files   artifact.Repository
}

type Result struct {
	SourceID     string
	SubmissionID int64
	AgentID      int64
	Status       string
	Message      string
	Output       string
}

func New(st *store.Store, cfg config.Config, builder Builder, files artifact.Repository) *Service {
	return &Service{store: st, cfg: cfg, builder: builder, files: files}
}

func (s *Service) Submit(ctx context.Context, userID int64, code string) (Result, error) {
	if s.builder == nil {
		return Result{}, fmt.Errorf("no builder configured")
	}
	code = strings.TrimSpace(code)
	if err := s.validate(code); err != nil {
		return Result{}, err
	}
	sourceID, submissionID, err := s.intake(ctx, userID, code)
	if err != nil {
		return Result{}, err
	}
	return s.compile(ctx, userID, submissionID, sourceID)
}

func (s *Service) validate(code string) error {
	if code == "" {
		return errors.New("source code is required")
	}
	if int64(len(code)) > s.cfg.MaxSourceBytes {
		return fmt.Errorf("source code exceeds %d bytes", s.cfg.MaxSourceBytes)
	}
	if !strings.Contains(code, "package main") || !strings.Contains(code, "func main()") {
		return errors.New("source must be a single Go file with package main and func main()")
	}
	return nil
}

// intake stores the source file and creates the submission record.
// On source_code DB failure the file is compensated; submission record failure
// leaves the source orphaned until the startup scan reconciles it.
func (s *Service) intake(ctx context.Context, userID int64, code string) (artifact.ID, int64, error) {
	sourceID, err := s.files.Write(ctx, strings.NewReader(code))
	if err != nil {
		return "", 0, err
	}
	if err := s.store.CreateSourceCode(string(sourceID), string(sourceID), int64(len(code))); err != nil {
		if derr := s.files.Delete(ctx, sourceID); derr != nil {
			zap.L().Error("submission.source_delete_failed", zap.String("source_id", string(sourceID)), zap.Error(derr))
		}
		return "", 0, err
	}
	submissionID, err := s.store.CreateSubmission(userID, string(sourceID))
	if err != nil {
		return "", 0, err
	}
	zap.L().Info("submission.created",
		zap.Int64("user_id", userID),
		zap.Int64("submission_id", submissionID),
		zap.String("source_id", string(sourceID)),
	)
	return sourceID, submissionID, nil
}

func (s *Service) compile(ctx context.Context, userID, submissionID int64, sourceID artifact.ID) (Result, error) {
	zap.L().Info("submission.build_started",
		zap.Int64("submission_id", submissionID),
		zap.String("source_id", string(sourceID)),
	)

	start := time.Now()
	artifactID, output, buildErr := s.builder.Build(ctx, sourceID)
	durMS := time.Since(start).Milliseconds()

	if buildErr != nil {
		if err := s.store.UpdateSubmissionBuild(submissionID, "invalid", "build failed", output, ""); err != nil {
			zap.L().Error("submission.status_update_failed", zap.Int64("submission_id", submissionID), zap.Error(err))
		}
		obs.SubmissionsInvalid.Add(1)
		zap.L().Info("submission.build_failed",
			zap.Int64("submission_id", submissionID),
			zap.Int64("dur_ms", durMS),
			zap.Error(buildErr),
		)
		return Result{
			SourceID:     string(sourceID),
			SubmissionID: submissionID,
			Status:       "invalid",
			Message:      "build failed",
			Output:       output,
		}, nil
	}

	if err := s.store.UpdateSubmissionBuild(submissionID, "compiled", "build succeeded", output, string(artifactID)); err != nil {
		if derr := s.files.Delete(ctx, artifactID); derr != nil {
			zap.L().Error("submission.artifact_delete_failed", zap.String("artifact_id", string(artifactID)), zap.Error(derr))
		}
		return Result{}, err
	}
	obs.SubmissionsCompiled.Add(1)
	zap.L().Info("submission.build_succeeded",
		zap.Int64("submission_id", submissionID),
		zap.Int64("dur_ms", durMS),
		zap.String("artifact_id", string(artifactID)),
	)

	agentID, err := s.store.CreateAgent(userID, submissionID, agentname.Generate())
	if err != nil {
		return Result{}, err
	}
	zap.L().Info("submission.agent_created",
		zap.Int64("submission_id", submissionID),
		zap.Int64("agent_id", agentID),
	)

	return Result{
		SourceID:     string(sourceID),
		SubmissionID: submissionID,
		AgentID:      agentID,
		Status:       "compiled",
		Message:      "build succeeded",
		Output:       output,
	}, nil
}
