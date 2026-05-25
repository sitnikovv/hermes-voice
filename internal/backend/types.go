package backend

// Request carries one transport-neutral backend invocation attempt.
type Request struct {
	ID    string
	Input string

	DeviceID  string
	Alias     string
	PersonID  string
	ProfileID string
	ModelID   string
	BackendID string

	ModelName    string
	SystemPrompt string

	Metadata map[string]string
}

// Status describes the outcome shape of a backend invocation.
type Status string

const (
	StatusCompleted Status = "completed"
	StatusAccepted  Status = "accepted"
	StatusFailed    Status = "failed"
)

// Response carries a transport-neutral backend invocation result.
type Response struct {
	ID       string
	Output   string
	Status   Status
	TaskID   string
	Usage    *Usage
	Metadata map[string]string
}

// Usage reports backend token accounting when available.
type Usage struct {
	InputTokens  int
	OutputTokens int
}
