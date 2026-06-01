package backend

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
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
	args := []string{"--ignore-stdin", "send", req.InputPath}
	env := []string{"CROC_SECRET=" + req.SessionID}
	if runtime.GOOS == "windows" {
		args = []string{"--ignore-stdin", "send", "--code", req.SessionID, req.InputPath}
		env = nil
	}
	output, err := runBackendCommand(ctx, b.path, env, args...)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return fmt.Errorf("transfer backend send failed: %s: %w", sanitizeOutput(output, req.SessionID), err)
	}
	return nil
}

func (b CrocBackend) Receive(ctx context.Context, req ReceiveRequest) error {
	output, err := runBackendCommand(ctx, b.path, nil, "--ignore-stdin", "--yes", "--out", req.OutputDir, req.SessionID)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return fmt.Errorf("transfer backend receive failed: %s: %w", sanitizeOutput(output, req.SessionID), err)
	}
	return nil
}

func runBackendCommand(ctx context.Context, path string, env []string, args ...string) ([]byte, error) {
	outputFile, err := os.CreateTemp("", "filepilot-backend-output-*")
	if err != nil {
		return nil, err
	}
	outputPath := outputFile.Name()
	defer os.Remove(outputPath)

	cmd := exec.CommandContext(ctx, path, args...)
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
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
