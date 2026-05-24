package registry_test

import (
	"errors"
	"os"
	"strings"
	"testing"

	"hermes-voice/internal/registry"
)

func loadFixture(t *testing.T) *registry.Registry {
	t.Helper()
	f, err := os.Open("../../testdata/registry.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	reg, err := registry.Load(f)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	return reg
}

func TestLoadRejectsUnsupportedSchemaVersion(t *testing.T) {
	_, err := registry.Load(strings.NewReader("schema_version: 999\n"))
	if !errors.Is(err, registry.ErrUnsupportedSchemaVersion) {
		t.Fatalf("Load() error = %v, want ErrUnsupportedSchemaVersion", err)
	}
}

func TestLoadRegistryYAML(t *testing.T) {
	reg := loadFixture(t)

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

func TestResolveDefaultRegistryRoute(t *testing.T) {
	reg := loadFixture(t)

	resolved, err := reg.Resolve("phone_ha", "")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	assertResolvedIDs(t, resolved, "sve", "default", "default_chat", "local_hermes")
}

func TestResolveAliasRegistryRoute(t *testing.T) {
	reg := loadFixture(t)

	resolved, err := reg.Resolve("phone_ha", "coding")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	assertResolvedIDs(t, resolved, "sve", "coding", "default_chat", "local_hermes")
}

func assertResolvedIDs(t *testing.T, resolved *registry.ResolvedContext, personID, profileID, modelID, backendID string) {
	t.Helper()
	if resolved.PersonID != personID || resolved.ProfileID != profileID || resolved.ModelID != modelID || resolved.BackendID != backendID {
		t.Fatalf("resolved ids = person:%q profile:%q model:%q backend:%q", resolved.PersonID, resolved.ProfileID, resolved.ModelID, resolved.BackendID)
	}
}
