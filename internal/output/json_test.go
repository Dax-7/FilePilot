package output

import (
	"bytes"
	"encoding/json"
	"testing"

	"filepilot/internal/fperrors"
)

func TestWriteSuccessIncludesStableAgentFields(t *testing.T) {
	var stdout bytes.Buffer
	result := Result{
		OK:     true,
		Status: "not_implemented",
		Mode:   "send",
		Error:  nil,
		Fields: map[string]any{
			"backend": "unresolved",
		},
	}

	if err := WriteJSON(&stdout, result); err != nil {
		t.Fatalf("WriteJSON returned error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("response is not JSON: %v\n%s", err, stdout.String())
	}
	if payload["ok"] != true || payload["status"] != "not_implemented" || payload["mode"] != "send" {
		t.Fatalf("missing stable success fields: %#v", payload)
	}
	if payload["error"] != nil {
		t.Fatalf("success error field = %#v, want nil", payload["error"])
	}
	if payload["backend"] != "unresolved" {
		t.Fatalf("extra field not preserved: %#v", payload)
	}
}

func TestWriteFailureIncludesStableErrorFields(t *testing.T) {
	var stdout bytes.Buffer
	fpErr := fperrors.New(fperrors.InvalidArgument, "Unsupported command.", "Run filepilot --help.")

	if err := WriteJSON(&stdout, Failure("cli", fpErr)); err != nil {
		t.Fatalf("WriteJSON returned error: %v", err)
	}

	var payload struct {
		OK     bool   `json:"ok"`
		Status string `json:"status"`
		Mode   string `json:"mode"`
		Error  struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Hint    string `json:"hint"`
		} `json:"error"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("response is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.OK || payload.Status != "failed" || payload.Mode != "cli" {
		t.Fatalf("missing stable failure fields: %#v", payload)
	}
	if payload.Error.Code != "INVALID_ARGUMENT" || payload.Error.Message == "" || payload.Error.Hint == "" {
		t.Fatalf("missing stable error fields: %#v", payload.Error)
	}
}
