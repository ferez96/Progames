package submission

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"progames/internal/config"
	"progames/internal/obs"
	"progames/internal/store"
)

type Service struct {
	store     *store.Store
	cfg       config.Config
	dockerCli *client.Client
}

type Result struct {
	SourceID     string
	SubmissionID int64
	AgentID      int64
	Status       string
	Message      string
	Output       string
}

func New(st *store.Store, cfg config.Config, dockerCli *client.Client) *Service {
	return &Service{store: st, cfg: cfg, dockerCli: dockerCli}
}

func (s *Service) Submit(userID int64, code string) (Result, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return Result{}, errors.New("source code is required")
	}
	if int64(len(code)) > s.cfg.MaxSourceBytes {
		return Result{}, fmt.Errorf("source code exceeds %d bytes", s.cfg.MaxSourceBytes)
	}
	if !strings.Contains(code, "package main") || !strings.Contains(code, "func main()") {
		return Result{}, errors.New("source must be a single Go file with package main and func main()")
	}

	sourceGUID, err := uuid.NewV7()
	if err != nil {
		return Result{}, fmt.Errorf("unable to create source code ID")
	}
	sourceID := sourceGUID.String()
	sourcePath := s.store.SourcePath(sourceID)
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		return Result{}, err
	}
	if err := os.WriteFile(sourcePath, []byte(code), 0o644); err != nil {
		return Result{}, err
	}
	if err := s.store.CreateSourceCode(sourceID, sourcePath, int64(len(code))); err != nil {
		return Result{}, err
	}
	submissionID, err := s.store.CreateSubmission(userID, sourceID)
	if err != nil {
		return Result{}, err
	}

	binaryPath := s.store.BinaryPath(submissionID)
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		return Result{}, err
	}
	buildStart := time.Now()
	output, buildErr := build(sourcePath, binaryPath)
	buildMS := time.Since(buildStart).Milliseconds()
	if buildErr != nil {
		msg := "build failed"
		_ = s.store.UpdateSubmissionBuild(submissionID, "invalid", msg, output, "")
		obs.SubmissionsInvalid.Add(1)
		zap.L().Info("submission.invalid", zap.Int64("submission_id", submissionID), zap.Int64("dur_ms", buildMS))
		return Result{SourceID: sourceID, SubmissionID: submissionID, Status: "invalid", Message: msg, Output: output}, nil
	}
	if err := s.store.UpdateSubmissionBuild(submissionID, "compiled", "build succeeded", output, binaryPath); err != nil {
		return Result{}, err
	}
	obs.SubmissionsCompiled.Add(1)
	zap.L().Info("submission.compiled", zap.Int64("submission_id", submissionID), zap.Int64("dur_ms", buildMS))

	agentID, err := s.store.CreateAgent(userID, submissionID, fmt.Sprintf("Submission #%d", submissionID))
	if err != nil {
		return Result{}, err
	}
	return Result{
		SourceID:     sourceID,
		SubmissionID: submissionID,
		AgentID:      agentID,
		Status:       "compiled",
		Message:      "build succeeded",
		Output:       output,
	}, nil
}

func build(sourcePath, binaryPath string) (string, error) {
	cmd := exec.Command("go", "build", "-o", binaryPath, "main.go")
	cmd.Dir = filepath.Dir(sourcePath)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}
