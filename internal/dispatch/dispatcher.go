package dispatch

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"
	"unicode"

	"hermes-voice/internal/backend"
	"hermes-voice/internal/taskstore"
)

const defaultQuickTimeout = 1500 * time.Millisecond

var defaultTaskCounter uint64

// Dispatcher wraps a backend adapter with a quick-response budget and minimal accepted fallback.
type Dispatcher struct {
	backend      backend.Adapter
	runner       TaskRunner
	store        taskstore.Store
	quickTimeout time.Duration
	taskID       TaskIDFunc
}

// New returns a Dispatcher with validated dependencies and defaults.
func New(cfg Config) (*Dispatcher, error) {
	if cfg.Backend == nil {
		return nil, errors.New("dispatch: backend is required")
	}
	if cfg.Runner == nil && cfg.Store == nil {
		return nil, errors.New("dispatch: task store or runner is required when fallback is enabled")
	}
	quickTimeout := cfg.QuickTimeout
	if quickTimeout <= 0 {
		quickTimeout = defaultQuickTimeout
	}
	taskID := cfg.TaskID
	if taskID == nil {
		taskID = defaultTaskID
	}
	return &Dispatcher{backend: cfg.Backend, runner: cfg.Runner, store: cfg.Store, quickTimeout: quickTimeout, taskID: taskID}, nil
}

// Invoke runs the backend until it responds, the parent context is canceled, or the quick timeout expires.
// The backend invoke context is dispatcher-owned: parent cancellation before the
// timeout cancels it, but after an accepted fallback the original invoke may continue
// detached from the caller while TaskRunner records the accepted handoff.
func (d *Dispatcher) Invoke(ctx context.Context, req backend.Request) (*backend.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	invokeCtx, cancelInvoke := context.WithCancel(context.Background())

	result := make(chan invokeResult, 1)
	go func() {
		defer cancelInvoke()
		resp, err := d.backend.Invoke(invokeCtx, cloneRequest(req))
		result <- invokeResult{resp: resp, err: err}
	}()

	timer := time.NewTimer(d.quickTimeout)
	defer timer.Stop()

	select {
	case res := <-result:
		return res.resp, res.err
	case <-ctx.Done():
		cancelInvoke()
		return nil, ctx.Err()
	case <-timer.C:
		taskID := normalizeTaskID(d.taskID(req))
		if taskID == "" {
			taskID = defaultTaskID(req)
		}
		if d.store != nil {
			if err := d.store.CreateAccepted(context.Background(), taskstore.Record{
				TaskID:    taskID,
				RequestID: req.ID,
				Request:   cloneRequest(req),
				Metadata: map[string]string{
					"accepted_by": "dispatcher",
					"reason":      "quick_timeout",
				},
			}); err != nil {
				cancelInvoke()
				return nil, err
			}
			go d.storeBackendResult(context.Background(), taskID, result)
		}
		if d.runner != nil {
			go d.runner.Start(context.Background(), Task{ID: taskID, Request: cloneRequest(req)})
		}
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

func (d *Dispatcher) storeBackendResult(ctx context.Context, taskID string, result <-chan invokeResult) {
	res := <-result
	if res.err != nil {
		_ = d.store.Fail(ctx, taskID, safeTaskError(res.err))
		return
	}
	if res.resp == nil {
		_ = d.store.Fail(ctx, taskID, taskstore.Error{Code: "internal_error", Message: "backend returned nil response"})
		return
	}
	if res.resp.Status == backend.StatusFailed {
		_ = d.store.Fail(ctx, taskID, taskstore.Error{Code: "backend_failed", Message: "backend returned failed status"})
		return
	}
	_ = d.store.Complete(ctx, taskID, res.resp)
}

func cloneRequest(req backend.Request) backend.Request {
	cloned := req
	if req.Metadata != nil {
		cloned.Metadata = make(map[string]string, len(req.Metadata))
		for key, value := range req.Metadata {
			cloned.Metadata[key] = value
		}
	}
	return cloned
}

func normalizeTaskID(id string) string {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return ""
	}
	return strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || unicode.IsSpace(r) {
			return '-'
		}
		return r
	}, trimmed)
}

func safeTaskError(err error) taskstore.Error {
	switch {
	case errors.Is(err, context.Canceled):
		return taskstore.Error{Code: "context_canceled", Message: "backend invocation canceled"}
	case errors.Is(err, context.DeadlineExceeded):
		return taskstore.Error{Code: "context_deadline_exceeded", Message: "backend invocation deadline exceeded"}
	case errors.Is(err, backend.ErrInvalidRequest):
		return taskstore.Error{Code: "backend_invalid_request", Message: "backend request is invalid"}
	case errors.Is(err, backend.ErrUnauthorized):
		return taskstore.Error{Code: "backend_unauthorized", Message: "backend unauthorized"}
	case errors.Is(err, backend.ErrTemporary):
		return taskstore.Error{Code: "backend_temporary", Message: "backend temporarily unavailable"}
	case errors.Is(err, backend.ErrInvocationFailed):
		return taskstore.Error{Code: "backend_invocation_failed", Message: "backend invocation failed"}
	default:
		return taskstore.Error{Code: "backend_error", Message: "backend failed"}
	}
}

func defaultTaskID(req backend.Request) string {
	n := atomic.AddUint64(&defaultTaskCounter, 1)
	if normalized := normalizeTaskID(req.ID); normalized != "" {
		return fmt.Sprintf("%s-task-%d", normalized, n)
	}
	return fmt.Sprintf("task-%d", n)
}
