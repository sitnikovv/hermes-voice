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
