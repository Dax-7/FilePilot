package backend

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
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
	output, err := runBackendCommand(ctx, b.path, "--ignore-stdin", "send", "--code", req.SessionID, req.InputPath)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return fmt.Errorf("transfer backend send failed: %s: %w", sanitizeOutput(output, req.SessionID), err)
	}
	return nil
}

func (b CrocBackend) Receive(ctx context.Context, req ReceiveRequest) error {
	output, err := runBackendCommand(ctx, b.path, "--ignore-stdin", "--yes", "--out", req.OutputDir, req.SessionID)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return fmt.Errorf("transfer backend receive failed: %s: %w", sanitizeOutput(output, req.SessionID), err)
	}
	return nil
}

func runBackendCommand(ctx context.Context, path string, args ...string) ([]byte, error) {
	outputFile, err := os.CreateTemp("", "filepilot-backend-output-*")
	if err != nil {
		return nil, err
	}
	outputPath := outputFile.Name()
	defer os.Remove(outputPath)

	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Stdout = outputFile
	cmd.Stderr = outputFile
	err = cmd.Run()
	if closeErr := outputFile.Close(); err == nil && closeErr != nil {
		err = closeErr
	}
	output, readErr := readBackendOutput(outputPath)
	if err != nil {
		return output, err
	}
	return output, readErr
}

func readBackendOutput(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(io.LimitReader(file, 64*1024))
}

func sanitizeOutput(output []byte, sessionID string) string {
	text := strings.TrimSpace(string(output))
	if text == "" {
		return "no backend output"
	}
	if sessionID != "" {
		text = strings.ReplaceAll(text, sessionID, "[redacted-session-id]")
	}
	return text
}
