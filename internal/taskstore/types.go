package taskstore

import (
	"context"
	"errors"
	"time"

	"hermes-voice/internal/backend"
)

type Status string

const (
	StatusAccepted  Status = "accepted"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

type Error struct {
	Code    string
	Message string
}

type Record struct {
	TaskID      string
	RequestID   string
	Status      Status
	Request     backend.Request
	Response    *backend.Response
	Error       *Error
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time
	Metadata    map[string]string
}

type Store interface {
	CreateAccepted(ctx context.Context, rec Record) error
	Complete(ctx context.Context, taskID string, resp *backend.Response) error
	Fail(ctx context.Context, taskID string, taskErr Error) error
	Get(ctx context.Context, taskID string) (Record, bool, error)
}

var (
	ErrInvalidTaskID   = errors.New("taskstore: invalid task id")
	ErrTaskExists      = errors.New("taskstore: task exists")
	ErrTaskNotFound    = errors.New("taskstore: task not found")
	ErrInvalidResponse = errors.New("taskstore: invalid response")
)
