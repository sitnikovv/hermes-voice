package backend

import (
	"fmt"
	"strings"
)

// Validate rejects requests that do not contain the transport-neutral fields
// required to attempt backend invocation.
func (r Request) Validate() error {
	if isBlank(r.Input) {
		return fmt.Errorf("%w: input is required", ErrInvalidRequest)
	}
	if isBlank(r.PersonID) {
		return fmt.Errorf("%w: person ID is required", ErrInvalidRequest)
	}
	if isBlank(r.ProfileID) {
		return fmt.Errorf("%w: profile ID is required", ErrInvalidRequest)
	}
	if isBlank(r.ModelID) {
		return fmt.Errorf("%w: model ID is required", ErrInvalidRequest)
	}
	if isBlank(r.BackendID) {
		return fmt.Errorf("%w: backend ID is required", ErrInvalidRequest)
	}
	if isBlank(r.ModelName) {
		return fmt.Errorf("%w: model name is required", ErrInvalidRequest)
	}
	return nil
}

func isBlank(value string) bool {
	return strings.TrimSpace(value) == ""
}
