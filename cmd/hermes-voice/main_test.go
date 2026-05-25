package main

import "testing"

func TestServerConfigDefaults(t *testing.T) {
	cfg := defaultServerConfig()
	if cfg.RegistryPath != "testdata/registry.yaml" {
		t.Fatalf("RegistryPath = %q", cfg.RegistryPath)
	}
	if cfg.ListenAddr != "127.0.0.1:8081" {
		t.Fatalf("ListenAddr = %q", cfg.ListenAddr)
	}
	if cfg.StaticOutput != "static dev response" {
		t.Fatalf("StaticOutput = %q", cfg.StaticOutput)
	}
}
