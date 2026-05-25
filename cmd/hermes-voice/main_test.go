package main

import "testing"

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
