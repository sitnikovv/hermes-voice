package devclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hermes-voice/internal/backend"
	"hermes-voice/internal/cleanup"
	"hermes-voice/internal/registry"
)

type recordingBackend struct {
	calls int
	req   backend.Request
	resp  backend.Response
	err   error
}

func (b *recordingBackend) Invoke(ctx context.Context, req backend.Request) (*backend.Response, error) {
	b.calls++
	b.req = req
	if b.err != nil {
		return nil, b.err
	}
	resp := b.resp
	return &resp, nil
}

func TestDevTextPipelineHappyPath(t *testing.T) {
	backendSpy := &recordingBackend{resp: backend.Response{
		Status:   backend.StatusCompleted,
		Output:   "static dev response",
		TaskID:   "task-1",
		Usage:    &backend.Usage{InputTokens: 2, OutputTokens: 3},
		Metadata: map[string]string{"backend": "static"},
	}}
	handler := newTestHandler(backendSpy)

	body := `{"request_id":"req-1","device_id":"phone_ha","alias":"coding","input":"  гермес   помоги  ","metadata":{"source":"curl","x":"y"}}`
	rec := postDevText(handler, body)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if backendSpy.calls != 1 {
		t.Fatalf("backend calls = %d, want 1", backendSpy.calls)
	}
	wantReq := backend.Request{
		ID:           "req-1",
		Input:        "помоги",
		DeviceID:     "phone_ha",
		Alias:        "coding",
		PersonID:     "sve",
		ProfileID:    "coding",
		ModelID:      "default_chat",
		BackendID:    "local_hermes",
		ModelName:    "hermes-test-model",
		SystemPrompt: "coding prompt",
	}
	if backendSpy.req.ID != wantReq.ID || backendSpy.req.Input != wantReq.Input || backendSpy.req.DeviceID != wantReq.DeviceID || backendSpy.req.Alias != wantReq.Alias || backendSpy.req.PersonID != wantReq.PersonID || backendSpy.req.ProfileID != wantReq.ProfileID || backendSpy.req.ModelID != wantReq.ModelID || backendSpy.req.BackendID != wantReq.BackendID || backendSpy.req.ModelName != wantReq.ModelName || backendSpy.req.SystemPrompt != wantReq.SystemPrompt {
		t.Fatalf("backend request = %+v, want matching %+v", backendSpy.req, wantReq)
	}
	if backendSpy.req.Metadata["source"] != "curl" || backendSpy.req.Metadata["x"] != "y" {
		t.Fatalf("metadata = %#v, want copied request metadata", backendSpy.req.Metadata)
	}

	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got["request_id"] != "req-1" || got["status"] != "completed" || got["output"] != "static dev response" || got["task_id"] != "task-1" {
		t.Fatalf("unexpected top-level response: %#v", got)
	}
	route := got["route"].(map[string]any)
	for key, want := range map[string]string{
		"device_id":  "phone_ha",
		"alias":      "coding",
		"person_id":  "sve",
		"profile_id": "coding",
		"model_id":   "default_chat",
		"backend_id": "local_hermes",
		"model_name": "hermes-test-model",
	} {
		if route[key] != want {
			t.Fatalf("route[%s] = %#v, want %q", key, route[key], want)
		}
	}
	cleanupTrace := got["cleanup"].(map[string]any)
	if cleanupTrace["original"] != "  гермес   помоги  " || cleanupTrace["cleaned"] != "помоги" {
		t.Fatalf("cleanup = %#v", cleanupTrace)
	}
	if applied := cleanupTrace["applied"].([]any); len(applied) == 0 {
		t.Fatalf("cleanup applied trace is empty")
	}
	responseBody := rec.Body.String()
	if strings.Contains(responseBody, "https://secret.example") || strings.Contains(responseBody, "api_key_ref") || strings.Contains(responseBody, "env:SECRET") {
		t.Fatalf("response exposes backend secret details: %s", responseBody)
	}
}

func TestDevTextAddsDefaultSourceMetadataWhenAbsent(t *testing.T) {
	backendSpy := &recordingBackend{resp: backend.Response{Status: backend.StatusCompleted}}
	handler := newTestHandler(backendSpy)

	rec := postDevText(handler, `{"device_id":"phone_ha","input":"hello","metadata":{"x":"y"}}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if backendSpy.req.Metadata["source"] != "dev-http" || backendSpy.req.Metadata["x"] != "y" {
		t.Fatalf("metadata = %#v, want source=dev-http plus copied metadata", backendSpy.req.Metadata)
	}
}

func TestDevTextTextAliasPassesOriginalTextToCleanup(t *testing.T) {
	backendSpy := &recordingBackend{resp: backend.Response{Status: backend.StatusCompleted}}
	handler := newTestHandler(backendSpy)

	rec := postDevText(handler, `{"device_id":"phone_ha","text":"  гермес hello  "}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if backendSpy.req.Input != "hello" {
		t.Fatalf("backend input = %q, want cleaned text alias", backendSpy.req.Input)
	}
	if !strings.Contains(rec.Body.String(), `"original":"  гермес hello  "`) {
		t.Fatalf("cleanup original did not preserve text alias input: %s", rec.Body.String())
	}
}

func TestDevTextRouteErrorsSkipBackend(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "unknown device", body: `{"device_id":"missing","input":"hello"}`},
		{name: "unknown alias", body: `{"device_id":"phone_ha","alias":"missing","input":"hello"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backendSpy := &recordingBackend{resp: backend.Response{Status: backend.StatusCompleted}}
			handler := newTestHandler(backendSpy)

			rec := postDevText(handler, tt.body)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
			}
			if backendSpy.calls != 0 {
				t.Fatalf("backend calls = %d, want 0", backendSpy.calls)
			}
			var got errorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode error: %v", err)
			}
			if got.Error.Code != "route_error" {
				t.Fatalf("code = %q, want route_error", got.Error.Code)
			}
		})
	}
}

func TestDevTextPreservesRawDeviceAndAliasForResolve(t *testing.T) {
	backendSpy := &recordingBackend{resp: backend.Response{Status: backend.StatusCompleted}}
	handler := newTestHandler(backendSpy)

	rec := postDevText(handler, `{"device_id":" Phone_HA ","alias":" Coding ","input":"hello"}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 because raw untrimmed IDs must not resolve; body=%s", rec.Code, rec.Body.String())
	}
	if backendSpy.calls != 0 {
		t.Fatalf("backend calls = %d, want 0", backendSpy.calls)
	}
}

func newTestHandler(adapter backend.Adapter) http.Handler {
	cleaner, err := cleanup.New(cleanup.DefaultRules())
	if err != nil {
		panic(err)
	}
	return NewHandler(HandlerConfig{
		Registry: testRegistry(),
		Cleaner:  cleaner,
		Backend:  adapter,
	})
}

func postDevText(handler http.Handler, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/dev/text", strings.NewReader(body))
	handler.ServeHTTP(rec, req)
	return rec
}

func testRegistry() *registry.Registry {
	return &registry.Registry{
		Backends: map[string]registry.Backend{
			"local_hermes": {Type: "openai_compatible", Endpoint: "https://secret.example", APIKeyRef: "env:SECRET"},
		},
		Models: map[string]registry.Model{
			"default_chat": {Backend: "local_hermes", Name: "hermes-test-model"},
		},
		Persons: map[string]registry.Person{
			"sve": {DisplayName: "SVE"},
		},
		Profiles: map[string]registry.Profile{
			"default": {Person: "sve", Model: "default_chat", SystemPrompt: "default prompt"},
			"coding":  {Person: "sve", Model: "default_chat", SystemPrompt: "coding prompt"},
		},
		Devices: map[string]registry.Device{
			"phone_ha": {
				DefaultPerson:  "sve",
				DefaultProfile: "default",
				Aliases: map[string]registry.AliasBinding{
					"coding": {Profile: "coding"},
				},
			},
		},
	}
}
