package devclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"hermes-voice/internal/backend"
)

func TestDevTextMapsBackendErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "invalid request", err: fmt.Errorf("wrapped: %w", backend.ErrInvalidRequest), wantStatus: http.StatusBadRequest, wantCode: "backend_invalid_request"},
		{name: "unauthorized", err: fmt.Errorf("wrapped: %w", backend.ErrUnauthorized), wantStatus: http.StatusUnauthorized, wantCode: "backend_unauthorized"},
		{name: "temporary", err: fmt.Errorf("wrapped: %w", backend.ErrTemporary), wantStatus: http.StatusServiceUnavailable, wantCode: "backend_temporary"},
		{name: "invocation failed", err: fmt.Errorf("wrapped: %w", backend.ErrInvocationFailed), wantStatus: http.StatusBadGateway, wantCode: "backend_invocation_failed"},
		{name: "unexpected", err: errors.New("https://secret.example env:SECRET boom"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backendSpy := &recordingBackend{err: tt.err}
			handler := newTestHandler(backendSpy)

			rec := postDevText(handler, `{"request_id":"req-err","device_id":"phone_ha","input":"hello"}`)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			var got errorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if got.RequestID != "req-err" {
				t.Fatalf("request_id = %q, want req-err", got.RequestID)
			}
			if got.Error.Code != tt.wantCode {
				t.Fatalf("code = %q, want %q", got.Error.Code, tt.wantCode)
			}
			if strings.Contains(got.Error.Message, "secret") || strings.Contains(got.Error.Message, "env:") || strings.Contains(got.Error.Message, "https://") {
				t.Fatalf("error message leaks backend details: %q", got.Error.Message)
			}
		})
	}
}
