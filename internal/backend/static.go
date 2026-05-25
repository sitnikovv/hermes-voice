package backend

import "context"

type staticAdapter struct {
	resp Response
	err  error
}

// NewStaticAdapter returns a deterministic adapter that validates input and
// returns the configured response without any transport side effects.
func NewStaticAdapter(resp Response) Adapter {
	return staticAdapter{resp: resp}
}

// NewErrorAdapter returns a deterministic adapter that validates input and then
// returns the configured error without any transport side effects.
func NewErrorAdapter(err error) Adapter {
	return staticAdapter{err: err}
}

func (a staticAdapter) Invoke(ctx context.Context, req Request) (*Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if a.err != nil {
		return nil, a.err
	}
	resp := a.resp
	return &resp, nil
}
