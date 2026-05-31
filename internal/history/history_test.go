package history

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"filepilot/internal/session"
)

func TestWriterAppendsTransferAttemptJSONL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "transfer-history.jsonl")
	writer := NewWriter(path)

	attempt := Attempt{
		Timestamp:     time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC),
		Mode:          "send",
		Status:        "completed",
		PayloadType:   "file",
		InputPath:     filepath.Join("data", "file.zip"),
		OutputPath:    "",
		PackagePath:   "",
		FileSize:      42,
		Packed:        false,
		Unpacked:      false,
		Backend:       "transfer-engine",
		BackendSource: "configured",
		SessionID:     "FP-river-copper-lamp-7K2Q9M4XP8",
		Duration:      1500 * time.Millisecond,
	}

	if err := writer.Append(attempt); err != nil {
		t.Fatalf("Append returned error: %v", err)
	}
	if err := writer.Append(attempt); err != nil {
		t.Fatalf("second Append returned error: %v", err)
	}

	rows := readHistoryRows(t, path)
	if len(rows) != 2 {
		t.Fatalf("expected 2 history rows, got %d", len(rows))
	}
	if rows[0]["mode"] != "send" || rows[0]["status"] != "completed" || rows[0]["payload_type"] != "file" {
		t.Fatalf("missing basic attempt fields: %#v", rows[0])
	}
	if rows[0]["session_id_redacted"] != session.Redact("FP-river-copper-lamp-7K2Q9M4XP8") {
		t.Fatalf("session_id_redacted mismatch: %#v", rows[0])
	}
	if rows[0]["duration_ms"] != float64(1500) {
		t.Fatalf("duration_ms mismatch: %#v", rows[0])
	}
}

func TestFailedAttemptRecordsErrorWithoutSensitiveValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "transfer-history.jsonl")
	writer := NewWriter(path)
	fullSessionID := "FP-river-copper-lamp-7K2Q9M4XP8"

	attempt := Attempt{
		Timestamp:         time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC),
		Mode:              "receive",
		Status:            "failed",
		PayloadType:       "directory_package",
		OutputPath:        filepath.Join("downloads", "results"),
		Backend:           "transfer-engine",
		BackendSource:     "path",
		SessionID:         fullSessionID,
		Duration:          250 * time.Millisecond,
		ErrorCode:         "TRANSFER_FAILED",
		ErrorMessage:      "transfer engine failed while receiving",
		BackendRawCommand: "backend send --code " + fullSessionID,
		BackendCredential: "secret-token-" + fullSessionID,
	}

	if err := writer.Append(attempt); err != nil {
		t.Fatalf("Append returned error: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	content := string(raw)
	if strings.Contains(content, fullSessionID) {
		t.Fatalf("history leaked full session id:\n%s", content)
	}
	if strings.Contains(content, "backend send --code") || strings.Contains(content, "secret-token") {
		t.Fatalf("history leaked backend raw command or credential:\n%s", content)
	}

	rows := readHistoryRows(t, path)
	if rows[0]["error_code"] != "TRANSFER_FAILED" || rows[0]["error_message"] == "" {
		t.Fatalf("missing error fields: %#v", rows[0])
	}
	if rows[0]["session_id_redacted"] != session.Redact(fullSessionID) {
		t.Fatalf("session redaction mismatch: %#v", rows[0])
	}
	if _, ok := rows[0]["backend_raw_command"]; ok {
		t.Fatalf("backend_raw_command should not be persisted: %#v", rows[0])
	}
	if _, ok := rows[0]["backend_credential"]; ok {
		t.Fatalf("backend_credential should not be persisted: %#v", rows[0])
	}
}

func TestRedactSessionIDHandlesShortAndEmptyValues(t *testing.T) {
	cases := map[string]string{
		"":              "",
		"FP-84K2":       "FP-****",
		"FP-a-b-c-d":    "FP-a-****-d",
		"not-filepilot": "****",
	}

	for input, want := range cases {
		got := RedactSessionID(input)
		if got != want {
			t.Fatalf("RedactSessionID(%q) = %q, want %q", input, got, want)
		}
	}
}

func readHistoryRows(t *testing.T, path string) []map[string]any {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer file.Close()

	var rows []map[string]any
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var row map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			t.Fatalf("history row is not JSON: %v\n%s", err, scanner.Text())
		}
		rows = append(rows, row)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("Scanner returned error: %v", err)
	}
	return rows
}
