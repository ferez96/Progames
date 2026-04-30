package runner

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type AgentRunner interface {
	Start() error
	Move(state string, timeout time.Duration) (MoveResult, error)
	Close() error
	Stderr() string
}

type MoveResult struct {
	RawLine    string
	DurationMS int64
}

type ProcessRunner struct {
	BinaryPath string
	MaxLine    int
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     *bufio.Reader
	stderr     bytes.Buffer
	stderrMu   sync.Mutex
}

func NewProcess(binaryPath string, maxLine int) *ProcessRunner {
	return &ProcessRunner{BinaryPath: binaryPath, MaxLine: maxLine}
}

func (r *ProcessRunner) Start() error {
	cmd := exec.Command(r.BinaryPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	r.cmd = cmd
	r.stdin = stdin
	r.stdout = bufio.NewReader(stdout)
	go r.captureStderr(stderr)
	return nil
}

func (r *ProcessRunner) Move(state string, timeout time.Duration) (MoveResult, error) {
	if r.cmd == nil {
		return MoveResult{}, errors.New("bot process not started")
	}
	start := time.Now()
	if _, err := io.WriteString(r.stdin, state+"\n"); err != nil {
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
		if r.MaxLine > 0 && len(line) > r.MaxLine {
			return MoveResult{RawLine: line, DurationMS: time.Since(start).Milliseconds()}, fmt.Errorf("stdout line exceeds %d bytes", r.MaxLine)
		}
		return MoveResult{RawLine: line, DurationMS: time.Since(start).Milliseconds()}, nil
	case <-time.After(timeout):
		_ = r.Close()
		return MoveResult{DurationMS: timeout.Milliseconds()}, errors.New("timeout")
	}
}

func (r *ProcessRunner) Close() error {
	if r.stdin != nil {
		_ = r.stdin.Close()
	}
	if r.cmd == nil || r.cmd.Process == nil {
		return nil
	}
	_ = r.cmd.Process.Kill()
	_ = r.cmd.Wait()
	return nil
}

func (r *ProcessRunner) Stderr() string {
	r.stderrMu.Lock()
	defer r.stderrMu.Unlock()
	return r.stderr.String()
}

func (r *ProcessRunner) captureStderr(reader io.Reader) {
	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			r.stderrMu.Lock()
			r.stderr.Write(buf[:n])
			r.stderrMu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

type SystemAgent struct{}

func NewSystemAgent() *SystemAgent { return &SystemAgent{} }

func (s *SystemAgent) Start() error { return nil }

func (s *SystemAgent) Move(state string, _ time.Duration) (MoveResult, error) {
	start := time.Now()
	for i, cell := range state {
		if cell == '.' {
			x := i%8 + 1
			y := i/8 + 1
			return MoveResult{RawLine: fmt.Sprintf("%d,%d", x, y), DurationMS: time.Since(start).Milliseconds()}, nil
		}
	}
	return MoveResult{RawLine: "1,1", DurationMS: time.Since(start).Milliseconds()}, nil
}

func (s *SystemAgent) Close() error { return nil }

func (s *SystemAgent) Stderr() string { return "" }
