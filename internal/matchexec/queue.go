package matchexec

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"progames/internal/service"
)

// Queue manages the lifecycle of async match execution.
// The worker goroutine runs until ctx is cancelled; call Wait to block until it exits.
type Queue struct {
	proc *Processor
	jobs chan preparedJob
	wg   sync.WaitGroup
}

func NewQueue(ctx context.Context, proc *Processor, cap int) *Queue {
	q := &Queue{proc: proc, jobs: make(chan preparedJob, cap)}
	q.wg.Add(1)
	go q.work(ctx)
	return q
}

func (q *Queue) work(ctx context.Context) {
	defer q.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case pj, ok := <-q.jobs:
			if !ok {
				return
			}
			q.proc.execute(pj)
		}
	}
}

// Wait blocks until the worker goroutine has exited.
func (q *Queue) Wait() {
	q.wg.Wait()
}

// Enqueue prepares a match synchronously (creates DB record, returns matchID)
// then queues the execution asynchronously.
func (q *Queue) Enqueue(ctx context.Context, job service.MatchJob) (int64, error) {
	pj, err := q.proc.prepare(job)
	if err != nil {
		return 0, err
	}
	select {
	case <-ctx.Done():
		now := time.Now().UTC()
		_ = q.proc.store.FailMatch(pj.matchID, "server shutting down", now, now.Sub(pj.startedAt).Milliseconds())
		return 0, ctx.Err()
	case q.jobs <- pj:
		return pj.matchID, nil
	default:
		now := time.Now().UTC()
		_ = q.proc.store.FailMatch(pj.matchID, "server busy", now, now.Sub(pj.startedAt).Milliseconds())
		zap.L().Warn("match.rejected", zap.Int64("match_id", pj.matchID))
		return pj.matchID, fmt.Errorf("server busy, try again later")
	}
}
