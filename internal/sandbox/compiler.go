package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"go.uber.org/zap"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"

	"progames/internal/artifact"
)

// Compiler compiles user-submitted Go source inside an isolated Docker container.
// Network access is disabled and the container is removed after each build.
type Compiler struct {
	cli   *client.Client
	image string
	repo  artifact.Repository
}

func NewCompiler(cli *client.Client, image string, repo artifact.Repository) *Compiler {
	return &Compiler{cli: cli, image: image, repo: repo}
}

func (c *Compiler) Build(ctx context.Context, sourceID artifact.ID) (artifact.ID, string, error) {
	src, err := c.repo.Read(ctx, sourceID)
	if err != nil {
		return "", "", fmt.Errorf("read source: %w", err)
	}
	defer func() { _ = src.Content.Close() }()
	srcBytes, err := io.ReadAll(src.Content)
	if err != nil {
		return "", "", fmt.Errorf("read source content: %w", err)
	}

	gomod := []byte("module bot\n\ngo " + goVersionFromImage(c.image) + "\n")
	var srcTar bytes.Buffer
	tw := tar.NewWriter(&srcTar)
	_ = tw.WriteHeader(&tar.Header{Name: "go.mod", Size: int64(len(gomod)), Mode: 0o644})
	_, _ = tw.Write(gomod)
	_ = tw.WriteHeader(&tar.Header{Name: "main.go", Size: int64(len(srcBytes)), Mode: 0o644})
	_, _ = tw.Write(srcBytes)
	_ = tw.Close()

	resp, err := c.cli.ContainerCreate(ctx, client.ContainerCreateOptions{
		Image: c.image,
		Config: &container.Config{
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
		return "", msg, fmt.Errorf("create build container: %w", err)
	}
	id := resp.ID
	defer func() {
		_, _ = c.cli.ContainerRemove(ctx, id, client.ContainerRemoveOptions{Force: true})
	}()

	if _, err := c.cli.CopyToContainer(ctx, id, client.CopyToContainerOptions{
		DestinationPath: "/src",
		Content:         bytes.NewReader(srcTar.Bytes()),
	}); err != nil {
		msg := fmt.Sprintf("copy source to container: %v", err)
		return "", msg, fmt.Errorf("copy source to container: %w", err)
	}

	if _, err := c.cli.ContainerStart(ctx, id, client.ContainerStartOptions{}); err != nil {
		msg := fmt.Sprintf("start build container: %v", err)
		return "", msg, fmt.Errorf("start build container: %w", err)
	}

	waitResult := c.cli.ContainerWait(ctx, id, client.ContainerWaitOptions{
		Condition: container.WaitConditionNotRunning,
	})
	var exitCode int64
	select {
	case err := <-waitResult.Error:
		if err != nil {
			return "", "", fmt.Errorf("wait for build container: %w", err)
		}
	case res := <-waitResult.Result:
		exitCode = res.StatusCode
	}

	logStream, err := c.cli.ContainerLogs(ctx, id, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return "", "", fmt.Errorf("get build logs: %w", err)
	}
	defer func() { _ = logStream.Close() }()
	var logBuf bytes.Buffer
	if _, err := stdcopy.StdCopy(&logBuf, &logBuf, logStream); err != nil {
		zap.L().Error("build.log_read_failed", zap.String("container_id", id), zap.Error(err))
	}
	output := logBuf.String()

	if exitCode != 0 {
		return "", output, fmt.Errorf("compiler exited with code %d", exitCode)
	}

	binResult, err := c.cli.CopyFromContainer(ctx, id, client.CopyFromContainerOptions{
		SourcePath: "/tmp/bot",
	})
	if err != nil {
		return "", output, fmt.Errorf("copy binary from container: %w", err)
	}
	defer func() { _ = binResult.Content.Close() }()

	tr := tar.NewReader(binResult.Content)
	if _, err := tr.Next(); err != nil {
		return "", output, fmt.Errorf("read binary tar entry: %w", err)
	}

	artifactID, err := c.repo.Write(ctx, tr)
	if err != nil {
		return "", output, fmt.Errorf("write binary: %w", err)
	}
	return artifactID, output, nil
}

// goVersionFromImage extracts the Go version from a builder image tag like
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
