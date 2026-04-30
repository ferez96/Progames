package events

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"progames/internal/store"
)

type Store struct {
	store *store.Store
}

func New(st *store.Store) *Store {
	return &Store{store: st}
}

func (s *Store) Append(matchID int64, gameID sql.NullInt64, typ string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return s.store.AppendEvent(matchID, gameID, typ, string(raw))
}

func (s *Store) ProjectMove(gameID, seq, agentID int64, actionType string, payload any, accepted bool, durationMS sql.NullInt64) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return s.store.InsertMove(gameID, seq, agentID, actionType, string(raw), accepted, durationMS)
}

func (s *Store) RenderExecutionLog(matchID int64, maxBytes int) error {
	events, err := s.store.ListEvents(matchID)
	if err != nil {
		return err
	}
	var b strings.Builder
	truncated := false
	for _, event := range events {
		line := fmt.Sprintf("[%03d] %s %s\n", event.Seq, event.Type, event.Payload)
		if maxBytes > 0 && b.Len()+len(line) > maxBytes {
			omitted := b.Len() + len(line) - maxBytes
			b.WriteString(fmt.Sprintf("\n--- log truncated (%d bytes omitted) ---", omitted))
			truncated = true
			break
		}
		b.WriteString(line)
	}
	agentLogs, err := s.store.AgentLogs(matchID)
	if err != nil {
		return err
	}
	for _, log := range agentLogs {
		if log.Content == "" {
			continue
		}
		block := fmt.Sprintf("\n--- agent:%d stderr ---\n%s\n", log.AgentID, log.Content)
		if maxBytes > 0 && b.Len()+len(block) > maxBytes {
			omitted := b.Len() + len(block) - maxBytes
			b.WriteString(fmt.Sprintf("\n--- log truncated (%d bytes omitted) ---", omitted))
			truncated = true
			break
		}
		b.WriteString(block)
	}
	return s.store.UpsertExecutionLog(matchID, b.String(), truncated)
}
