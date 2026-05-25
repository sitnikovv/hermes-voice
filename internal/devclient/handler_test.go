package devclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthzReturnsOK(t *testing.T) {
	handler := NewHandler(HandlerConfig{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got, want := strings.TrimSpace(rec.Body.String()), `{"status":"ok"}`; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
}

func TestDevTextRejectsUnsupportedMethod(t *testing.T) {
	handler := NewHandler(HandlerConfig{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/dev/text", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestDevTextEndpointExists(t *testing.T) {
	handler := NewHandler(HandlerConfig{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/dev/text", strings.NewReader(`{}`))
	handler.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Fatalf("POST /v1/dev/text returned 404")
	}
}

func TestDevTextValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantMsg string
	}{
		{name: "malformed JSON", body: `{`, wantMsg: "malformed JSON"},
		{name: "missing device", body: `{"input":"hello"}`, wantMsg: "device_id is required"},
		{name: "blank input", body: `{"device_id":"phone","input":"   "}`, wantMsg: "input is required"},
		{name: "blank text", body: `{"device_id":"phone","text":"\t"}`, wantMsg: "input is required"},
		{name: "conflicting input text", body: `{"device_id":"phone","input":"one","text":"two"}`, wantMsg: "input and text conflict"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(HandlerConfig{})

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/v1/dev/text", strings.NewReader(tt.body))
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			var got errorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if got.Error.Code != "invalid_request" {
				t.Fatalf("code = %q, want invalid_request", got.Error.Code)
			}
			if got.Error.Message != tt.wantMsg {
				t.Fatalf("message = %q, want %q", got.Error.Message, tt.wantMsg)
			}
		})
	}
}

func TestDevTextUsesTextOnlyWhenInputEmpty(t *testing.T) {
	handler := NewHandler(HandlerConfig{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/dev/text", strings.NewReader(`{"device_id":"phone","input":"","text":"hello"}`))
	handler.ServeHTTP(rec, req)

	if rec.Code == http.StatusBadRequest && strings.Contains(rec.Body.String(), "input is required") {
		t.Fatalf("text should provide effective input when input is empty: body=%s", rec.Body.String())
	}
}

func TestDevTextAllowsEqualInputAndText(t *testing.T) {
	handler := NewHandler(HandlerConfig{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/dev/text", strings.NewReader(`{"device_id":"phone","input":"hello","text":"hello"}`))
	handler.ServeHTTP(rec, req)

	if rec.Code == http.StatusBadRequest && strings.Contains(rec.Body.String(), "conflict") {
		t.Fatalf("equal input/text should not conflict: body=%s", rec.Body.String())
	}
}
