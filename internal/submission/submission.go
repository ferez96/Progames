package submission

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
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

func (s *Service) Submit(ctx context.Context, userID int64, code string) (Result, error) {
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
	output, buildErr := s.build(ctx, sourcePath, binaryPath)
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

func (s *Service) build(ctx context.Context, sourcePath, binaryPath string) (string, error) {
	if s.dockerCli == nil {
		return buildWithProcess(ctx, sourcePath, binaryPath)
	}
	return buildWithDocker(ctx, s.dockerCli, s.cfg.GoBuilderImage, sourcePath, binaryPath)
}

func buildWithProcess(ctx context.Context, sourcePath, binaryPath string) (string, error) {
	src, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("read source: %w", err)
	}

	dir, err := os.MkdirTemp("", "progames-build-*")
	if err != nil {
		return "", fmt.Errorf("create build dir: %w", err)
	}
	defer os.RemoveAll(dir)

	goVer := strings.TrimPrefix(runtime.Version(), "go")
	gomod := "module bot\n\ngo " + goVer + "\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o644); err != nil {
		return "", fmt.Errorf("write go.mod: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), src, 0o644); err != nil {
		return "", fmt.Errorf("write main.go: %w", err)
	}

	cmd := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// goVersionFromImage extracts the Go language version from an image tag like
// "golang:1.26" or "golang:1.26-alpine", defaulting to "1.21" if unparseable.
func goVersionFromImage(image string) string {
	tag := image
	if i := strings.LastIndex(image, ":"); i >= 0 {
		tag = image[i+1:]
	}
	if i := strings.Index(tag, "-"); i >= 0 {
		tag = tag[:i]
	}
	if tag == "" || tag == "latest" {
		return "1.21"
	}
	return tag
}

func buildWithDocker(ctx context.Context, cli *client.Client, builderImage, sourcePath, binaryPath string) (string, error) {
	src, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("read source: %w", err)
	}

	// Pack source + go.mod into a tar for CopyToContainer. The go directive
	// matches the builder image version so language features up to that version
	// are available to user code.
	gomod := []byte("module bot\n\ngo " + goVersionFromImage(builderImage) + "\n")
	var srcTar bytes.Buffer
	tw := tar.NewWriter(&srcTar)
	_ = tw.WriteHeader(&tar.Header{Name: "go.mod", Size: int64(len(gomod)), Mode: 0o644})
	_, _ = tw.Write(gomod)
	_ = tw.WriteHeader(&tar.Header{Name: "main.go", Size: int64(len(src)), Mode: 0o644})
	_, _ = tw.Write(src)
	_ = tw.Close()

	resp, err := cli.ContainerCreate(ctx, client.ContainerCreateOptions{
		Image: builderImage,
		Config: &container.Config{
			// Use full path to avoid PATH lookup issues (Env below does not inherit image PATH).
			Cmd:        []string{"/usr/local/go/bin/go", "build", "-o", "/tmp/bot", "."},
			WorkingDir: "/src",
			Env: []string{
				"CGO_ENABLED=0",
				"GOOS=linux",
				"GOPATH=/tmp/gopath",
				"GOCACHE=/tmp/gocache",
				"HOME=/root",
			},
		},
		HostConfig: &container.HostConfig{
			NetworkMode: "none",
		},
	})
	if err != nil {
		msg := fmt.Sprintf("create build container: %v", err)
		return msg, fmt.Errorf("%s", msg)
	}
	id := resp.ID
	defer func() {
		_, _ = cli.ContainerRemove(ctx, id, client.ContainerRemoveOptions{Force: true})
	}()

	if _, err := cli.CopyToContainer(ctx, id, client.CopyToContainerOptions{
		DestinationPath: "/src",
		Content:         bytes.NewReader(srcTar.Bytes()),
	}); err != nil {
		msg := fmt.Sprintf("copy source to container: %v", err)
		return msg, fmt.Errorf("%s", msg)
	}

	if _, err := cli.ContainerStart(ctx, id, client.ContainerStartOptions{}); err != nil {
		msg := fmt.Sprintf("start build container: %v", err)
		return msg, fmt.Errorf("%s", msg)
	}

	waitResult := cli.ContainerWait(ctx, id, client.ContainerWaitOptions{
		Condition: container.WaitConditionNotRunning,
	})
	var exitCode int64
	select {
	case err := <-waitResult.Error:
		if err != nil {
			return "", fmt.Errorf("wait for build container: %w", err)
		}
	case res := <-waitResult.Result:
		exitCode = res.StatusCode
	}

	logStream, err := cli.ContainerLogs(ctx, id, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return "", fmt.Errorf("get build logs: %w", err)
	}
	defer func() { _ = logStream.Close() }()
	var logBuf bytes.Buffer
	_, _ = stdcopy.StdCopy(&logBuf, &logBuf, logStream)
	output := logBuf.String()

	if exitCode != 0 {
		return output, fmt.Errorf("compiler exited with code %d", exitCode)
	}

	binResult, err := cli.CopyFromContainer(ctx, id, client.CopyFromContainerOptions{
		SourcePath: "/tmp/bot",
	})
	if err != nil {
		return output, fmt.Errorf("copy binary from container: %w", err)
	}
	defer func() { _ = binResult.Content.Close() }()

	tr := tar.NewReader(binResult.Content)
	if _, err := tr.Next(); err != nil {
		return output, fmt.Errorf("read binary tar entry: %w", err)
	}
	f, err := os.OpenFile(binaryPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return output, fmt.Errorf("open binary for writing: %w", err)
	}
	if _, err := io.Copy(f, tr); err != nil {
		_ = f.Close()
		return output, fmt.Errorf("write binary: %w", err)
	}
	if err := f.Close(); err != nil {
		return output, fmt.Errorf("close binary: %w", err)
	}

	return output, nil
}
