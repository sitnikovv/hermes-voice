package backend

import (
	"context"
	"errors"
	"testing"
)

func TestStaticAdapterImplementsAdapter(t *testing.T) {
	var _ Adapter = NewStaticAdapter(Response{})
}

func TestStaticAdapterReturnsConfiguredCompletedResponse(t *testing.T) {
	want := Response{
		ID:     "resp-1",
		Output: "done",
		Status: StatusCompleted,
		Usage:  &Usage{InputTokens: 3, OutputTokens: 5},
		Metadata: map[string]string{
			"backend": "static",
		},
	}

	got, err := NewStaticAdapter(want).Invoke(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("Invoke() error = %v, want nil", err)
	}
	if got == nil {
		t.Fatal("Invoke() response = nil, want configured response")
	}
	if got.ID != want.ID || got.Output != want.Output || got.Status != want.Status {
		t.Fatalf("Invoke() response = %#v, want %#v", got, want)
	}
	if got.Usage == nil || *got.Usage != *want.Usage {
		t.Fatalf("Invoke() usage = %#v, want %#v", got.Usage, want.Usage)
	}
	if got.Metadata["backend"] != "static" {
		t.Fatalf("Invoke() metadata = %#v, want configured metadata", got.Metadata)
	}
}

func TestStaticAdapterReturnsConfiguredAcceptedResponseWithTaskID(t *testing.T) {
	want := Response{ID: "resp-1", Status: StatusAccepted, TaskID: "task-1"}

	got, err := NewStaticAdapter(want).Invoke(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("Invoke() error = %v, want nil", err)
	}
	if got == nil || got.Status != StatusAccepted || got.TaskID != "task-1" {
		t.Fatalf("Invoke() response = %#v, want accepted response with task ID", got)
	}
}

func TestStaticAdapterRejectsInvalidRequest(t *testing.T) {
	req := validRequest()
	req.Input = ""

	_, err := NewStaticAdapter(Response{Status: StatusCompleted}).Invoke(context.Background(), req)
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Invoke() error = %v, want ErrInvalidRequest", err)
	}
}

func TestStaticAdapterRespectsAlreadyCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := NewStaticAdapter(Response{Status: StatusCompleted}).Invoke(ctx, validRequest())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Invoke() error = %v, want context.Canceled", err)
	}
}

func TestStaticAdapterReturnsConfiguredError(t *testing.T) {
	want := &Error{Op: "invoke", BackendID: "backend-1", Code: "failed", Err: ErrInvocationFailed}

	_, err := NewErrorAdapter(want).Invoke(context.Background(), validRequest())
	if !errors.Is(err, ErrInvocationFailed) {
		t.Fatalf("Invoke() error = %v, want ErrInvocationFailed", err)
	}
	if !errors.Is(err, want) {
		t.Fatalf("Invoke() error = %v, want configured error", err)
	}
}
