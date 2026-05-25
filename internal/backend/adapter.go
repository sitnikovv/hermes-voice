package backend

import "context"

// Adapter is the transport-neutral backend execution boundary.
type Adapter interface {
	Invoke(ctx context.Context, req Request) (*Response, error)
}
