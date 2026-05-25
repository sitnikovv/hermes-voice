package dispatch

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"hermes-voice/internal/backend"
)

const defaultQuickTimeout = 1500 * time.Millisecond

var defaultTaskCounter uint64

// Dispatcher wraps a backend adapter with a quick-response budget and minimal accepted fallback.
type Dispatcher struct {
	backend      backend.Adapter
	runner       TaskRunner
	quickTimeout time.Duration
	taskID       TaskIDFunc
}

// New returns a Dispatcher with validated dependencies and defaults.
func New(cfg Config) (*Dispatcher, error) {
	if cfg.Backend == nil {
		return nil, errors.New("dispatch: backend is required")
	}
	if cfg.Runner == nil {
		return nil, errors.New("dispatch: runner is required when fallback is enabled")
	}
	quickTimeout := cfg.QuickTimeout
	if quickTimeout <= 0 {
		quickTimeout = defaultQuickTimeout
	}
	taskID := cfg.TaskID
	if taskID == nil {
		taskID = defaultTaskID
	}
	return &Dispatcher{backend: cfg.Backend, runner: cfg.Runner, quickTimeout: quickTimeout, taskID: taskID}, nil
}

// Invoke currently delegates to the wrapped backend; timeout fallback behavior is added in later tasks.
func (d *Dispatcher) Invoke(ctx context.Context, req backend.Request) (*backend.Response, error) {
	return d.backend.Invoke(ctx, req)
}

func defaultTaskID(req backend.Request) string {
	n := atomic.AddUint64(&defaultTaskCounter, 1)
	if strings.TrimSpace(req.ID) != "" {
		return fmt.Sprintf("%s-task-%d", req.ID, n)
	}
	return fmt.Sprintf("task-%d", n)
}
