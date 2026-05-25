package devclient

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"hermes-voice/internal/backend"
	"hermes-voice/internal/cleanup"
	"hermes-voice/internal/registry"
)

const maxRequestBodyBytes = 1 << 20

// HandlerConfig contains the temporary dev HTTP handler dependencies.
type HandlerConfig struct {
	Registry *registry.Registry
	Cleaner  *cleanup.Cleaner
	Backend  backend.Adapter
}

// NewHandler returns a dev-only HTTP handler for local text requests.
func NewHandler(cfg HandlerConfig) http.Handler {
	mux := http.NewServeMux()
	h := handler{cfg: cfg}
	mux.HandleFunc("/healthz", h.healthz)
	mux.HandleFunc("/v1/dev/text", h.devText)
	return mux
}

type handler struct {
	cfg HandlerConfig
}

func (h handler) healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h handler) devText(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	req, effectiveInput, ok := parseDevTextRequest(w, r)
	if !ok {
		return
	}

	_ = effectiveInput
	writeError(w, http.StatusBadRequest, "route_error", "registry is not configured", req.RequestID)
}

type devTextRequest struct {
	RequestID string            `json:"request_id"`
	DeviceID  string            `json:"device_id"`
	Alias     string            `json:"alias"`
	Input     string            `json:"input"`
	Text      string            `json:"text"`
	Metadata  map[string]string `json:"metadata"`
}

func parseDevTextRequest(w http.ResponseWriter, r *http.Request) (devTextRequest, string, bool) {
	var req devTextRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		message := "malformed JSON"
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			message = "request body too large"
		}
		writeError(w, http.StatusBadRequest, "invalid_request", message, req.RequestID)
		return devTextRequest{}, "", false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid_request", "malformed JSON", req.RequestID)
		return devTextRequest{}, "", false
	}
	if req.DeviceID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "device_id is required", req.RequestID)
		return devTextRequest{}, "", false
	}

	effectiveInput := req.Input
	if effectiveInput == "" {
		effectiveInput = req.Text
	} else if req.Text != "" && req.Text != req.Input {
		writeError(w, http.StatusBadRequest, "invalid_request", "input and text conflict", req.RequestID)
		return devTextRequest{}, "", false
	}
	if strings.TrimSpace(effectiveInput) == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "input is required", req.RequestID)
		return devTextRequest{}, "", false
	}
	return req, effectiveInput, true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

type errorResponse struct {
	RequestID string         `json:"request_id,omitempty"`
	Error     devclientError `json:"error"`
}

type devclientError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeError(w http.ResponseWriter, status int, code, message, requestID string) {
	writeJSON(w, status, errorResponse{
		RequestID: requestID,
		Error: devclientError{
			Code:    code,
			Message: message,
		},
	})
}
