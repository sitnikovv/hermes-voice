package registry_test

import (
	"errors"
	"strings"
	"testing"

	"hermes-voice/internal/registry"
)

func TestResolveDefaultRoutePreservesEmptyAlias(t *testing.T) {
	reg := loadRoutingFixture(t)

	ctx, err := reg.Resolve("phone_ha", "")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	assertRoutingResolvedIDs(t, ctx, "phone_ha", "", "sve", "default", "default_chat", "local_hermes")
}

func TestResolveDoesNotNormalizeAliases(t *testing.T) {
	reg := loadRoutingFixture(t)

	for _, alias := range []string{"Status", " status ", "STATUS"} {
		t.Run(alias, func(t *testing.T) {
			ctx, err := reg.Resolve("phone_ha", alias)
			if !errors.Is(err, registry.ErrAliasNotFound) {
				t.Fatalf("Resolve(%q) error = %v, want ErrAliasNotFound", alias, err)
			}
			if ctx != nil {
				t.Fatalf("Resolve(%q) context = %#v, want nil", alias, ctx)
			}
		})
	}
}

func TestResolveAliasWithOnlyProfileInheritsDefaultPerson(t *testing.T) {
	reg := loadRoutingFixture(t)

	ctx, err := reg.Resolve("phone_ha", "coding")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	assertRoutingResolvedIDs(t, ctx, "phone_ha", "coding", "sve", "coding", "default_chat", "local_hermes")
}

func TestResolveAliasWithOnlyPersonInheritsDefaultProfile(t *testing.T) {
	reg := loadRoutingFixture(t)

	ctx, err := reg.Resolve("kitchen_voice", "guest")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	assertRoutingResolvedIDs(t, ctx, "kitchen_voice", "guest", "guest", "guest_default", "default_chat", "local_hermes")
}

func TestResolveAliasesAreDeviceLocal(t *testing.T) {
	reg := loadRoutingFixture(t)

	phone, err := reg.Resolve("phone_ha", "status")
	if err != nil {
		t.Fatalf("Resolve(phone_ha, status) error = %v", err)
	}
	kitchen, err := reg.Resolve("kitchen_voice", "status")
	if err != nil {
		t.Fatalf("Resolve(kitchen_voice, status) error = %v", err)
	}

	assertRoutingResolvedIDs(t, phone, "phone_ha", "status", "sve", "status", "status_model", "local_hermes")
	assertRoutingResolvedIDs(t, kitchen, "kitchen_voice", "status", "guest", "guest_default", "default_chat", "local_hermes")
}

func TestResolveUnknownAliasDoesNotFallback(t *testing.T) {
	reg := loadRoutingFixture(t)

	ctx, err := reg.Resolve("phone_ha", "missing")
	if !errors.Is(err, registry.ErrAliasNotFound) {
		t.Fatalf("Resolve() error = %v, want ErrAliasNotFound", err)
	}
	if ctx != nil {
		t.Fatalf("Resolve() context = %#v, want nil", ctx)
	}
}

func TestResolveReturnedContextContainsRouteData(t *testing.T) {
	reg := loadRoutingFixture(t)

	ctx, err := reg.Resolve("phone_ha", "status")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	assertRoutingResolvedIDs(t, ctx, "phone_ha", "status", "sve", "status", "status_model", "local_hermes")
	if ctx.Person.DisplayName != "Sve" {
		t.Fatalf("Person.DisplayName = %q, want Sve", ctx.Person.DisplayName)
	}
	if ctx.Profile.SystemPrompt != "Status only" {
		t.Fatalf("Profile.SystemPrompt = %q, want Status only", ctx.Profile.SystemPrompt)
	}
	if ctx.Model.Name != "hermes-status" {
		t.Fatalf("Model.Name = %q, want hermes-status", ctx.Model.Name)
	}
	if ctx.Backend.Endpoint != "http://127.0.0.1:8080" {
		t.Fatalf("Backend.Endpoint = %q, want endpoint", ctx.Backend.Endpoint)
	}
	if ctx.Backend.APIKeyRef != "env:HERMES_API_KEY" {
		t.Fatalf("Backend.APIKeyRef = %q, want env:HERMES_API_KEY", ctx.Backend.APIKeyRef)
	}
}

func loadRoutingFixture(t *testing.T) *registry.Registry {
	t.Helper()
	reg, err := registry.Load(strings.NewReader(routingRegistryYAML()))
	if err != nil {
		t.Fatalf("Load routing fixture: %v", err)
	}
	return reg
}

func assertRoutingResolvedIDs(t *testing.T, ctx *registry.ResolvedContext, deviceID, alias, personID, profileID, modelID, backendID string) {
	t.Helper()
	if ctx == nil {
		t.Fatal("ResolvedContext is nil")
	}
	if ctx.DeviceID != deviceID || ctx.Alias != alias || ctx.PersonID != personID || ctx.ProfileID != profileID || ctx.ModelID != modelID || ctx.BackendID != backendID {
		t.Fatalf("resolved IDs = device=%q alias=%q person=%q profile=%q model=%q backend=%q; want device=%q alias=%q person=%q profile=%q model=%q backend=%q",
			ctx.DeviceID, ctx.Alias, ctx.PersonID, ctx.ProfileID, ctx.ModelID, ctx.BackendID,
			deviceID, alias, personID, profileID, modelID, backendID)
	}
}

func routingRegistryYAML() string {
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
  status_model:
    backend: local_hermes
    name: hermes-status
persons:
  sve:
    display_name: Sve
  guest:
    display_name: Guest
profiles:
  default:
    person: sve
    model: default_chat
    system_prompt: Default profile
  coding:
    person: sve
    model: default_chat
    system_prompt: Coding profile
  status:
    person: sve
    model: status_model
    system_prompt: Status only
  guest_default:
    person: guest
    model: default_chat
    system_prompt: Guest default
  guest_status:
    person: guest
    model: status_model
    system_prompt: Guest status

devices:
  phone_ha:
    label: Android HA Assist
    default_person: sve
    default_profile: default
    aliases:
      coding:
        profile: coding
      status:
        profile: status
  kitchen_voice:
    label: Kitchen Voice
    default_person: guest
    default_profile: guest_default
    aliases:
      guest:
        person: guest
      status:
        profile: guest_default
`
}
