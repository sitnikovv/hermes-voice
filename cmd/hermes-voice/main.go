package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"hermes-voice/internal/backend"
	"hermes-voice/internal/cleanup"
	"hermes-voice/internal/devclient"
	"hermes-voice/internal/dispatch"
	"hermes-voice/internal/registry"
)

type serverConfig struct {
	RegistryPath     string
	ListenAddr       string
	StaticOutput     string
	QuickTimeout     time.Duration
	StaticDelay      time.Duration
	AcceptedTaskID   string
	AllowNonLoopback bool
}

func defaultServerConfig() serverConfig {
	return serverConfig{
		RegistryPath: "testdata/registry.yaml",
		ListenAddr:   "127.0.0.1:8081",
		StaticOutput: "static dev response",
	}
}

func parseFlags() serverConfig {
	cfg := defaultServerConfig()
	flag.StringVar(&cfg.RegistryPath, "registry", cfg.RegistryPath, "path to local registry YAML")
	flag.StringVar(&cfg.ListenAddr, "listen", cfg.ListenAddr, "dev HTTP listen address")
	flag.StringVar(&cfg.StaticOutput, "static-output", cfg.StaticOutput, "static backend output for the dev HTTP client")
	flag.DurationVar(&cfg.QuickTimeout, "quick-timeout", cfg.QuickTimeout, "enable dispatcher accepted fallback after this duration; 0 disables dispatcher")
	flag.DurationVar(&cfg.StaticDelay, "static-delay", cfg.StaticDelay, "delay static backend response for dev fallback demonstration")
	flag.StringVar(&cfg.AcceptedTaskID, "accepted-task-id", cfg.AcceptedTaskID, "optional deterministic accepted fallback task ID for dev/testing")
	flag.BoolVar(&cfg.AllowNonLoopback, "allow-non-loopback", cfg.AllowNonLoopback, "allow dev HTTP server to listen on a non-loopback address")
	flag.Parse()
	return cfg
}

func main() {
	cfg := parseFlags()
	if err := run(cfg); err != nil {
		log.Fatal(err)
	}
}

func (cfg serverConfig) validate() error {
	if cfg.AllowNonLoopback {
		return nil
	}
	host, _, err := net.SplitHostPort(cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("invalid listen address %q: %w", cfg.ListenAddr, err)
	}
	ip := net.ParseIP(host)
	if ip == nil {
		addrs, lookupErr := net.LookupIP(host)
		if lookupErr != nil {
			return fmt.Errorf("invalid listen host %q: %w", host, lookupErr)
		}
		for _, addr := range addrs {
			if !addr.IsLoopback() {
				return fmt.Errorf("dev HTTP listen address %q is not loopback; pass --allow-non-loopback to override", cfg.ListenAddr)
			}
		}
		return nil
	}
	if !ip.IsLoopback() {
		return fmt.Errorf("dev HTTP listen address %q is not loopback; pass --allow-non-loopback to override", cfg.ListenAddr)
	}
	return nil
}
func run(cfg serverConfig) error {
	if err := cfg.validate(); err != nil {
		return err
	}
	reg, err := registry.LoadFile(cfg.RegistryPath)
	if err != nil {
		return err
	}
	cleaner, err := cleanup.New(cleanup.DefaultRules())
	if err != nil {
		return err
	}
	adapter, err := buildBackend(cfg)
	if err != nil {
		return err
	}
	handler := devclient.NewHandler(devclient.HandlerConfig{
		Registry: reg,
		Cleaner:  cleaner,
		Backend:  adapter,
	})
	log.Printf("starting dev-only HTTP text client on %s", cfg.ListenAddr)
	return http.ListenAndServe(cfg.ListenAddr, handler)
}

func buildBackend(cfg serverConfig) (backend.Adapter, error) {
	var adapter backend.Adapter = backend.NewStaticAdapter(backend.Response{
		Status: backend.StatusCompleted,
		Output: cfg.StaticOutput,
	})
	if cfg.StaticDelay > 0 {
		adapter = delayedAdapter{backend: adapter, delay: cfg.StaticDelay}
	}
	if cfg.QuickTimeout <= 0 {
		return adapter, nil
	}
	taskID := dispatch.TaskIDFunc(nil)
	if cfg.AcceptedTaskID != "" {
		taskID = func(backend.Request) string { return cfg.AcceptedTaskID }
	}
	return dispatch.New(dispatch.Config{
		Backend:      adapter,
		Runner:       discardRunner{},
		QuickTimeout: cfg.QuickTimeout,
		TaskID:       taskID,
	})
}

type delayedAdapter struct {
	backend backend.Adapter
	delay   time.Duration
}

func (a delayedAdapter) Invoke(ctx context.Context, req backend.Request) (*backend.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	timer := time.NewTimer(a.delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return a.backend.Invoke(ctx, req)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

type discardRunner struct{}

func (discardRunner) Start(context.Context, dispatch.Task) {}
