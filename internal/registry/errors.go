package registry

import (
	"errors"
	"fmt"
)

var (
	ErrUnsupportedSchemaVersion = errors.New("unsupported schema version")
	ErrDeviceNotFound           = errors.New("device not found")
	ErrAliasNotFound            = errors.New("alias not found")
	ErrPersonNotFound           = errors.New("person not found")
	ErrProfileNotFound          = errors.New("profile not found")
	ErrModelNotFound            = errors.New("model not found")
	ErrBackendNotFound          = errors.New("backend not found")
	ErrMissingDefaultPerson     = errors.New("missing default person")
	ErrMissingDefaultProfile    = errors.New("missing default profile")
	ErrInlineSecret             = errors.New("inline secret not allowed")
)

func registryError(sentinel error, format string, args ...any) error {
	return fmt.Errorf("%w: %s", sentinel, fmt.Sprintf(format, args...))
}
