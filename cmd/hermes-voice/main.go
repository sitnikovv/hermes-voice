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
	"hermes-voice/internal/forwarder"
	"hermes-voice/internal/registry"
	"hermes-voice/internal/taskstore"
)

type serverConfig struct {
	Mode              string
	RegistryPath      string
	RegistryBackupDir string
	BackupRegistry    bool
	ListenAddr        string
	BackendMode       string
	StaticOutput      string
	QuickTimeout      time.Duration
	StaticDelay       time.Duration
	AcceptedTaskID    string
	AllowNonLoopback  bool
	HermesCommand     string
	HermesSource      string
	HermesMaxTurns    int
	HermesTimeout     time.Duration
	HermesMaxOutput   int
	HermesPassModel   bool
	HermesCommandRun  backend.CommandRunFunc
	ForwarderUpstream string
	ForwarderEdgeID   string
	ForwarderEdgeRoom string
	ForwarderDeviceID string
	ForwarderTimeout  time.Duration
}

func defaultServerConfig() serverConfig {
	return serverConfig{
		Mode:             "serve",
		RegistryPath:     "testdata/registry.yaml",
		ListenAddr:       "127.0.0.1:8081",
		BackendMode:      "static",
		StaticOutput:     "static dev response",
		HermesCommand:    "hermes",
		HermesSource:     "hermes-voice",
		HermesMaxTurns:   1,
		HermesMaxOutput:  64 * 1024,
		ForwarderTimeout: 20 * time.Second,
	}
}

func parseFlags() serverConfig {
	cfg := defaultServerConfig()
	flag.StringVar(&cfg.Mode, "mode", cfg.Mode, "runtime mode: serve or forwarder")
	flag.StringVar(&cfg.RegistryPath, "registry", cfg.RegistryPath, "path to local registry YAML")
	flag.StringVar(&cfg.RegistryBackupDir, "registry-backup-dir", cfg.RegistryBackupDir, "optional registry backup directory; default is <registry-dir>/.registry-backups")
	flag.BoolVar(&cfg.BackupRegistry, "backup-registry", cfg.BackupRegistry, "create a registry backup and exit without starting the dev HTTP server")
	flag.StringVar(&cfg.ListenAddr, "listen", cfg.ListenAddr, "dev HTTP listen address")
	flag.StringVar(&cfg.BackendMode, "backend", cfg.BackendMode, "backend adapter mode: static or hermes-cli")
	flag.StringVar(&cfg.StaticOutput, "static-output", cfg.StaticOutput, "static backend output for the dev HTTP client")
	flag.DurationVar(&cfg.QuickTimeout, "quick-timeout", cfg.QuickTimeout, "enable dispatcher accepted fallback after this duration; 0 disables dispatcher")
	flag.DurationVar(&cfg.StaticDelay, "static-delay", cfg.StaticDelay, "delay static backend response for dev fallback demonstration")
	flag.StringVar(&cfg.AcceptedTaskID, "accepted-task-id", cfg.AcceptedTaskID, "optional deterministic accepted fallback task ID for dev/testing")
	flag.BoolVar(&cfg.AllowNonLoopback, "allow-non-loopback", cfg.AllowNonLoopback, "allow dev HTTP server to listen on a non-loopback address")
	flag.StringVar(&cfg.HermesCommand, "hermes-command", cfg.HermesCommand, "Hermes CLI command path for --backend hermes-cli")
	flag.StringVar(&cfg.HermesSource, "hermes-source", cfg.HermesSource, "Hermes session source tag for --backend hermes-cli")
	flag.IntVar(&cfg.HermesMaxTurns, "hermes-max-turns", cfg.HermesMaxTurns, "Hermes CLI max tool-calling turns for --backend hermes-cli")
	flag.DurationVar(&cfg.HermesTimeout, "hermes-timeout", cfg.HermesTimeout, "Hermes CLI subprocess timeout for --backend hermes-cli; 0 disables adapter timeout")
	flag.IntVar(&cfg.HermesMaxOutput, "hermes-max-output", cfg.HermesMaxOutput, "maximum stdout bytes accepted from Hermes CLI")
	flag.BoolVar(&cfg.HermesPassModel, "hermes-pass-model", cfg.HermesPassModel, "pass resolved registry model name to Hermes CLI with -m")
	flag.StringVar(&cfg.ForwarderUpstream, "forwarder-upstream", cfg.ForwarderUpstream, "upstream Hermes Voice base URL for --mode forwarder")
	flag.StringVar(&cfg.ForwarderEdgeID, "forwarder-edge-id", cfg.ForwarderEdgeID, "edge identifier added by --mode forwarder")
	flag.StringVar(&cfg.ForwarderEdgeRoom, "forwarder-edge-room", cfg.ForwarderEdgeRoom, "optional room metadata added by --mode forwarder")
	flag.StringVar(&cfg.ForwarderDeviceID, "forwarder-device-id", cfg.ForwarderDeviceID, "default device_id used by --mode forwarder when request omits it")
	flag.DurationVar(&cfg.ForwarderTimeout, "forwarder-timeout", cfg.ForwarderTimeout, "upstream timeout for --mode forwarder")
	flag.Parse()
	return cfg
}

func main() {
	cfg := parseFlags()
	if cfg.BackupRegistry {
		backupPath, err := backupRegistry(cfg)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("registry backup created: %s\n", backupPath)
		return
	}
	if err := run(cfg); err != nil {
		log.Fatal(err)
	}
}

func backupRegistry(cfg serverConfig) (string, error) {
	return registry.BackupFile(cfg.RegistryPath, cfg.RegistryBackupDir, time.Now())
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
	store := taskstore.NewMemoryStore()
	handler, err := buildHTTPHandler(cfg, store)
	if err != nil {
		return err
	}
	log.Printf("starting %s HTTP server on %s", cfg.Mode, cfg.ListenAddr)
	return http.ListenAndServe(cfg.ListenAddr, handler)
}

func buildHTTPHandler(cfg serverConfig, store taskstore.Store) (http.Handler, error) {
	switch cfg.Mode {
	case "", "serve":
		reg, err := registry.LoadFile(cfg.RegistryPath)
		if err != nil {
			return nil, err
		}
		cleaner, err := cleanup.New(cleanup.DefaultRules())
		if err != nil {
			return nil, err
		}
		adapter, err := buildBackend(cfg, store)
		if err != nil {
			return nil, err
		}
		return devclient.NewHandler(devclient.HandlerConfig{
			Registry:  reg,
			Cleaner:   cleaner,
			Backend:   adapter,
			TaskStore: store,
		}), nil
	case "forwarder":
		return forwarder.NewHandler(forwarder.Config{
			UpstreamBaseURL: cfg.ForwarderUpstream,
			EdgeID:          cfg.ForwarderEdgeID,
			EdgeRoom:        cfg.ForwarderEdgeRoom,
			DefaultDeviceID: cfg.ForwarderDeviceID,
			Timeout:         cfg.ForwarderTimeout,
		}), nil
	default:
		return nil, fmt.Errorf("unsupported mode: %s", cfg.Mode)
	}
}

func buildBackend(cfg serverConfig, store taskstore.Store) (backend.Adapter, error) {
	var adapter backend.Adapter
	switch cfg.BackendMode {
	case "", "static":
		adapter = backend.NewStaticAdapter(backend.Response{
			Status: backend.StatusCompleted,
			Output: cfg.StaticOutput,
		})
		if cfg.StaticDelay > 0 {
			adapter = delayedAdapter{backend: adapter, delay: cfg.StaticDelay}
		}
	case "hermes-cli":
		adapter = backend.NewHermesCLIAdapter(backend.HermesCLIConfig{
			Command:    cfg.HermesCommand,
			Source:     cfg.HermesSource,
			MaxTurns:   cfg.HermesMaxTurns,
			Timeout:    cfg.HermesTimeout,
			MaxOutput:  cfg.HermesMaxOutput,
			PassModel:  cfg.HermesPassModel,
			CommandRun: cfg.HermesCommandRun,
		})
	default:
		return nil, fmt.Errorf("%w: %s", backend.ErrUnsupportedBackend, cfg.BackendMode)
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
		Store:        store,
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
