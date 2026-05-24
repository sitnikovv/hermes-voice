package registry_test

import (
	"os"
	"testing"

	"hermes-voice/internal/registry"
)

func TestLoadRegistryYAML(t *testing.T) {
	f, err := os.Open("../../testdata/registry.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	reg, err := registry.Load(f)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if reg.SchemaVersion != 1 {
		t.Fatalf("SchemaVersion = %d, want 1", reg.SchemaVersion)
	}
	if _, ok := reg.Backends["local_hermes"]; !ok {
		t.Fatalf("missing backend local_hermes")
	}
	if _, ok := reg.Models["default_chat"]; !ok {
		t.Fatalf("missing model default_chat")
	}
	if _, ok := reg.Persons["sve"]; !ok {
		t.Fatalf("missing person sve")
	}
	if _, ok := reg.Profiles["default"]; !ok {
		t.Fatalf("missing profile default")
	}
	if _, ok := reg.Devices["phone_ha"]; !ok {
		t.Fatalf("missing device phone_ha")
	}

	backend := reg.Backends["local_hermes"]
	if backend.APIKeyRef != "env:HERMES_API_KEY" {
		t.Fatalf("APIKeyRef = %q, want env:HERMES_API_KEY", backend.APIKeyRef)
	}
	if backend.APIKey != "" {
		t.Fatalf("fixture must not contain inline API key")
	}
}
