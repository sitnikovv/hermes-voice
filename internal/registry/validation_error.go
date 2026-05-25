package registry

import (
	"errors"
	"fmt"
	"strings"
)

type ValidationIssue struct {
	Path    string
	Code    string
	Message string
}

type ValidationError struct {
	Issues []ValidationIssue
}

func (e *ValidationError) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return ErrInvalidRegistry.Error()
	}

	parts := make([]string, 0, len(e.Issues))
	for _, issue := range e.Issues {
		parts = append(parts, fmt.Sprintf("%s: %s", issue.Path, issue.Message))
	}
	return fmt.Sprintf("%s: %s", ErrInvalidRegistry, strings.Join(parts, "; "))
}

func (e *ValidationError) Is(target error) bool {
	return errors.Is(ErrInvalidRegistry, target)
}
