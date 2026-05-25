package backend

import (
	"errors"
	"strings"
	"testing"
)

func TestErrorWrapsSentinel(t *testing.T) {
	err := &Error{Op: "invoke", BackendID: "backend-1", Code: "bad_request", Err: ErrInvalidRequest}

	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("errors.Is(%v, ErrInvalidRequest) = false, want true", err)
	}
}

func TestErrorMessageIncludesNonSecretDetails(t *testing.T) {
	err := &Error{Op: "invoke", BackendID: "backend-1", Code: "temporary", Err: ErrTemporary}
	msg := err.Error()

	for _, want := range []string{"invoke", "backend-1", "temporary"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("Error() = %q, want to contain %q", msg, want)
		}
	}
}

func TestErrorDoesNotRequireOrExposeSecretFields(t *testing.T) {
	err := &Error{Op: "invoke", BackendID: "backend-1", Code: "unauthorized", Err: ErrUnauthorized}
	msg := err.Error()

	for _, forbidden := range []string{"secret", "token", "key", "password"} {
		if strings.Contains(strings.ToLower(msg), forbidden) {
			t.Fatalf("Error() = %q, want no secret-like field %q", msg, forbidden)
		}
	}
}

func TestSentinelErrorsExist(t *testing.T) {
	for name, err := range map[string]error{
		"ErrInvalidRequest":     ErrInvalidRequest,
		"ErrUnsupportedBackend": ErrUnsupportedBackend,
		"ErrInvocationFailed":   ErrInvocationFailed,
		"ErrTemporary":          ErrTemporary,
		"ErrUnauthorized":       ErrUnauthorized,
	} {
		if err == nil {
			t.Fatalf("%s is nil", name)
		}
	}
}
