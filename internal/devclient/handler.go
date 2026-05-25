package devclient

import (
	"encoding/json"
	"net/http"

	"hermes-voice/internal/backend"
	"hermes-voice/internal/cleanup"
	"hermes-voice/internal/registry"
)

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
	writeError(w, http.StatusBadRequest, "invalid_request", "device_id is required", "")
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
