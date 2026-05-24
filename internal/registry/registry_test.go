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
}

func TestLoadRejectsInlineBackendAPIKey(t *testing.T) {
	_, err := registry.Load(strings.NewReader(`
schema_version: 1
backends:
  local_hermes:
    type: hermes
    api_key: plaintext-secret
`))
	if !errors.Is(err, registry.ErrInlineSecret) {
		t.Fatalf("Load() error = %v, want ErrInlineSecret", err)
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

func TestResolveReturnsTypedLookupErrors(t *testing.T) {
	tests := []struct {
		name    string
		device  string
		alias   string
		mutate  func(*registry.Registry)
		wantErr error
	}{
		{
			name:    "missing device",
			device:  "missing_device",
			wantErr: registry.ErrDeviceNotFound,
		},
		{
			name:    "missing alias",
			device:  "phone_ha",
			alias:   "missing_alias",
			wantErr: registry.ErrAliasNotFound,
		},
		{
			name:   "missing person reference",
			device: "phone_ha",
			mutate: func(reg *registry.Registry) {
				device := reg.Devices["phone_ha"]
				device.DefaultPerson = "missing_person"
				reg.Devices["phone_ha"] = device
			},
			wantErr: registry.ErrPersonNotFound,
		},
		{
			name:   "missing profile reference",
			device: "phone_ha",
			mutate: func(reg *registry.Registry) {
				device := reg.Devices["phone_ha"]
				device.DefaultProfile = "missing_profile"
				reg.Devices["phone_ha"] = device
			},
			wantErr: registry.ErrProfileNotFound,
		},
		{
			name:   "missing model reference",
			device: "phone_ha",
			mutate: func(reg *registry.Registry) {
				profile := reg.Profiles["default"]
				profile.Model = "missing_model"
				reg.Profiles["default"] = profile
			},
			wantErr: registry.ErrModelNotFound,
		},
		{
			name:   "missing backend reference",
			device: "phone_ha",
			mutate: func(reg *registry.Registry) {
				model := reg.Models["default_chat"]
				model.Backend = "missing_backend"
				reg.Models["default_chat"] = model
			},
			wantErr: registry.ErrBackendNotFound,
		},
		{
			name:   "missing default person",
			device: "phone_ha",
			mutate: func(reg *registry.Registry) {
				device := reg.Devices["phone_ha"]
				device.DefaultPerson = ""
				reg.Devices["phone_ha"] = device
			},
			wantErr: registry.ErrMissingDefaultPerson,
		},
		{
			name:   "missing default profile",
			device: "phone_ha",
			mutate: func(reg *registry.Registry) {
				device := reg.Devices["phone_ha"]
				device.DefaultProfile = ""
				reg.Devices["phone_ha"] = device
			},
			wantErr: registry.ErrMissingDefaultProfile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := loadFixture(t)
			if tt.mutate != nil {
				tt.mutate(reg)
			}

			_, err := reg.Resolve(tt.device, tt.alias)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Resolve() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
