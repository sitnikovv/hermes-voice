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

// Invoke runs the backend until it responds, the parent context is canceled, or the quick timeout expires.
func (d *Dispatcher) Invoke(ctx context.Context, req backend.Request) (*backend.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	invokeCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	result := make(chan invokeResult, 1)
	go func() {
		resp, err := d.backend.Invoke(invokeCtx, req)
		result <- invokeResult{resp: resp, err: err}
	}()

	timer := time.NewTimer(d.quickTimeout)
	defer timer.Stop()

	select {
	case res := <-result:
		return res.resp, res.err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		cancel()
		taskID := strings.TrimSpace(d.taskID(req))
		if taskID == "" {
			taskID = defaultTaskID(req)
		}
		d.runner.Start(context.Background(), Task{ID: taskID, Request: req})
		return &backend.Response{
			ID:     req.ID,
			Status: backend.StatusAccepted,
			TaskID: taskID,
			Metadata: map[string]string{
				"accepted_by": "dispatcher",
				"reason":      "quick_timeout",
			},
		}, nil
	}
}

type invokeResult struct {
	resp *backend.Response
	err  error
}

func defaultTaskID(req backend.Request) string {
	n := atomic.AddUint64(&defaultTaskCounter, 1)
	if strings.TrimSpace(req.ID) != "" {
		return fmt.Sprintf("%s-task-%d", req.ID, n)
	}
	return fmt.Sprintf("task-%d", n)
}
