package registry

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	registryIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)
	secretRefPattern  = regexp.MustCompile(`^env:[A-Z_][A-Z0-9_]*$`)
)

func (r *Registry) Validate() error {
	var issues []ValidationIssue
	addIssue := func(path, code, message string) {
		issues = append(issues, ValidationIssue{Path: path, Code: code, Message: message})
	}

	if r == nil {
		addIssue("registry", "missing_required", "registry is required")
		return validationResult(issues)
	}

	if r.SchemaVersion != 1 {
		addIssue("schema_version", "unsupported_schema_version", fmt.Sprintf("schema_version must be 1, got %d", r.SchemaVersion))
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
		if isBlank(backend.Type) {
			addIssue(base+".type", "missing_required", "type is required")
		} else if backend.Type != "hermes" {
			addIssue(base+".type", "unknown_backend_type", "backend type must be hermes")
		}
		if backend.Type == "hermes" && isBlank(backend.Endpoint) {
			addIssue(base+".endpoint", "missing_required", "endpoint is required for hermes backend")
		}
		if backend.APIKeyRef != "" && !secretRefPattern.MatchString(backend.APIKeyRef) {
			addIssue(base+".api_key_ref", "invalid_secret_ref", "api_key_ref must be empty or match env:[A-Z_][A-Z0-9_]*")
		}
	}
	for id, model := range r.Models {
		base := "models." + id
		validateTopLevelID(addIssue, base, id)
		if isBlank(model.Backend) {
			addIssue(base+".backend", "missing_required", "backend is required")
		} else if _, ok := r.Backends[model.Backend]; !ok {
			addIssue(base+".backend", "missing_reference", "backend reference not found")
		}
		if isBlank(model.Name) {
			addIssue(base+".name", "missing_required", "name is required")
		}
	}
	for id, person := range r.Persons {
		base := "persons." + id
		validateTopLevelID(addIssue, base, id)
		if isBlank(person.DisplayName) {
			addIssue(base+".display_name", "missing_required", "display_name is required")
		}
	}
	for id, profile := range r.Profiles {
		base := "profiles." + id
		validateTopLevelID(addIssue, base, id)
		if isBlank(profile.Person) {
			addIssue(base+".person", "missing_required", "person is required")
		} else if _, ok := r.Persons[profile.Person]; !ok {
			addIssue(base+".person", "missing_reference", "person reference not found")
		}
		if isBlank(profile.Model) {
			addIssue(base+".model", "missing_required", "model is required")
		} else if _, ok := r.Models[profile.Model]; !ok {
			addIssue(base+".model", "missing_reference", "model reference not found")
		}
	}
	for id, device := range r.Devices {
		base := "devices." + id
		validateTopLevelID(addIssue, base, id)
		defaultPersonOK := false
		defaultProfileOK := false
		if isBlank(device.Label) {
			addIssue(base+".label", "missing_required", "label is required")
		}
		if isBlank(device.DefaultPerson) {
			addIssue(base+".default_person", "missing_required", "default_person is required")
		} else if _, ok := r.Persons[device.DefaultPerson]; !ok {
			addIssue(base+".default_person", "missing_reference", "default_person reference not found")
		} else {
			defaultPersonOK = true
		}
		if isBlank(device.DefaultProfile) {
			addIssue(base+".default_profile", "missing_required", "default_profile is required")
		} else if _, ok := r.Profiles[device.DefaultProfile]; !ok {
			addIssue(base+".default_profile", "missing_reference", "default_profile reference not found")
		} else {
			defaultProfileOK = true
		}
		if defaultPersonOK && defaultProfileOK {
			validateRoutePersonProfile(addIssue, base+".default_profile", device.DefaultPerson, device.DefaultProfile, r.Profiles[device.DefaultProfile])
		}
		for alias, binding := range device.Aliases {
			aliasPath := base + ".aliases." + alias
			if strings.TrimSpace(alias) == "" {
				addIssue(aliasPath, "invalid_id", "alias key must not be empty")
			}
			if isBlank(binding.Person) && isBlank(binding.Profile) {
				addIssue(aliasPath, "missing_required", "alias binding requires person or profile")
			}

			personID := device.DefaultPerson
			personOK := defaultPersonOK
			if !isBlank(binding.Person) {
				personID = binding.Person
				if _, ok := r.Persons[binding.Person]; !ok {
					addIssue(aliasPath+".person", "missing_reference", "person reference not found")
					personOK = false
				} else {
					personOK = true
				}
			}

			profileID := device.DefaultProfile
			profileOK := defaultProfileOK
			var profile Profile
			if profileOK {
				profile = r.Profiles[profileID]
			}
			if !isBlank(binding.Profile) {
				profileID = binding.Profile
				var ok bool
				profile, ok = r.Profiles[binding.Profile]
				if !ok {
					addIssue(aliasPath+".profile", "missing_reference", "profile reference not found")
					profileOK = false
				} else {
					profileOK = true
				}
			}
			if personOK && profileOK {
				validateRoutePersonProfile(addIssue, aliasPath+".profile", personID, profileID, profile)
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

func isBlank(s string) bool {
	return strings.TrimSpace(s) == ""
}

func validateRoutePersonProfile(addIssue issueAdder, path, personID, profileID string, profile Profile) {
	if profile.Person != personID {
		addIssue(path, "incompatible_person_profile", fmt.Sprintf("profile %q belongs to person %q, not %q", profileID, profile.Person, personID))
	}
}

func validationResult(issues []ValidationIssue) error {
	if len(issues) == 0 {
		return nil
	}
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Path != issues[j].Path {
			return issues[i].Path < issues[j].Path
		}
		if issues[i].Code != issues[j].Code {
			return issues[i].Code < issues[j].Code
		}
		return issues[i].Message < issues[j].Message
	})
	return &ValidationError{Issues: issues}
}
