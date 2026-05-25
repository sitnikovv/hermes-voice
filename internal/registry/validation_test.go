package registry_test

import (
	"errors"
	"strings"
	"testing"

	"hermes-voice/internal/registry"
)

func TestValidationErrorIsInvalidRegistryAndListsIssuePaths(t *testing.T) {
	err := &registry.ValidationError{Issues: []registry.ValidationIssue{
		{Path: "backends.local_hermes.endpoint", Code: "missing_required", Message: "endpoint is required"},
		{Path: "profiles.default.model", Code: "missing_reference", Message: "model not found"},
	}}

	if !errors.Is(err, registry.ErrInvalidRegistry) {
		t.Fatalf("errors.Is(err, ErrInvalidRegistry) = false")
	}
	msg := err.Error()
	for _, want := range []string{"backends.local_hermes.endpoint", "profiles.default.model"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("Error() = %q, want path %q", msg, want)
		}
	}
}

func TestValidateAcceptsFixture(t *testing.T) {
	reg := loadFixture(t)
	if err := reg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateRequiresTopLevelSections(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*registry.Registry)
		path   string
	}{
		{name: "missing backends", mutate: func(reg *registry.Registry) { reg.Backends = nil }, path: "backends"},
		{name: "empty backends", mutate: func(reg *registry.Registry) { reg.Backends = map[string]registry.Backend{} }, path: "backends"},
		{name: "missing models", mutate: func(reg *registry.Registry) { reg.Models = nil }, path: "models"},
		{name: "empty models", mutate: func(reg *registry.Registry) { reg.Models = map[string]registry.Model{} }, path: "models"},
		{name: "missing persons", mutate: func(reg *registry.Registry) { reg.Persons = nil }, path: "persons"},
		{name: "empty persons", mutate: func(reg *registry.Registry) { reg.Persons = map[string]registry.Person{} }, path: "persons"},
		{name: "missing profiles", mutate: func(reg *registry.Registry) { reg.Profiles = nil }, path: "profiles"},
		{name: "empty profiles", mutate: func(reg *registry.Registry) { reg.Profiles = map[string]registry.Profile{} }, path: "profiles"},
		{name: "missing devices", mutate: func(reg *registry.Registry) { reg.Devices = nil }, path: "devices"},
		{name: "empty devices", mutate: func(reg *registry.Registry) { reg.Devices = map[string]registry.Device{} }, path: "devices"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := loadFixture(t)
			tt.mutate(reg)

			err := reg.Validate()
			assertValidationIssue(t, err, tt.path, "missing_required")
		})
	}
}

func TestValidateNilRegistryFails(t *testing.T) {
	var reg *registry.Registry
	assertValidationIssue(t, reg.Validate(), "registry", "missing_required")
}

func TestLoadRejectsUnknownYAMLFields(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{name: "top-level", yaml: strings.Replace(validRegistryYAML(), "devices:", "unexpected: true\ndevices:", 1)},
		{name: "nested", yaml: strings.Replace(validRegistryYAML(), "endpoint: http://127.0.0.1:8080", "endpoint: http://127.0.0.1:8080\n    unexpected: true", 1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := registry.Load(strings.NewReader(tt.yaml)); err == nil {
				t.Fatalf("Load() error = nil, want unknown field error")
			}
		})
	}
}

func TestLoadReturnsValidationErrorForInvalidRegistry(t *testing.T) {
	_, err := registry.Load(strings.NewReader("schema_version: 1\nbackends: {}\n"))
	assertValidationIssue(t, err, "backends", "missing_required")
}

func TestLoadInlineAPIKeyStillMatchesInlineSecret(t *testing.T) {
	yaml := strings.Replace(validRegistryYAML(), "api_key_ref: env:HERMES_API_KEY", "api_key: plaintext-secret", 1)
	_, err := registry.Load(strings.NewReader(yaml))
	if !errors.Is(err, registry.ErrInlineSecret) {
		t.Fatalf("Load() error = %v, want ErrInlineSecret", err)
	}
}

func validRegistryYAML() string {
	return `schema_version: 1
backends:
  local_hermes:
    type: hermes
    endpoint: http://127.0.0.1:8080
    api_key_ref: env:HERMES_API_KEY
models:
  default_chat:
    backend: local_hermes
    name: hermes-default
persons:
  sve:
    display_name: Sve
profiles:
  default:
    person: sve
    model: default_chat
  coding:
    person: sve
    model: default_chat
devices:
  phone_ha:
    label: Android HA Assist
    default_person: sve
    default_profile: default
    aliases:
      coding:
        person: sve
        profile: coding
`
}

func TestValidateIDsAndRequiredFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*registry.Registry)
		path   string
		code   string
	}{
		{name: "invalid backend id", mutate: func(reg *registry.Registry) { reg.Backends["Bad"] = reg.Backends["local_hermes"] }, path: "backends.Bad", code: "invalid_id"},
		{name: "invalid model id", mutate: func(reg *registry.Registry) { reg.Models["bad.id"] = reg.Models["default_chat"] }, path: "models.bad.id", code: "invalid_id"},
		{name: "invalid person id", mutate: func(reg *registry.Registry) { reg.Persons["-bad"] = reg.Persons["sve"] }, path: "persons.-bad", code: "invalid_id"},
		{name: "invalid profile id", mutate: func(reg *registry.Registry) { reg.Profiles["bad id"] = reg.Profiles["default"] }, path: "profiles.bad id", code: "invalid_id"},
		{name: "invalid device id", mutate: func(reg *registry.Registry) { reg.Devices["bad.id"] = reg.Devices["phone_ha"] }, path: "devices.bad.id", code: "invalid_id"},
		{name: "empty alias key", mutate: func(reg *registry.Registry) {
			d := reg.Devices["phone_ha"]
			d.Aliases[" \t"] = registry.AliasBinding{Person: "sve"}
			reg.Devices["phone_ha"] = d
		}, path: "devices.phone_ha.aliases. \t", code: "invalid_id"},
		{name: "backend missing type", mutate: func(reg *registry.Registry) {
			b := reg.Backends["local_hermes"]
			b.Type = ""
			reg.Backends["local_hermes"] = b
		}, path: "backends.local_hermes.type", code: "missing_required"},
		{name: "hermes missing endpoint", mutate: func(reg *registry.Registry) {
			b := reg.Backends["local_hermes"]
			b.Endpoint = ""
			reg.Backends["local_hermes"] = b
		}, path: "backends.local_hermes.endpoint", code: "missing_required"},
		{name: "model missing backend", mutate: func(reg *registry.Registry) {
			m := reg.Models["default_chat"]
			m.Backend = ""
			reg.Models["default_chat"] = m
		}, path: "models.default_chat.backend", code: "missing_required"},
		{name: "model missing name", mutate: func(reg *registry.Registry) {
			m := reg.Models["default_chat"]
			m.Name = ""
			reg.Models["default_chat"] = m
		}, path: "models.default_chat.name", code: "missing_required"},
		{name: "person missing display name", mutate: func(reg *registry.Registry) { p := reg.Persons["sve"]; p.DisplayName = ""; reg.Persons["sve"] = p }, path: "persons.sve.display_name", code: "missing_required"},
		{name: "profile missing person", mutate: func(reg *registry.Registry) { p := reg.Profiles["default"]; p.Person = ""; reg.Profiles["default"] = p }, path: "profiles.default.person", code: "missing_required"},
		{name: "profile missing model", mutate: func(reg *registry.Registry) { p := reg.Profiles["default"]; p.Model = ""; reg.Profiles["default"] = p }, path: "profiles.default.model", code: "missing_required"},
		{name: "device missing label", mutate: func(reg *registry.Registry) { d := reg.Devices["phone_ha"]; d.Label = ""; reg.Devices["phone_ha"] = d }, path: "devices.phone_ha.label", code: "missing_required"},
		{name: "device missing default person", mutate: func(reg *registry.Registry) {
			d := reg.Devices["phone_ha"]
			d.DefaultPerson = ""
			reg.Devices["phone_ha"] = d
		}, path: "devices.phone_ha.default_person", code: "missing_required"},
		{name: "device missing default profile", mutate: func(reg *registry.Registry) {
			d := reg.Devices["phone_ha"]
			d.DefaultProfile = ""
			reg.Devices["phone_ha"] = d
		}, path: "devices.phone_ha.default_profile", code: "missing_required"},
		{name: "empty alias binding", mutate: func(reg *registry.Registry) {
			d := reg.Devices["phone_ha"]
			d.Aliases["empty"] = registry.AliasBinding{}
			reg.Devices["phone_ha"] = d
		}, path: "devices.phone_ha.aliases.empty", code: "missing_required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := loadFixture(t)
			tt.mutate(reg)
			assertValidationIssue(t, reg.Validate(), tt.path, tt.code)
		})
	}
}

func TestValidateAggregatesIndependentIssues(t *testing.T) {
	reg := loadFixture(t)
	reg.Backends["Bad"] = registry.Backend{}
	reg.Models["default_chat"] = registry.Model{}
	err := reg.Validate()
	assertValidationIssue(t, err, "backends.Bad", "invalid_id")
	assertValidationIssue(t, err, "models.default_chat.backend", "missing_required")
	if len(validationIssues(t, err)) < 2 {
		t.Fatalf("got fewer than 2 issues: %v", err)
	}
}

func TestValidateReferencesAndRouteConsistency(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*registry.Registry)
		path   string
		code   string
	}{
		{name: "model backend", mutate: func(reg *registry.Registry) {
			m := reg.Models["default_chat"]
			m.Backend = "missing_backend"
			reg.Models["default_chat"] = m
		}, path: "models.default_chat.backend", code: "missing_reference"},
		{name: "profile person", mutate: func(reg *registry.Registry) {
			p := reg.Profiles["default"]
			p.Person = "missing_person"
			reg.Profiles["default"] = p
		}, path: "profiles.default.person", code: "missing_reference"},
		{name: "profile model", mutate: func(reg *registry.Registry) {
			p := reg.Profiles["default"]
			p.Model = "missing_model"
			reg.Profiles["default"] = p
		}, path: "profiles.default.model", code: "missing_reference"},
		{name: "device default person", mutate: func(reg *registry.Registry) {
			d := reg.Devices["phone_ha"]
			d.DefaultPerson = "missing_person"
			reg.Devices["phone_ha"] = d
		}, path: "devices.phone_ha.default_person", code: "missing_reference"},
		{name: "device default profile", mutate: func(reg *registry.Registry) {
			d := reg.Devices["phone_ha"]
			d.DefaultProfile = "missing_profile"
			reg.Devices["phone_ha"] = d
		}, path: "devices.phone_ha.default_profile", code: "missing_reference"},
		{name: "alias person", mutate: func(reg *registry.Registry) {
			d := reg.Devices["phone_ha"]
			d.Aliases["coding"] = registry.AliasBinding{Person: "missing_person", Profile: "coding"}
			reg.Devices["phone_ha"] = d
		}, path: "devices.phone_ha.aliases.coding.person", code: "missing_reference"},
		{name: "alias profile", mutate: func(reg *registry.Registry) {
			d := reg.Devices["phone_ha"]
			d.Aliases["coding"] = registry.AliasBinding{Person: "sve", Profile: "missing_profile"}
			reg.Devices["phone_ha"] = d
		}, path: "devices.phone_ha.aliases.coding.profile", code: "missing_reference"},
		{name: "default person profile mismatch", mutate: func(reg *registry.Registry) {
			reg.Persons["other"] = registry.Person{DisplayName: "Other"}
			p := reg.Profiles["default"]
			p.Person = "other"
			reg.Profiles["default"] = p
		}, path: "devices.phone_ha.default_profile", code: "incompatible_person_profile"},
		{name: "alias person profile mismatch", mutate: func(reg *registry.Registry) {
			reg.Persons["other"] = registry.Person{DisplayName: "Other"}
			reg.Profiles["other_profile"] = registry.Profile{Person: "other", Model: "default_chat"}
			d := reg.Devices["phone_ha"]
			d.Aliases["coding"] = registry.AliasBinding{Person: "sve", Profile: "other_profile"}
			reg.Devices["phone_ha"] = d
		}, path: "devices.phone_ha.aliases.coding.profile", code: "incompatible_person_profile"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := loadFixture(t)
			tt.mutate(reg)
			assertValidationIssue(t, reg.Validate(), tt.path, tt.code)
		})
	}
}

func TestValidateSecretRefsAndBackendTypes(t *testing.T) {
	t.Run("valid empty api_key_ref", func(t *testing.T) {
		reg := loadFixture(t)
		b := reg.Backends["local_hermes"]
		b.APIKeyRef = ""
		reg.Backends["local_hermes"] = b
		if err := reg.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	})
	t.Run("valid env api_key_ref", func(t *testing.T) {
		reg := loadFixture(t)
		b := reg.Backends["local_hermes"]
		b.APIKeyRef = "env:HERMES_API_KEY"
		reg.Backends["local_hermes"] = b
		if err := reg.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	})

	tests := []struct {
		name      string
		apiKeyRef string
	}{
		{name: "invalid prefix", apiKeyRef: "file:/secret"},
		{name: "invalid env name", apiKeyRef: "env:hermes_api_key"},
		{name: "invalid empty env name", apiKeyRef: "env:"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := loadFixture(t)
			b := reg.Backends["local_hermes"]
			b.APIKeyRef = tt.apiKeyRef
			reg.Backends["local_hermes"] = b
			assertValidationIssue(t, reg.Validate(), "backends.local_hermes.api_key_ref", "invalid_secret_ref")
		})
	}

	t.Run("unknown backend type", func(t *testing.T) {
		reg := loadFixture(t)
		b := reg.Backends["local_hermes"]
		b.Type = "other"
		reg.Backends["local_hermes"] = b
		assertValidationIssue(t, reg.Validate(), "backends.local_hermes.type", "unknown_backend_type")
	})
}

func assertValidationIssue(t *testing.T, err error, path, code string) {
	t.Helper()
	issues := validationIssues(t, err)
	for _, issue := range issues {
		if issue.Path == path && issue.Code == code {
			return
		}
	}
	t.Fatalf("issues = %#v, want path=%q code=%q", issues, path, code)
}

func validationIssues(t *testing.T, err error) []registry.ValidationIssue {
	t.Helper()
	if !errors.Is(err, registry.ErrInvalidRegistry) {
		t.Fatalf("error = %v, want ErrInvalidRegistry", err)
	}
	var validationErr *registry.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("error = %T, want *ValidationError", err)
	}
	return validationErr.Issues
}
