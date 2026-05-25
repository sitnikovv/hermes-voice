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

func assertValidationIssue(t *testing.T, err error, path, code string) {
	t.Helper()
	if !errors.Is(err, registry.ErrInvalidRegistry) {
		t.Fatalf("error = %v, want ErrInvalidRegistry", err)
	}
	var validationErr *registry.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("error = %T, want *ValidationError", err)
	}
	for _, issue := range validationErr.Issues {
		if issue.Path == path && issue.Code == code {
			return
		}
	}
	t.Fatalf("issues = %#v, want path=%q code=%q", validationErr.Issues, path, code)
}
