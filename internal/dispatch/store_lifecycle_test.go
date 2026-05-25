package dispatch

import (
	"context"
	"errors"
	"testing"
	"time"

	"hermes-voice/internal/backend"
	"hermes-voice/internal/taskstore"
)

func TestStoreLifecycleQuickTimeoutCreatesAcceptedBeforeReturnAndCompletes(t *testing.T) {
	store := taskstore.NewMemoryStore()
	adapter := blockingAdapter{started: make(chan struct{}), release: make(chan struct{}), resp: &backend.Response{Status: backend.StatusCompleted, Output: "done"}}
	d, err := New(Config{Backend: adapter, Store: store, QuickTimeout: time.Millisecond, TaskID: func(backend.Request) string { return "task-success" }})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	resp, err := d.Invoke(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if resp.Status != backend.StatusAccepted || resp.TaskID != "task-success" {
		t.Fatalf("response = %+v, want accepted task-success", resp)
	}
	rec, found, err := store.Get(context.Background(), "task-success")
	if err != nil || !found || rec.Status != taskstore.StatusAccepted {
		t.Fatalf("accepted record found=%v err=%v rec=%+v", found, err, rec)
	}
	close(adapter.release)
	rec = waitForStoredStatus(t, store, "task-success", taskstore.StatusCompleted)
	if rec.Response == nil || rec.Response.Output != "done" {
		t.Fatalf("completed record = %+v, want response output", rec)
	}
}

func TestStoreLifecycleQuickTimeoutFailures(t *testing.T) {
	tests := []struct {
		name string
		resp *backend.Response
		err  error
	}{
		{name: "backend error", err: errors.New("boom")},
		{name: "nil response"},
		{name: "failed response", resp: &backend.Response{Status: backend.StatusFailed, Output: "bad"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := taskstore.NewMemoryStore()
			adapter := blockingAdapter{started: make(chan struct{}), release: make(chan struct{}), resp: tt.resp, err: tt.err}
			d, err := New(Config{Backend: adapter, Store: store, QuickTimeout: time.Millisecond, TaskID: func(backend.Request) string { return "task-fail" }})
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			resp, err := d.Invoke(context.Background(), validRequest())
			if err != nil || resp.Status != backend.StatusAccepted {
				t.Fatalf("Invoke() resp=%+v err=%v", resp, err)
			}
			close(adapter.release)
			rec := waitForStoredStatus(t, store, "task-fail", taskstore.StatusFailed)
			if rec.Error == nil || rec.Error.Code == "" || rec.Error.Message == "" {
				t.Fatalf("error = %+v, want code/message", rec.Error)
			}
		})
	}
}

func TestStoreLifecycleFastOutcomesAndParentCancelCreateNoTask(t *testing.T) {
	store := taskstore.NewMemoryStore()
	d, err := New(Config{Backend: immediateAdapter{resp: &backend.Response{Status: backend.StatusCompleted}}, Store: store, QuickTimeout: time.Second, TaskID: func(backend.Request) string { return "task-none" }})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Invoke(context.Background(), validRequest()); err != nil {
		t.Fatal(err)
	}
	assertTaskMissing(t, store, "task-none")

	store = taskstore.NewMemoryStore()
	d, err = New(Config{Backend: immediateAdapter{err: errors.New("boom")}, Store: store, QuickTimeout: time.Second, TaskID: func(backend.Request) string { return "task-none" }})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = d.Invoke(context.Background(), validRequest())
	assertTaskMissing(t, store, "task-none")

	store = taskstore.NewMemoryStore()
	ctx, cancel := context.WithCancel(context.Background())
	adapter := blockingAdapter{started: make(chan struct{}), release: make(chan struct{})}
	d, err = New(Config{Backend: adapter, Store: store, QuickTimeout: time.Hour, TaskID: func(backend.Request) string { return "task-none" }})
	if err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() { _, err := d.Invoke(ctx, validRequest()); result <- err }()
	<-adapter.started
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("Invoke() error = %v, want context.Canceled", err)
	}
	assertTaskMissing(t, store, "task-none")
}

func waitForStoredStatus(t *testing.T, store taskstore.Store, taskID string, want taskstore.Status) taskstore.Record {
	t.Helper()
	deadline := time.After(time.Second)
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-deadline:
			rec, found, _ := store.Get(context.Background(), taskID)
			t.Fatalf("timed out waiting for %s; found=%v rec=%+v", want, found, rec)
		case <-tick.C:
			rec, found, err := store.Get(context.Background(), taskID)
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			if found && rec.Status == want {
				return rec
			}
		}
	}
}

func assertTaskMissing(t *testing.T, store taskstore.Store, taskID string) {
	t.Helper()
	_, found, err := store.Get(context.Background(), taskID)
	if err != nil || found {
		t.Fatalf("task %s found=%v err=%v, want missing nil", taskID, found, err)
	}
}
