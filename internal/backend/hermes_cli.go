package backend

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const defaultHermesCLIMaxOutput = 64 * 1024

// CommandResult carries bounded subprocess output for Hermes CLI invocation.
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// CommandRunFunc executes a command with an argv vector. It exists so tests can
// exercise command construction without spawning a real Hermes process.
type CommandRunFunc func(ctx context.Context, name string, args ...string) (CommandResult, error)

// HermesCLIConfig configures a CLI-backed Hermes Agent adapter.
type HermesCLIConfig struct {
	Command    string
	Source     string
	MaxTurns   int
	Timeout    time.Duration
	MaxOutput  int
	PassModel  bool
	CommandRun CommandRunFunc
}

type hermesCLIAdapter struct {
	cfg HermesCLIConfig
}

// NewHermesCLIAdapter returns a backend adapter that invokes Hermes Agent via
// `hermes chat` in non-interactive quiet mode.
func NewHermesCLIAdapter(cfg HermesCLIConfig) Adapter {
	if strings.TrimSpace(cfg.Command) == "" {
		cfg.Command = "hermes"
	}
	if strings.TrimSpace(cfg.Source) == "" {
		cfg.Source = "hermes-voice"
	}
	if cfg.MaxTurns <= 0 {
		cfg.MaxTurns = 1
	}
	if cfg.MaxOutput <= 0 {
		cfg.MaxOutput = defaultHermesCLIMaxOutput
	}
	if cfg.CommandRun == nil {
		cfg.CommandRun = runCommand
	}
	return hermesCLIAdapter{cfg: cfg}
}

func (a hermesCLIAdapter) Invoke(ctx context.Context, req Request) (*Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := req.Validate(); err != nil {
		return nil, &Error{Op: "hermes_cli_invoke", BackendID: req.BackendID, Code: "invalid_request", Err: err}
	}
	if a.cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.cfg.Timeout)
		defer cancel()
	}

	args := []string{"chat", "-q", req.Input, "-Q", "--source", a.cfg.Source, "--max-turns", strconv.Itoa(a.cfg.MaxTurns)}
	if a.cfg.PassModel && strings.TrimSpace(req.ModelName) != "" {
		args = append(args, "-m", req.ModelName)
	}
	result, err := a.cfg.CommandRun(ctx, a.cfg.Command, args...)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, &Error{Op: "hermes_cli_invoke", BackendID: req.BackendID, Code: "timeout", Err: ErrTemporary}
		}
		return nil, &Error{Op: "hermes_cli_invoke", BackendID: req.BackendID, Code: "command_error", Err: ErrInvocationFailed}
	}
	if result.ExitCode != 0 {
		return nil, &Error{Op: "hermes_cli_invoke", BackendID: req.BackendID, Code: "non_zero_exit", Err: ErrInvocationFailed}
	}
	if len(result.Stdout) > a.cfg.MaxOutput {
		return nil, &Error{Op: "hermes_cli_invoke", BackendID: req.BackendID, Code: "output_too_large", Err: ErrInvocationFailed}
	}
	answer, sessionID := parseHermesCLIStdout(result.Stdout)
	if strings.TrimSpace(answer) == "" {
		return nil, &Error{Op: "hermes_cli_invoke", BackendID: req.BackendID, Code: "empty_output", Err: ErrInvocationFailed}
	}
	metadata := map[string]string{"backend": "hermes_cli"}
	if sessionID != "" {
		metadata["session_id"] = sessionID
	}
	return &Response{Status: StatusCompleted, Output: answer, Metadata: metadata}, nil
}

func parseHermesCLIStdout(stdout string) (answer string, sessionID string) {
	stdout = strings.ReplaceAll(stdout, "\r\n", "\n")
	lines := strings.Split(stdout, "\n")
	answerLines := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "session_id:") {
			sessionID = strings.TrimSpace(strings.TrimPrefix(trimmed, "session_id:"))
			continue
		}
		answerLines = append(answerLines, line)
	}
	return strings.TrimSpace(strings.Join(answerLines, "\n")), sessionID
}

func runCommand(ctx context.Context, name string, args ...string) (CommandResult, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result := CommandResult{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: 0}
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return result, nil
		}
		return result, fmt.Errorf("run command: %w", err)
	}
	return result, nil
}
