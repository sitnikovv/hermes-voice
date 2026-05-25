package registry

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

	return validationResult(issues)
}

func validationResult(issues []ValidationIssue) error {
	if len(issues) == 0 {
		return nil
	}
	return &ValidationError{Issues: issues}
}
