package forwarder

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHealthzReturnsOK(t *testing.T) {
	h := NewHandler(Config{UpstreamBaseURL: "http://upstream.local", EdgeID: "edge-ha"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if strings.TrimSpace(rec.Body.String()) != `{"status":"ok"}` {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestForwarderAddsEdgeMetadataAndForwardsText(t *testing.T) {
	var gotPath string
	var gotEdgeHeader string
	var gotPayload map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotEdgeHeader = r.Header.Get("X-Hermes-Edge-Id")
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("decode upstream payload: %v", err)
		}
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"completed","output":"ok"}`))
	}))
	defer upstream.Close()

	h := NewHandler(Config{UpstreamBaseURL: upstream.URL, EdgeID: "edge-ha", EdgeRoom: "cabinet", DefaultDeviceID: "edge-phone", Timeout: time.Second})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/dev/text", strings.NewReader(`{"request_id":"r1","alias":"coding","input":"hello","metadata":{"source":"ha"}}`))
	req.Header.Set("content-type", "application/json")

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if gotPath != "/v1/dev/text" {
		t.Fatalf("upstream path = %q", gotPath)
	}
	if gotEdgeHeader != "edge-ha" {
		t.Fatalf("edge header = %q", gotEdgeHeader)
	}
	if gotPayload["device_id"] != "edge-phone" {
		t.Fatalf("device_id = %#v, want default edge-phone", gotPayload["device_id"])
	}
	metadata, ok := gotPayload["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("metadata = %#v, want object", gotPayload["metadata"])
	}
	if metadata["source"] != "ha" || metadata["edge_id"] != "edge-ha" || metadata["edge_room"] != "cabinet" {
		t.Fatalf("metadata = %#v", metadata)
	}
	if strings.TrimSpace(rec.Body.String()) != `{"status":"completed","output":"ok"}` {
		t.Fatalf("response body = %q", rec.Body.String())
	}
}

func TestForwarderPreservesExistingDeviceID(t *testing.T) {
	var gotPayload map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("decode upstream payload: %v", err)
		}
		_, _ = w.Write([]byte(`{"status":"completed"}`))
	}))
	defer upstream.Close()

	h := NewHandler(Config{UpstreamBaseURL: upstream.URL, EdgeID: "edge-ha", DefaultDeviceID: "edge-default"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/dev/text", strings.NewReader(`{"request_id":"r1","device_id":"phone_ha","input":"hello"}`))
	req.Header.Set("content-type", "application/json")

	h.ServeHTTP(rec, req)

	if gotPayload["device_id"] != "phone_ha" {
		t.Fatalf("device_id = %#v, want original phone_ha", gotPayload["device_id"])
	}
}

func TestForwarderForwardsTaskStatus(t *testing.T) {
	var gotPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"task_id":"task-1","status":"completed"}`))
	}))
	defer upstream.Close()

	h := NewHandler(Config{UpstreamBaseURL: upstream.URL, EdgeID: "edge-ha"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/dev/tasks/task-1", nil)

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if gotPath != "/v1/dev/tasks/task-1" {
		t.Fatalf("upstream path = %q", gotPath)
	}
	if strings.TrimSpace(rec.Body.String()) != `{"task_id":"task-1","status":"completed"}` {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestForwarderMapsUpstreamUnavailableToSafeError(t *testing.T) {
	h := NewHandler(Config{UpstreamBaseURL: "http://127.0.0.1:1", EdgeID: "edge-ha", Timeout: time.Millisecond})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/dev/text", strings.NewReader(`{"request_id":"r1","input":"hello"}`))
	req.Header.Set("content-type", "application/json")

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway && rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var got struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if got.Error.Code == "" || strings.Contains(got.Error.Message, "127.0.0.1") {
		t.Fatalf("unsafe error response: %+v", got.Error)
	}
}
