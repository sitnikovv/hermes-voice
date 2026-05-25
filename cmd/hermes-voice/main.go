package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"

	"hermes-voice/internal/backend"
	"hermes-voice/internal/cleanup"
	"hermes-voice/internal/devclient"
	"hermes-voice/internal/registry"
)

type serverConfig struct {
	RegistryPath     string
	ListenAddr       string
	StaticOutput     string
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
	adapter := backend.NewStaticAdapter(backend.Response{
		Status: backend.StatusCompleted,
		Output: cfg.StaticOutput,
	})
	handler := devclient.NewHandler(devclient.HandlerConfig{
		Registry: reg,
		Cleaner:  cleaner,
		Backend:  adapter,
	})
	log.Printf("starting dev-only HTTP text client on %s", cfg.ListenAddr)
	return http.ListenAndServe(cfg.ListenAddr, handler)
}
