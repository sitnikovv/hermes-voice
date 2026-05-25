package taskstore

import (
	"context"
	"strings"
	"sync"
	"time"

	"hermes-voice/internal/backend"
)

type MemoryStore struct {
	mu      sync.RWMutex
	records map[string]Record
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{records: make(map[string]Record)}
}

func (s *MemoryStore) CreateAccepted(ctx context.Context, rec Record) error {
	_ = ctx
	taskID := strings.TrimSpace(rec.TaskID)
	if taskID == "" {
		return ErrInvalidTaskID
	}
	now := time.Now()
	stored := cloneRecord(rec)
	stored.TaskID = taskID
	stored.Status = StatusAccepted
	if stored.RequestID == "" {
		stored.RequestID = stored.Request.ID
	}
	if stored.CreatedAt.IsZero() {
		stored.CreatedAt = now
	}
	if stored.UpdatedAt.IsZero() {
		stored.UpdatedAt = stored.CreatedAt
	}
	stored.CompletedAt = nil
	stored.Response = nil
	stored.Error = nil

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.records[taskID]; ok {
		return ErrTaskExists
	}
	s.records[taskID] = stored
	return nil
}

func (s *MemoryStore) Complete(ctx context.Context, taskID string, resp *backend.Response) error {
	_ = ctx
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return ErrInvalidTaskID
	}
	if resp == nil {
		return ErrInvalidResponse
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.records[taskID]
	if !ok {
		return ErrTaskNotFound
	}
	if rec.Status != StatusAccepted {
		return ErrInvalidTransition
	}
	now := time.Now()
	rec.Status = StatusCompleted
	rec.Response = cloneResponse(resp)
	rec.Error = nil
	rec.UpdatedAt = now
	rec.CompletedAt = &now
	s.records[taskID] = rec
	return nil
}

func (s *MemoryStore) Fail(ctx context.Context, taskID string, taskErr Error) error {
	_ = ctx
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return ErrInvalidTaskID
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.records[taskID]
	if !ok {
		return ErrTaskNotFound
	}
	if rec.Status != StatusAccepted {
		return ErrInvalidTransition
	}
	now := time.Now()
	errCopy := taskErr
	rec.Status = StatusFailed
	rec.Response = nil
	rec.Error = &errCopy
	rec.UpdatedAt = now
	rec.CompletedAt = &now
	s.records[taskID] = rec
	return nil
}

func (s *MemoryStore) Get(ctx context.Context, taskID string) (Record, bool, error) {
	_ = ctx
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return Record{}, false, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.records[taskID]
	if !ok {
		return Record{}, false, nil
	}
	return cloneRecord(rec), true, nil
}

func cloneRecord(rec Record) Record {
	cloned := rec
	cloned.Request = cloneRequest(rec.Request)
	cloned.Response = cloneResponse(rec.Response)
	if rec.Error != nil {
		errCopy := *rec.Error
		cloned.Error = &errCopy
	}
	if rec.CompletedAt != nil {
		completedAt := *rec.CompletedAt
		cloned.CompletedAt = &completedAt
	}
	cloned.Metadata = cloneStringMap(rec.Metadata)
	return cloned
}

func cloneRequest(req backend.Request) backend.Request {
	cloned := req
	cloned.Metadata = cloneStringMap(req.Metadata)
	return cloned
}

func cloneResponse(resp *backend.Response) *backend.Response {
	if resp == nil {
		return nil
	}
	cloned := *resp
	if resp.Usage != nil {
		usage := *resp.Usage
		cloned.Usage = &usage
	}
	cloned.Metadata = cloneStringMap(resp.Metadata)
	return &cloned
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
