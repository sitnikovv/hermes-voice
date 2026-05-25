package main

import (
	"flag"
	"log"
	"net/http"

	"hermes-voice/internal/backend"
	"hermes-voice/internal/cleanup"
	"hermes-voice/internal/devclient"
	"hermes-voice/internal/registry"
)

type serverConfig struct {
	RegistryPath string
	ListenAddr   string
	StaticOutput string
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
	flag.Parse()
	return cfg
}

func main() {
	cfg := parseFlags()
	if err := run(cfg); err != nil {
		log.Fatal(err)
	}
}

func run(cfg serverConfig) error {
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
