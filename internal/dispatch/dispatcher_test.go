package dispatch

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"hermes-voice/internal/backend"
)

func TestNewRejectsNilBackend(t *testing.T) {
	_, err := New(Config{Runner: spyRunner{}})
	if err == nil {
		t.Fatal("New() error = nil, want nil backend rejection")
	}
}

func TestNewRejectsNilRunnerWhenFallbackEnabled(t *testing.T) {
	_, err := New(Config{Backend: immediateAdapter{resp: &backend.Response{Status: backend.StatusCompleted}}, QuickTimeout: time.Millisecond})
	if err == nil {
		t.Fatal("New() error = nil, want nil runner rejection")
	}
}

func TestDispatcherImplementsBackendAdapter(t *testing.T) {
	var _ backend.Adapter = (*Dispatcher)(nil)
}

func TestNewAppliesDefaults(t *testing.T) {
	d, err := New(Config{Backend: immediateAdapter{resp: &backend.Response{Status: backend.StatusCompleted}}, Runner: spyRunner{}})
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}
	if d.quickTimeout <= 0 {
		t.Fatalf("quickTimeout = %v, want positive default", d.quickTimeout)
	}
	if got := d.taskID(validRequest()); got == "" {
		t.Fatal("default task ID is empty")
	}
}

type immediateAdapter struct {
	resp *backend.Response
	err  error
}

func (a immediateAdapter) Invoke(context.Context, backend.Request) (*backend.Response, error) {
	return a.resp, a.err
}

type blockingAdapter struct {
	started chan struct{}
	release chan struct{}
	resp    *backend.Response
	err     error
}

func (a blockingAdapter) Invoke(ctx context.Context, req backend.Request) (*backend.Response, error) {
	close(a.started)
	select {
	case <-a.release:
		return a.resp, a.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

type spyRunner struct {
	started chan Task
	ctx     chan context.Context
}

func (r spyRunner) Start(ctx context.Context, task Task) {
	if r.ctx != nil {
		r.ctx <- ctx
	}
	if r.started != nil {
		r.started <- task
	}
}

func validRequest() backend.Request {
	return backend.Request{
		ID:        "req-1",
		Input:     "hello",
		PersonID:  "person",
		ProfileID: "profile",
		ModelID:   "model",
		BackendID: "backend",
		ModelName: "test-model",
	}
}

func TestInvokeReturnsFastResponseUnchangedAndDoesNotStartRunner(t *testing.T) {
	want := &backend.Response{ID: "resp-1", Status: backend.StatusCompleted, Output: "done", Metadata: map[string]string{"k": "v"}}
	runner := spyRunner{started: make(chan Task, 1)}
	d, err := New(Config{Backend: immediateAdapter{resp: want}, Runner: runner, QuickTimeout: time.Second})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	got, err := d.Invoke(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if got != want {
		t.Fatalf("Invoke() response pointer changed: got %p want %p", got, want)
	}
	select {
	case task := <-runner.started:
		t.Fatalf("runner started on fast success: %+v", task)
	default:
	}
}

func TestInvokeReturnsFastErrorUnchangedAndDoesNotStartRunner(t *testing.T) {
	wantErr := errors.New("boom")
	runner := spyRunner{started: make(chan Task, 1)}
	d, err := New(Config{Backend: immediateAdapter{err: wantErr}, Runner: runner, QuickTimeout: time.Second})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	got, err := d.Invoke(context.Background(), validRequest())
	if !errors.Is(err, wantErr) {
		t.Fatalf("Invoke() error = %v, want %v", err, wantErr)
	}
	if got != nil {
		t.Fatalf("Invoke() response = %+v, want nil", got)
	}
	select {
	case task := <-runner.started:
		t.Fatalf("runner started on fast error: %+v", task)
	default:
	}
}

func TestInvokeParentCanceledBeforeTimeoutReturnsContextErrorNoRunner(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	adapter := blockingAdapter{started: make(chan struct{}), release: make(chan struct{})}
	runner := spyRunner{started: make(chan Task, 1)}
	d, err := New(Config{Backend: adapter, Runner: runner, QuickTimeout: time.Hour})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	result := make(chan error, 1)
	go func() {
		_, err := d.Invoke(ctx, validRequest())
		result <- err
	}()
	<-adapter.started
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("Invoke() error = %v, want context canceled", err)
	}
	select {
	case task := <-runner.started:
		t.Fatalf("runner started on parent cancel: %+v", task)
	default:
	}
}

func TestInvokeQuickTimeoutReturnsAcceptedAndStartsRunner(t *testing.T) {
	req := validRequest()
	req.ID = "req-timeout"
	adapter := blockingAdapter{started: make(chan struct{}), release: make(chan struct{})}
	runner := spyRunner{started: make(chan Task, 1), ctx: make(chan context.Context, 1)}
	d, err := New(Config{
		Backend:      adapter,
		Runner:       runner,
		QuickTimeout: time.Millisecond,
		TaskID:       func(backend.Request) string { return "task-static" },
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	resp, err := d.Invoke(ctx, req)
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if resp.ID != req.ID || resp.Status != backend.StatusAccepted || resp.TaskID != "task-static" {
		t.Fatalf("accepted response = %+v, want req id, accepted, deterministic task id", resp)
	}
	wantMeta := map[string]string{"accepted_by": "dispatcher", "reason": "quick_timeout"}
	if !reflect.DeepEqual(resp.Metadata, wantMeta) {
		t.Fatalf("metadata = %#v, want %#v", resp.Metadata, wantMeta)
	}
	var task Task
	select {
	case task = <-runner.started:
	case <-time.After(time.Second):
		t.Fatal("runner was not started")
	}
	if task.ID != "task-static" || !reflect.DeepEqual(task.Request, req) {
		t.Fatalf("task = %+v, want original request and deterministic id", task)
	}
	runnerCtx := <-runner.ctx
	cancel()
	if err := runnerCtx.Err(); err != nil {
		t.Fatalf("runner context Err() = %v, want detached from parent cancel", err)
	}
}

func TestInvokeQuickTimeoutFallsBackWhenTaskIDFuncReturnsEmpty(t *testing.T) {
	adapter := blockingAdapter{started: make(chan struct{}), release: make(chan struct{})}
	runner := spyRunner{started: make(chan Task, 1)}
	d, err := New(Config{
		Backend:      adapter,
		Runner:       runner,
		QuickTimeout: time.Millisecond,
		TaskID:       func(backend.Request) string { return "" },
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	resp, err := d.Invoke(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if resp.TaskID == "" {
		t.Fatal("TaskID is empty, want non-empty fallback")
	}
}
