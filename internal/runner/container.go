package runner

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

type ContainerRunner struct {
	cli         *client.Client
	imageTag    string
	maxLine     int
	memBytes    int64
	nanoCPUs    int64
	containerID string
	attach      *client.ContainerAttachResult
	stdout      *bufio.Reader
	stderr      safeBuffer
}

func NewContainer(cli *client.Client, imageTag string, maxLine int, memBytes, nanoCPUs int64) *ContainerRunner {
	return &ContainerRunner{cli: cli, imageTag: imageTag, maxLine: maxLine, memBytes: memBytes, nanoCPUs: nanoCPUs}
}

func (r *ContainerRunner) Start() error {
	ctx := context.Background()

	created, err := r.cli.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config: &container.Config{
			Image:        r.imageTag,
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			OpenStdin:    true,
			StdinOnce:    false,
		},
		HostConfig: &container.HostConfig{
			NetworkMode: "none",
			AutoRemove:  true,
			Resources: container.Resources{
				Memory:   r.memBytes,
				NanoCPUs: r.nanoCPUs,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("container create: %w", err)
	}
	r.containerID = created.ID

	attach, err := r.cli.ContainerAttach(ctx, r.containerID, client.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return fmt.Errorf("container attach: %w", err)
	}
	r.attach = &attach

	if _, err := r.cli.ContainerStart(ctx, r.containerID, client.ContainerStartOptions{}); err != nil {
		r.attach.Close()
		return fmt.Errorf("container start: %w", err)
	}

	stdoutR, stdoutW := io.Pipe()
	go func() {
		defer func() { _ = stdoutW.Close() }()
		_, _ = stdcopy.StdCopy(stdoutW, &r.stderr, attach.Reader)
	}()

	r.stdout = bufio.NewReader(stdoutR)
	return nil
}

func (r *ContainerRunner) Move(state string, timeout time.Duration) (MoveResult, error) {
	if r.attach == nil {
		return MoveResult{}, fmt.Errorf("container not started")
	}
	start := time.Now()
	if _, err := io.WriteString(r.attach.Conn, state+"\n"); err != nil {
		return MoveResult{}, err
	}

	type readResult struct {
		line string
		err  error
	}
	ch := make(chan readResult, 1)
	go func() {
		line, err := r.stdout.ReadString('\n')
		ch <- readResult{line: line, err: err}
	}()

	select {
	case got := <-ch:
		if got.err != nil {
			return MoveResult{}, got.err
		}
		line := strings.TrimSpace(got.line)
		if r.maxLine > 0 && len(line) > r.maxLine {
			return MoveResult{RawLine: line, DurationMS: time.Since(start).Milliseconds()}, fmt.Errorf("stdout line exceeds %d bytes", r.maxLine)
		}
		return MoveResult{RawLine: line, DurationMS: time.Since(start).Milliseconds()}, nil
	case <-time.After(timeout):
		_ = r.Close()
		return MoveResult{DurationMS: timeout.Milliseconds()}, fmt.Errorf("timeout")
	}
}

func (r *ContainerRunner) Close() error {
	if r.attach != nil {
		r.attach.Close()
		r.attach = nil
	}
	if r.containerID == "" {
		return nil
	}
	ctx := context.Background()
	timeout := 0
	_, _ = r.cli.ContainerStop(ctx, r.containerID, client.ContainerStopOptions{Timeout: &timeout})
	return nil
}

func (r *ContainerRunner) Stderr() string {
	return r.stderr.String()
}
