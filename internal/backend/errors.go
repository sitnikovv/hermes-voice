package backend

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidRequest     = errors.New("invalid backend request")
	ErrUnsupportedBackend = errors.New("unsupported backend")
	ErrInvocationFailed   = errors.New("backend invocation failed")
	ErrTemporary          = errors.New("temporary backend error")
	ErrUnauthorized       = errors.New("backend unauthorized")
)

// Error adds backend operation details while preserving the wrapped cause for
// errors.Is/errors.As callers.
type Error struct {
	Op        string
	BackendID string
	Code      string
	Err       error
}

func (e *Error) Error() string {
	if e == nil {
		return "backend error"
	}

	msg := "backend error"
	if e.Op != "" {
		msg = fmt.Sprintf("%s: %s", e.Op, msg)
	}
	if e.BackendID != "" {
		msg = fmt.Sprintf("%s: backend=%s", msg, e.BackendID)
	}
	if e.Code != "" {
		msg = fmt.Sprintf("%s: code=%s", msg, e.Code)
	}
	if e.Err != nil {
		msg = fmt.Sprintf("%s: %v", msg, e.Err)
	}
	return msg
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
