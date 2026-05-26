package backend

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestHermesCLIAdapterInvokesHermesChatAndParsesAnswer(t *testing.T) {
	var gotName string
	var gotArgs []string
	adapter := NewHermesCLIAdapter(HermesCLIConfig{
		Command:   "/usr/local/bin/hermes",
		Source:    "hermes-voice-test",
		MaxTurns:  1,
		Timeout:   time.Second,
		MaxOutput: 1024,
		PassModel: true,
		CommandRun: func(ctx context.Context, name string, args ...string) (CommandResult, error) {
			gotName = name
			gotArgs = append([]string(nil), args...)
			return CommandResult{Stdout: "session_id: 20260526_165141_9eb642\nреальный ответ\n", ExitCode: 0}, nil
		},
	})

	resp, err := adapter.Invoke(context.Background(), validCLIRequest())
	if err != nil {
		t.Fatalf("Invoke() error = %v, want nil", err)
	}
	if resp.Status != StatusCompleted || resp.Output != "реальный ответ" {
		t.Fatalf("response = %+v, want completed real answer", resp)
	}
	if resp.Metadata["backend"] != "hermes_cli" || resp.Metadata["session_id"] != "20260526_165141_9eb642" {
		t.Fatalf("metadata = %#v, want backend and parsed session_id", resp.Metadata)
	}
	if gotName != "/usr/local/bin/hermes" {
		t.Fatalf("command name = %q", gotName)
	}
	wantArgs := []string{"chat", "-q", "hello Hermes", "-Q", "--source", "hermes-voice-test", "--max-turns", "1", "-m", "test-model"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
	}
}

func TestHermesCLIAdapterRejectsOversizedOutput(t *testing.T) {
	adapter := NewHermesCLIAdapter(HermesCLIConfig{
		Command:   "hermes",
		MaxOutput: 8,
		CommandRun: func(ctx context.Context, name string, args ...string) (CommandResult, error) {
			return CommandResult{Stdout: strings.Repeat("x", 64), ExitCode: 0}, nil
		},
	})

	_, err := adapter.Invoke(context.Background(), validCLIRequest())
	if !errors.Is(err, ErrInvocationFailed) {
		t.Fatalf("Invoke() error = %v, want ErrInvocationFailed", err)
	}
}

func TestHermesCLIAdapterMapsTimeoutAsTemporary(t *testing.T) {
	adapter := NewHermesCLIAdapter(HermesCLIConfig{
		Command: "hermes",
		Timeout: time.Millisecond,
		CommandRun: func(ctx context.Context, name string, args ...string) (CommandResult, error) {
			<-ctx.Done()
			return CommandResult{}, ctx.Err()
		},
	})

	_, err := adapter.Invoke(context.Background(), validCLIRequest())
	if !errors.Is(err, ErrTemporary) {
		t.Fatalf("Invoke() error = %v, want ErrTemporary", err)
	}
}

func TestHermesCLIAdapterMapsCommandFailure(t *testing.T) {
	adapter := NewHermesCLIAdapter(HermesCLIConfig{
		Command: "hermes",
		CommandRun: func(ctx context.Context, name string, args ...string) (CommandResult, error) {
			return CommandResult{Stderr: "auth token [REDACTED] invalid", ExitCode: 2}, nil
		},
	})

	_, err := adapter.Invoke(context.Background(), validCLIRequest())
	if !errors.Is(err, ErrInvocationFailed) {
		t.Fatalf("Invoke() error = %v, want ErrInvocationFailed", err)
	}
	if strings.Contains(err.Error(), "token") || strings.Contains(err.Error(), "[REDACTED]") {
		t.Fatalf("error leaks stderr details: %v", err)
	}
}

func validCLIRequest() Request {
	return Request{
		ID:        "req-cli",
		Input:     "hello Hermes",
		PersonID:  "owner",
		ProfileID: "default",
		ModelID:   "qwen",
		BackendID: "hermes",
		ModelName: "test-model",
	}
}
