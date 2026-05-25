package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"hermes-voice/internal/backend"
	"hermes-voice/internal/cleanup"
	"hermes-voice/internal/devclient"
	"hermes-voice/internal/registry"
	"hermes-voice/internal/taskstore"
)

func TestServerConfigRejectsNonLoopbackListenByDefault(t *testing.T) {
	cfg := defaultServerConfig()
	cfg.ListenAddr = "0.0.0.0:8081"

	if err := cfg.validate(); err == nil {
		t.Fatal("validate() error = nil, want non-loopback rejection")
	}
}

func TestServerConfigAllowsNonLoopbackWhenExplicitlyEnabled(t *testing.T) {
	cfg := defaultServerConfig()
	cfg.ListenAddr = "0.0.0.0:8081"
	cfg.AllowNonLoopback = true

	if err := cfg.validate(); err != nil {
		t.Fatalf("validate() error = %v, want nil", err)
	}
}

func TestBuildBackendDefaultKeepsCompletedStaticResponse(t *testing.T) {
	cfg := defaultServerConfig()
	adapter, err := buildBackend(cfg, nil)
	if err != nil {
		t.Fatalf("buildBackend() error = %v", err)
	}
	resp, err := adapter.Invoke(nil, validBackendRequest())
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if resp.Status != backend.StatusCompleted || resp.Output != cfg.StaticOutput {
		t.Fatalf("response = %+v, want completed static output", resp)
	}
}

func TestBuildBackendCanEnableQuickTimeoutDispatcher(t *testing.T) {
	cfg := defaultServerConfig()
	cfg.QuickTimeout = time.Millisecond
	adapter, err := buildBackend(cfg, taskstore.NewMemoryStore())
	if err != nil {
		t.Fatalf("buildBackend() error = %v", err)
	}
	if adapter == nil {
		t.Fatal("adapter = nil, want configured dispatcher")
	}
}

func TestDevHandlerStaticDelayLongerThanQuickTimeoutStoresAcceptedTask(t *testing.T) {
	cfg := defaultServerConfig()
	cfg.QuickTimeout = time.Millisecond
	cfg.StaticDelay = 50 * time.Millisecond
	cfg.AcceptedTaskID = "task-demo"
	store := taskstore.NewMemoryStore()
	adapter, err := buildBackend(cfg, store)
	if err != nil {
		t.Fatalf("buildBackend() error = %v", err)
	}
	reg, err := registry.LoadFile("../../testdata/registry.yaml")
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	cleaner, err := cleanup.New(cleanup.DefaultRules())
	if err != nil {
		t.Fatalf("cleanup.New() error = %v", err)
	}
	handler := devclient.NewHandler(devclient.HandlerConfig{Registry: reg, Cleaner: cleaner, Backend: adapter, TaskStore: store})
	req := httptest.NewRequest(http.MethodPost, "/v1/dev/text", strings.NewReader(`{"request_id":"dev-accepted","device_id":"phone_ha","alias":"coding","input":"hello"}`))
	req.Header.Set("content-type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	var payload struct {
		Status string `json:"status"`
		TaskID string `json:"task_id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.Status != string(backend.StatusAccepted) || payload.TaskID != "task-demo" {
		t.Fatalf("payload = %+v, want accepted task-demo", payload)
	}
	taskRec := httptest.NewRecorder()
	taskReq := httptest.NewRequest(http.MethodGet, "/v1/dev/tasks/task-demo", nil)
	handler.ServeHTTP(taskRec, taskReq)
	if taskRec.Code != http.StatusOK {
		t.Fatalf("task status = %d body = %s", taskRec.Code, taskRec.Body.String())
	}
	var taskPayload struct {
		TaskID string `json:"task_id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(taskRec.Body.Bytes(), &taskPayload); err != nil {
		t.Fatalf("task json.Unmarshal() error = %v", err)
	}
	if taskPayload.TaskID != "task-demo" || taskPayload.Status != string(taskstore.StatusAccepted) {
		t.Fatalf("task payload = %+v, want accepted task-demo", taskPayload)
	}
}

func TestBackupRegistryCreatesBackupAndReportsPath(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "registry.yaml")
	if err := os.WriteFile(source, []byte("registry-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := defaultServerConfig()
	cfg.RegistryPath = source
	cfg.BackupRegistry = true
	cfg.RegistryBackupDir = filepath.Join(dir, "backups")

	backupPath, err := backupRegistry(cfg)
	if err != nil {
		t.Fatalf("backupRegistry() error = %v", err)
	}
	got, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "registry-bytes" {
		t.Fatalf("backup content = %q", got)
	}
}

func TestBackupRegistryModeDoesNotRequireListenValidation(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "registry.yaml")
	if err := os.WriteFile(source, []byte("registry-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := defaultServerConfig()
	cfg.RegistryPath = source
	cfg.BackupRegistry = true
	cfg.ListenAddr = "not-a-host-port"

	if _, err := backupRegistry(cfg); err != nil {
		t.Fatalf("backupRegistry() error = %v, want listen-independent backup", err)
	}
}

func validBackendRequest() backend.Request {
	return backend.Request{
		ID:        "req-1",
		Input:     "hello",
		PersonID:  "person",
		ProfileID: "profile",
		ModelID:   "model",
		BackendID: "backend",
		ModelName: "test-model",
	}
}
