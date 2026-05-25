package devclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hermes-voice/internal/backend"
	"hermes-voice/internal/taskstore"
)

func TestDevTaskEndpointReturnsAcceptedCompletedAndFailed(t *testing.T) {
	store := taskstore.NewMemoryStore()
	if err := store.CreateAccepted(context.Background(), taskstore.Record{TaskID: "task-accepted", RequestID: "req-a", Request: backend.Request{SystemPrompt: "secret"}, Metadata: map[string]string{"m": "v"}}); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateAccepted(context.Background(), taskstore.Record{TaskID: "task-completed", RequestID: "req-c"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Complete(context.Background(), "task-completed", &backend.Response{Status: backend.StatusCompleted, Output: "done", Usage: &backend.Usage{InputTokens: 1, OutputTokens: 2}, Metadata: map[string]string{"rk": "rv"}}); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateAccepted(context.Background(), taskstore.Record{TaskID: "task-failed", RequestID: "req-f"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Fail(context.Background(), "task-failed", taskstore.Error{Code: "backend_error", Message: "boom"}); err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(HandlerConfig{TaskStore: store})

	accepted := getTaskJSON(t, handler, "task-accepted", http.StatusOK)
	if accepted["task_id"] != "task-accepted" || accepted["request_id"] != "req-a" || accepted["status"] != "accepted" || accepted["response"] != nil || accepted["error"] != nil {
		t.Fatalf("accepted payload = %#v", accepted)
	}
	body, _ := json.Marshal(accepted)
	if strings.Contains(string(body), "SystemPrompt") || strings.Contains(string(body), "secret") {
		t.Fatalf("task response exposed request/system prompt: %s", body)
	}

	completed := getTaskJSON(t, handler, "task-completed", http.StatusOK)
	resp, ok := completed["response"].(map[string]any)
	if !ok || resp["status"] != "completed" || resp["output"] != "done" {
		t.Fatalf("completed response = %#v", completed["response"])
	}
	if usage, ok := resp["usage"].(map[string]any); !ok || usage["input_tokens"].(float64) != 1 || usage["output_tokens"].(float64) != 2 {
		t.Fatalf("usage = %#v", resp["usage"])
	}
	if metadata, ok := resp["metadata"].(map[string]any); !ok || metadata["rk"] != "rv" {
		t.Fatalf("metadata = %#v", resp["metadata"])
	}

	failed := getTaskJSON(t, handler, "task-failed", http.StatusOK)
	failure, ok := failed["error"].(map[string]any)
	if !ok || failure["code"] != "backend_error" || failure["message"] != "boom" {
		t.Fatalf("failed error = %#v", failed["error"])
	}
}

func TestDevTaskEndpointUnknownMethodAndMalformedPaths(t *testing.T) {
	handler := NewHandler(HandlerConfig{TaskStore: taskstore.NewMemoryStore()})
	unknown := getTaskJSON(t, handler, "missing", http.StatusNotFound)
	if errObj := unknown["error"].(map[string]any); errObj["code"] != "task_not_found" {
		t.Fatalf("unknown payload = %#v", unknown)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/dev/tasks/task-1", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("POST status=%d Allow=%q", rec.Code, rec.Header().Get("Allow"))
	}

	for _, path := range []string{"/v1/dev/tasks", "/v1/dev/tasks/", "/v1/dev/tasks/task-1/extra"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s status=%d body=%s, want 404", path, rec.Code, rec.Body.String())
		}
	}
}

func getTaskJSON(t *testing.T, handler http.Handler, taskID string, wantStatus int) map[string]any {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/dev/tasks/"+taskID, nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("GET task %s status=%d body=%s, want %d", taskID, rec.Code, rec.Body.String(), wantStatus)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	return payload
}
