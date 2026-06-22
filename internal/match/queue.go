package match

import (
	"fmt"

	"go.uber.org/zap"
)

const queueCap = 4

type Queue struct {
	svc  *Service
	jobs chan pendingMatch
}

func NewQueue(svc *Service) *Queue {
	q := &Queue{svc: svc, jobs: make(chan pendingMatch, queueCap)}
	go q.work()
	return q
}

func (q *Queue) work() {
	for p := range q.jobs {
		q.svc.run(p)
	}
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
