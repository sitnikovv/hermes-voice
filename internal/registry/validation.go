package registry

import (
	"fmt"
	"regexp"
	"strings"
)

var registryIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

func (r *Registry) Validate() error {
	var issues []ValidationIssue
	addIssue := func(path, code, message string) {
		issues = append(issues, ValidationIssue{Path: path, Code: code, Message: message})
	}

	if r == nil {
		addIssue("registry", "missing_required", "registry is required")
		return validationResult(issues)
	}

	if len(r.Backends) == 0 {
		addIssue("backends", "missing_required", "backends must be present and non-empty")
	}
	if len(r.Models) == 0 {
		addIssue("models", "missing_required", "models must be present and non-empty")
	}
	if len(r.Persons) == 0 {
		addIssue("persons", "missing_required", "persons must be present and non-empty")
	}
	if len(r.Profiles) == 0 {
		addIssue("profiles", "missing_required", "profiles must be present and non-empty")
	}
	if len(r.Devices) == 0 {
		addIssue("devices", "missing_required", "devices must be present and non-empty")
	}

	for id, backend := range r.Backends {
		base := "backends." + id
		validateTopLevelID(addIssue, base, id)
		if backend.Type == "" {
			addIssue(base+".type", "missing_required", "type is required")
		}
		if backend.Type == "hermes" && backend.Endpoint == "" {
			addIssue(base+".endpoint", "missing_required", "endpoint is required for hermes backend")
		}
	}
	for id, model := range r.Models {
		base := "models." + id
		validateTopLevelID(addIssue, base, id)
		if model.Backend == "" {
			addIssue(base+".backend", "missing_required", "backend is required")
		}
		if model.Name == "" {
			addIssue(base+".name", "missing_required", "name is required")
		}
	}
	for id, person := range r.Persons {
		base := "persons." + id
		validateTopLevelID(addIssue, base, id)
		if person.DisplayName == "" {
			addIssue(base+".display_name", "missing_required", "display_name is required")
		}
	}
	for id, profile := range r.Profiles {
		base := "profiles." + id
		validateTopLevelID(addIssue, base, id)
		if profile.Person == "" {
			addIssue(base+".person", "missing_required", "person is required")
		}
		if profile.Model == "" {
			addIssue(base+".model", "missing_required", "model is required")
		}
	}
	for id, device := range r.Devices {
		base := "devices." + id
		validateTopLevelID(addIssue, base, id)
		if device.Label == "" {
			addIssue(base+".label", "missing_required", "label is required")
		}
		if device.DefaultPerson == "" {
			addIssue(base+".default_person", "missing_required", "default_person is required")
		}
		if device.DefaultProfile == "" {
			addIssue(base+".default_profile", "missing_required", "default_profile is required")
		}
		for alias, binding := range device.Aliases {
			aliasPath := base + ".aliases." + alias
			if strings.TrimSpace(alias) == "" {
				addIssue(aliasPath, "invalid_id", "alias key must not be empty")
			}
			if binding.Person == "" && binding.Profile == "" {
				addIssue(aliasPath, "missing_required", "alias binding requires person or profile")
			}
		}
	}

	return validationResult(issues)
}

type issueAdder func(path, code, message string)

func validateTopLevelID(addIssue issueAdder, path, id string) {
	if !registryIDPattern.MatchString(id) {
		addIssue(path, "invalid_id", fmt.Sprintf("id %q must match [a-z0-9][a-z0-9_-]*", id))
	}
}

func validationResult(issues []ValidationIssue) error {
	if len(issues) == 0 {
		return nil
	}
	return &ValidationError{Issues: issues}
}
