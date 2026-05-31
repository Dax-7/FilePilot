package backend

import (
	"context"
	"fmt"
	"os/exec"
)

type CrocBackend struct {
	path string
}

func NewCrocBackend(path string) CrocBackend {
	return CrocBackend{path: path}
}

func (b CrocBackend) Name() string {
	return "transfer-engine"
}

func (b CrocBackend) Version(ctx context.Context) (string, error) {
	return DetectVersion(ctx, b.path), nil
}

func (b CrocBackend) Send(ctx context.Context, req SendRequest) error {
	if err := exec.CommandContext(ctx, b.path, "send", "--code", req.SessionID, req.InputPath).Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return fmt.Errorf("transfer backend send failed: %w", err)
	}
	return nil
}

func (b CrocBackend) Receive(ctx context.Context, req ReceiveRequest) error {
	if err := exec.CommandContext(ctx, b.path, "--yes", "--out", req.OutputDir, req.SessionID).Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return fmt.Errorf("transfer backend receive failed: %w", err)
	}
	return nil
}
