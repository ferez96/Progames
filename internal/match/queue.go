package match

import (
	"fmt"
	"sync"

	"go.uber.org/zap"
)

const queueCap = 4

type Queue struct {
	svc  *Service
	jobs chan pendingMatch
	wg   sync.WaitGroup
}

func NewQueue(svc *Service) *Queue {
	q := &Queue{svc: svc, jobs: make(chan pendingMatch, queueCap)}
	q.wg.Add(1)
	go q.work()
	return q
}

func (q *Queue) work() {
	defer q.wg.Done()
	for p := range q.jobs {
		q.svc.run(p)
	}
}

// Shutdown stops the queue from accepting new jobs and waits for the
// in-flight match to finish.
func (q *Queue) Shutdown() {
	close(q.jobs)
	q.wg.Wait()
}

func (q *Queue) Enqueue(userAgentID, systemAgentID int64) (int64, error) {
	p, err := q.svc.prepare(userAgentID, systemAgentID)
	if err != nil {
		return 0, err
	}
	select {
	case q.jobs <- p:
		return p.matchID, nil
	default:
		_ = q.svc.store.FailMatch(p.matchID, "server busy", p.startedAt)
		zap.L().Warn("match.rejected", zap.Int64("match_id", p.matchID))
		return p.matchID, fmt.Errorf("server busy, try again later")
	}
}
