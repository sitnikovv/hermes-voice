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
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", "")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h handler) devText(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", "")
		return
	}

	req, effectiveInput, ok := parseDevTextRequest(w, r)
	if !ok {
		return
	}
	if h.cfg.Registry == nil {
		writeError(w, http.StatusBadRequest, "route_error", "registry is not configured", req.RequestID)
		return
	}
	if h.cfg.Cleaner == nil || h.cfg.Backend == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "devclient is not configured", req.RequestID)
		return
	}

	resolved, err := h.cfg.Registry.Resolve(req.DeviceID, req.Alias)
	if err != nil {
		writeError(w, http.StatusBadRequest, "route_error", err.Error(), req.RequestID)
		return
	}

	cleaned := h.cfg.Cleaner.CleanWithTrace(effectiveInput)
	backendReq := backend.Request{
		ID:           req.RequestID,
		Input:        cleaned.Cleaned,
		DeviceID:     req.DeviceID,
		Alias:        req.Alias,
		PersonID:     resolved.PersonID,
		ProfileID:    resolved.ProfileID,
		ModelID:      resolved.ModelID,
		BackendID:    resolved.BackendID,
		ModelName:    resolved.Model.Name,
		SystemPrompt: resolved.Profile.SystemPrompt,
		Metadata:     requestMetadata(req.Metadata),
	}
	backendResp, err := h.cfg.Backend.Invoke(r.Context(), backendReq)
	if err != nil {
		status, code, message := backendErrorStatus(err)
		writeError(w, status, code, message, req.RequestID)
		return
	}
	if backendResp == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "backend returned nil response", req.RequestID)
		return
	}

	writeJSON(w, http.StatusOK, devTextResponse{
		RequestID: req.RequestID,
		Status:    string(backendResp.Status),
		Output:    backendResp.Output,
		TaskID:    backendResp.TaskID,
		Route: routeResponse{
			DeviceID:  resolved.DeviceID,
			Alias:     resolved.Alias,
			PersonID:  resolved.PersonID,
			ProfileID: resolved.ProfileID,
			ModelID:   resolved.ModelID,
			BackendID: resolved.BackendID,
			ModelName: resolved.Model.Name,
		},
		Cleanup:  cleanupResponseFromResult(cleaned),
		Usage:    usageResponseFromBackend(backendResp.Usage),
		Metadata: responseMetadata(backendResp.Metadata),
	})
}

type devTextRequest struct {
	RequestID string            `json:"request_id"`
	DeviceID  string            `json:"device_id"`
	Alias     string            `json:"alias"`
	Input     string            `json:"input"`
	Text      string            `json:"text"`
	Metadata  map[string]string `json:"metadata"`
}

type usageResponse struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type devTextResponse struct {
	RequestID string            `json:"request_id,omitempty"`
	Status    string            `json:"status"`
	Output    string            `json:"output"`
	TaskID    string            `json:"task_id"`
	Route     routeResponse     `json:"route"`
	Cleanup   cleanupResponse   `json:"cleanup"`
	Usage     *usageResponse    `json:"usage"`
	Metadata  map[string]string `json:"metadata"`
}

type routeResponse struct {
	DeviceID  string `json:"device_id"`
	Alias     string `json:"alias"`
	PersonID  string `json:"person_id"`
	ProfileID string `json:"profile_id"`
	ModelID   string `json:"model_id"`
	BackendID string `json:"backend_id"`
	ModelName string `json:"model_name"`
}

type cleanupResponse struct {
	Original string                  `json:"original"`
	Cleaned  string                  `json:"cleaned"`
	Applied  []cleanupAppliedRuleOut `json:"applied"`
}

type cleanupAppliedRuleOut struct {
	ID     string `json:"id"`
	Before string `json:"before"`
	After  string `json:"after"`
}

func parseDevTextRequest(w http.ResponseWriter, r *http.Request) (devTextRequest, string, bool) {
	var req devTextRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		message := "malformed JSON"
		var maxBytesErr *http.MaxBytesError
		status := http.StatusBadRequest
		if errors.As(err, &maxBytesErr) {
			message = "request body too large"
			status = http.StatusRequestEntityTooLarge
		}
		writeError(w, status, "invalid_request", message, req.RequestID)
		return devTextRequest{}, "", false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid_request", "malformed JSON", req.RequestID)
		return devTextRequest{}, "", false
	}
	if strings.TrimSpace(req.DeviceID) == "" {
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

func requestMetadata(metadata map[string]string) map[string]string {
	copied := make(map[string]string, len(metadata)+1)
	for key, value := range metadata {
		copied[key] = value
	}
	if _, ok := copied["source"]; !ok {
		copied["source"] = "dev-http"
	}
	return copied
}

func responseMetadata(metadata map[string]string) map[string]string {
	if metadata == nil {
		return map[string]string{}
	}
	copied := make(map[string]string, len(metadata))
	for key, value := range metadata {
		copied[key] = value
	}
	return copied
}

func backendErrorStatus(err error) (int, string, string) {
	switch {
	case errors.Is(err, backend.ErrInvalidRequest):
		return http.StatusBadRequest, "backend_invalid_request", "backend request is invalid"
	case errors.Is(err, backend.ErrUnauthorized):
		return http.StatusUnauthorized, "backend_unauthorized", "backend unauthorized"
	case errors.Is(err, backend.ErrTemporary):
		return http.StatusServiceUnavailable, "backend_temporary", "backend temporarily unavailable"
	case errors.Is(err, backend.ErrInvocationFailed):
		return http.StatusBadGateway, "backend_invocation_failed", "backend invocation failed"
	default:
		return http.StatusInternalServerError, "internal_error", "internal error"
	}
}

func cleanupResponseFromResult(result cleanup.Result) cleanupResponse {
	applied := make([]cleanupAppliedRuleOut, 0, len(result.Applied))
	for _, rule := range result.Applied {
		applied = append(applied, cleanupAppliedRuleOut{ID: rule.ID, Before: rule.Before, After: rule.After})
	}
	return cleanupResponse{Original: result.Original, Cleaned: result.Cleaned, Applied: applied}
}

func usageResponseFromBackend(usage *backend.Usage) *usageResponse {
	if usage == nil {
		return nil
	}
	return &usageResponse{InputTokens: usage.InputTokens, OutputTokens: usage.OutputTokens}
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
