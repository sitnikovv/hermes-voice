package forwarder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxForwardBodyBytes = 1 << 20

// Config describes a lightweight edge forwarder.
type Config struct {
	UpstreamBaseURL string
	EdgeID          string
	EdgeRoom        string
	DefaultDeviceID string
	Timeout         time.Duration
	Client          *http.Client
}

// NewHandler creates an HTTP handler that forwards HA-side requests to a
// central Hermes Voice backend.
func NewHandler(cfg Config) http.Handler {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 20 * time.Second
	}
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: cfg.Timeout}
	}
	h := handler{cfg: cfg}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", h.healthz)
	mux.HandleFunc("/v1/dev/text", h.text)
	mux.HandleFunc("/v1/dev/tasks/", h.task)
	mux.HandleFunc("/v1/dev/tasks", h.task)
	return mux
}

type handler struct {
	cfg Config
}

func (h handler) healthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h handler) text(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	defer r.Body.Close()
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxForwardBodyBytes))
	if err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "invalid_request", "request body too large")
		return
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "malformed JSON")
		return
	}
	h.applyEdgeMetadata(payload)
	forwardBody, err := json.Marshal(payload)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "forwarder error")
		return
	}
	h.forward(w, r.Context(), http.MethodPost, "/v1/dev/text", forwardBody)
}

func (h handler) task(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if r.URL.Path == "/v1/dev/tasks" || r.URL.Path == "/v1/dev/tasks/" || strings.Count(strings.TrimPrefix(r.URL.Path, "/v1/dev/tasks/"), "/") > 0 {
		writeError(w, http.StatusNotFound, "task_not_found", "task not found")
		return
	}
	h.forward(w, r.Context(), http.MethodGet, r.URL.Path, nil)
}

func (h handler) applyEdgeMetadata(payload map[string]any) {
	if h.cfg.DefaultDeviceID != "" {
		if v, ok := payload["device_id"].(string); !ok || strings.TrimSpace(v) == "" {
			payload["device_id"] = h.cfg.DefaultDeviceID
		}
	}
	metadata, _ := payload["metadata"].(map[string]any)
	if metadata == nil {
		metadata = map[string]any{}
	}
	if h.cfg.EdgeID != "" {
		metadata["edge_id"] = h.cfg.EdgeID
	}
	if h.cfg.EdgeRoom != "" {
		metadata["edge_room"] = h.cfg.EdgeRoom
	}
	payload["metadata"] = metadata
}

func (h handler) forward(w http.ResponseWriter, ctx context.Context, method string, path string, body []byte) {
	upstream, err := h.upstreamURL(path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "forwarder_config_error", "forwarder upstream is not configured")
		return
	}
	if h.cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.cfg.Timeout)
		defer cancel()
	}
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, upstream, reader)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "forwarder_request_error", "forwarder request error")
		return
	}
	if body != nil {
		req.Header.Set("content-type", "application/json")
	}
	if h.cfg.EdgeID != "" {
		req.Header.Set("X-Hermes-Edge-Id", h.cfg.EdgeID)
	}
	resp, err := h.cfg.Client.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "upstream_unavailable", "Hermes Voice upstream is unavailable")
		return
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxForwardBodyBytes))
	if err != nil {
		writeError(w, http.StatusBadGateway, "upstream_read_error", "Hermes Voice upstream response error")
		return
	}
	copyContentType(w, resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(respBody)
}

func (h handler) upstreamURL(path string) (string, error) {
	if strings.TrimSpace(h.cfg.UpstreamBaseURL) == "" {
		return "", fmt.Errorf("empty upstream")
	}
	base, err := url.Parse(h.cfg.UpstreamBaseURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return "", fmt.Errorf("invalid upstream")
	}
	base.Path = strings.TrimRight(base.Path, "/") + path
	return base.String(), nil
}

func copyContentType(w http.ResponseWriter, hdr http.Header) {
	if ct := hdr.Get("content-type"); ct != "" {
		w.Header().Set("content-type", ct)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
