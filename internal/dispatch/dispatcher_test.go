package dispatch

import (
	"context"
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

type spyRunner struct{}

func (spyRunner) Start(context.Context, Task) {}

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
