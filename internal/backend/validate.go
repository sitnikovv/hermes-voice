package backend

import "fmt"

// Validate rejects requests that do not contain the transport-neutral fields
// required to attempt backend invocation.
func (r Request) Validate() error {
	if r.Input == "" {
		return fmt.Errorf("%w: input is required", ErrInvalidRequest)
	}
	if r.PersonID == "" {
		return fmt.Errorf("%w: person ID is required", ErrInvalidRequest)
	}
	if r.ProfileID == "" {
		return fmt.Errorf("%w: profile ID is required", ErrInvalidRequest)
	}
	if r.ModelID == "" {
		return fmt.Errorf("%w: model ID is required", ErrInvalidRequest)
	}
	if r.BackendID == "" {
		return fmt.Errorf("%w: backend ID is required", ErrInvalidRequest)
	}
	if r.ModelName == "" {
		return fmt.Errorf("%w: model name is required", ErrInvalidRequest)
	}
	return nil
}
