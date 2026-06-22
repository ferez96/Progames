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
	x, y := systemBestMove(state)
	return MoveResult{RawLine: fmt.Sprintf("%d,%d", x, y), DurationMS: time.Since(start).Milliseconds()}, nil
}

func (s *SystemAgent) Close() error { return nil }

func (s *SystemAgent) Stderr() string { return "" }

const boardSize = 8

// offScore and defScore weight windows by number of marks already in them.
// Offensive weights are higher so a winning move beats blocking.
var offScore = [5]int{0, 10, 100, 1000, 100_000}
var defScore = [5]int{0, 9, 90, 900, 90_000}

var directions = [4][2]int{{1, 0}, {0, 1}, {1, 1}, {1, -1}}

func systemBestMove(state string) (int, int) {
	// Determine which mark belongs to the system agent.
	var xCount, oCount int
	for _, c := range state {
		switch c {
		case 'X':
			xCount++
		case 'O':
			oCount++
		}
	}
	var myMark, oppMark rune
	if xCount == oCount {
		myMark, oppMark = 'X', 'O'
	} else {
		myMark, oppMark = 'O', 'X'
	}

	bestVal := -1
	bestX, bestY := -1, -1

	for i, cell := range state {
		if cell != '.' {
			continue
		}
		cx, cy := i%boardSize, i/boardSize
		val := centerBonus(cx, cy) + cellScore(state, cx, cy, myMark, oppMark)
		if val > bestVal {
			bestVal = val
			bestX, bestY = cx+1, cy+1
		}
	}

	if bestX == -1 {
		for i, c := range state {
			if c == '.' {
				return i%boardSize + 1, i/boardSize + 1
			}
		}
	}
	return bestX, bestY
}

// cellScore sums scores from every window of 5 that passes through (cx, cy)
// in all four directions.
func cellScore(state string, cx, cy int, myMark, oppMark rune) int {
	total := 0
	for _, d := range directions {
		dx, dy := d[0], d[1]
		for offset := -4; offset <= 0; offset++ {
			sx, sy := cx+offset*dx, cy+offset*dy
			my, opp := 0, 0
			valid := true
			for i := 0; i < 5; i++ {
				nx, ny := sx+i*dx, sy+i*dy
				if nx < 0 || nx >= boardSize || ny < 0 || ny >= boardSize {
					valid = false
					break
				}
				switch rune(state[ny*boardSize+nx]) {
				case myMark:
					my++
				case oppMark:
					opp++
				}
			}
			if !valid || (my > 0 && opp > 0) {
				continue
			}
			if my > 0 {
				total += offScore[my]
			} else if opp > 0 {
				total += defScore[opp]
			}
		}
	}
	return total
}

func centerBonus(x, y int) int {
	mid := boardSize / 2
	return (boardSize - abs(x-mid) - abs(y-mid))
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
