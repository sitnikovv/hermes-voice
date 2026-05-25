package dispatch

import (
	"context"
	"time"

	"hermes-voice/internal/backend"
)

// Task is the minimal fire-and-forget background work shape returned after a quick timeout.
type Task struct {
	ID      string
	Request backend.Request
}

// TaskRunner observes or registers accepted fallback work after the original backend
// invocation has crossed the quick-response budget. It must not implicitly invoke the
// same backend request again; result storage/retrieval is introduced by later goals.
type TaskRunner interface {
	Start(ctx context.Context, task Task)
}

// TaskIDFunc generates task IDs for accepted fallback responses.
type TaskIDFunc func(backend.Request) string

// Config configures a Dispatcher wrapper around a backend adapter.
type Config struct {
	Backend      backend.Adapter
	Runner       TaskRunner
	QuickTimeout time.Duration
	TaskID       TaskIDFunc
}
