package taskstore

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"hermes-voice/internal/backend"
)

func TestMemoryStoreCreateAcceptedAndGet(t *testing.T) {
	store := NewMemoryStore()
	rec := Record{TaskID: "task-1", RequestID: "req-1", Request: backend.Request{ID: "req-1", Metadata: map[string]string{"rk": "rv"}}, Metadata: map[string]string{"mk": "mv"}}
	if err := store.CreateAccepted(context.Background(), rec); err != nil {
		t.Fatalf("CreateAccepted() error = %v", err)
	}
	got, found, err := store.Get(context.Background(), "task-1")
	if err != nil || !found {
		t.Fatalf("Get() = found %v err %v, want found nil", found, err)
	}
	if got.TaskID != "task-1" || got.RequestID != "req-1" || got.Status != StatusAccepted {
		t.Fatalf("record = %+v, want accepted task", got)
	}
	if got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Fatalf("timestamps not set: %+v", got)
	}
}

func TestMemoryStoreCreateAcceptedRejectsDuplicateAndEmptyTaskID(t *testing.T) {
	store := NewMemoryStore()
	if err := store.CreateAccepted(context.Background(), Record{}); !errors.Is(err, ErrInvalidTaskID) {
		t.Fatalf("empty task id error = %v, want ErrInvalidTaskID", err)
	}
	if err := store.CreateAccepted(context.Background(), Record{TaskID: "task-1"}); err != nil {
		t.Fatalf("CreateAccepted() error = %v", err)
	}
	if err := store.CreateAccepted(context.Background(), Record{TaskID: "task-1"}); !errors.Is(err, ErrTaskExists) {
		t.Fatalf("duplicate error = %v, want ErrTaskExists", err)
	}
}

func TestMemoryStoreCompleteTransitionsAcceptedToCompleted(t *testing.T) {
	store := NewMemoryStore()
	if err := store.CreateAccepted(context.Background(), Record{TaskID: "task-1"}); err != nil {
		t.Fatal(err)
	}
	resp := &backend.Response{ID: "resp-1", Status: backend.StatusCompleted, Output: "done", Usage: &backend.Usage{InputTokens: 1, OutputTokens: 2}, Metadata: map[string]string{"k": "v"}}
	if err := store.Complete(context.Background(), "task-1", resp); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	got, _, _ := store.Get(context.Background(), "task-1")
	if got.Status != StatusCompleted || got.Response == nil || got.Response.Output != "done" || got.CompletedAt == nil {
		t.Fatalf("record = %+v, want completed with response and completed_at", got)
	}
}

func TestMemoryStoreFailTransitionsAcceptedToFailed(t *testing.T) {
	store := NewMemoryStore()
	if err := store.CreateAccepted(context.Background(), Record{TaskID: "task-1"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Fail(context.Background(), "task-1", Error{Code: "backend_error", Message: "boom"}); err != nil {
		t.Fatalf("Fail() error = %v", err)
	}
	got, _, _ := store.Get(context.Background(), "task-1")
	if got.Status != StatusFailed || got.Error == nil || got.Error.Code != "backend_error" || got.CompletedAt == nil {
		t.Fatalf("record = %+v, want failed with error and completed_at", got)
	}
}

func TestMemoryStoreRejectsTerminalStateOverwrite(t *testing.T) {
	store := NewMemoryStore()
	if err := store.CreateAccepted(context.Background(), Record{TaskID: "task-completed"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Complete(context.Background(), "task-completed", &backend.Response{Status: backend.StatusCompleted, Output: "first"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Fail(context.Background(), "task-completed", Error{Code: "late", Message: "late"}); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("Fail terminal error = %v, want ErrInvalidTransition", err)
	}
	completed, _, _ := store.Get(context.Background(), "task-completed")
	if completed.Status != StatusCompleted || completed.Response.Output != "first" {
		t.Fatalf("completed record overwritten: %+v", completed)
	}

	if err := store.CreateAccepted(context.Background(), Record{TaskID: "task-failed"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Fail(context.Background(), "task-failed", Error{Code: "first", Message: "first"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Complete(context.Background(), "task-failed", &backend.Response{Status: backend.StatusCompleted}); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("Complete terminal error = %v, want ErrInvalidTransition", err)
	}
	failed, _, _ := store.Get(context.Background(), "task-failed")
	if failed.Status != StatusFailed || failed.Error.Code != "first" {
		t.Fatalf("failed record overwritten: %+v", failed)
	}
}

func TestMemoryStoreUnknownOperations(t *testing.T) {
	store := NewMemoryStore()
	if _, found, err := store.Get(context.Background(), "missing"); err != nil || found {
		t.Fatalf("Get missing = found %v err %v, want false nil", found, err)
	}
	if err := store.Complete(context.Background(), "missing", &backend.Response{}); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("Complete missing error = %v, want ErrTaskNotFound", err)
	}
	if err := store.Fail(context.Background(), "missing", Error{}); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("Fail missing error = %v, want ErrTaskNotFound", err)
	}
	if err := store.Complete(context.Background(), "", &backend.Response{}); !errors.Is(err, ErrInvalidTaskID) {
		t.Fatalf("Complete empty error = %v, want ErrInvalidTaskID", err)
	}
	if err := store.Complete(context.Background(), "task-1", nil); !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("Complete nil response error = %v, want ErrInvalidResponse", err)
	}
}

func TestMemoryStoreReturnsDefensiveCopies(t *testing.T) {
	store := NewMemoryStore()
	resp := &backend.Response{Output: "done", Usage: &backend.Usage{InputTokens: 1}, Metadata: map[string]string{"rk": "rv"}}
	rec := Record{TaskID: "task-1", Request: backend.Request{Metadata: map[string]string{"req": "orig"}}, Response: resp, Error: &Error{Code: "c"}, Metadata: map[string]string{"m": "orig"}}
	if err := store.CreateAccepted(context.Background(), rec); err != nil {
		t.Fatal(err)
	}
	rec.Request.Metadata["req"] = "mutated"
	rec.Metadata["m"] = "mutated"
	if err := store.Complete(context.Background(), "task-1", resp); err != nil {
		t.Fatal(err)
	}
	resp.Output = "mutated"
	resp.Usage.InputTokens = 99
	resp.Metadata["rk"] = "mutated"
	got, _, _ := store.Get(context.Background(), "task-1")
	got.Request.Metadata["req"] = "changed"
	got.Metadata["m"] = "changed"
	got.Response.Output = "changed"
	got.Response.Usage.InputTokens = 42
	got.Response.Metadata["rk"] = "changed"
	again, _, _ := store.Get(context.Background(), "task-1")
	if again.Request.Metadata["req"] != "orig" || again.Metadata["m"] != "orig" || again.Response.Output != "done" || again.Response.Usage.InputTokens != 1 || again.Response.Metadata["rk"] != "rv" {
		t.Fatalf("defensive copy failed: %+v", again)
	}
}

func TestMemoryStoreConcurrentCreateGetComplete(t *testing.T) {
	store := NewMemoryStore()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := fmt.Sprintf("task-%d", i)
			if err := store.CreateAccepted(context.Background(), Record{TaskID: id, CreatedAt: time.Now()}); err != nil {
				t.Errorf("CreateAccepted(%s) error = %v", id, err)
				return
			}
			if _, found, err := store.Get(context.Background(), id); err != nil || !found {
				t.Errorf("Get(%s) found=%v err=%v", id, found, err)
			}
			if err := store.Complete(context.Background(), id, &backend.Response{Status: backend.StatusCompleted}); err != nil {
				t.Errorf("Complete(%s) error = %v", id, err)
			}
		}()
	}
	wg.Wait()
}
