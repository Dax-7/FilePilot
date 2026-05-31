package backend

import "context"

// TransferBackend is the replaceable boundary for transfer engines.
// Concrete backend invocation is intentionally out of scope for Issue 1.
type TransferBackend interface {
	Name() string
	Version(ctx context.Context) (string, error)
	Send(ctx context.Context, req SendRequest) error
	Receive(ctx context.Context, req ReceiveRequest) error
}

type SendRequest struct {
	InputPath string
	SessionID string
}

type ReceiveRequest struct {
	SessionID string
	OutputDir string
}
