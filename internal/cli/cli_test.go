package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"filepilot/internal/packaging"
)

func runCLI(args ...string) (int, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestHelpListsMVPCommands(t *testing.T) {
	code, stdout, stderr := runCLI("--help")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%s", code, stderr)
	}

	for _, command := range []string{"send", "receive", "pack", "doctor", "clean", "config"} {
		if !strings.Contains(stdout, command) {
			t.Fatalf("expected help to list %q; help:\n%s", command, stdout)
		}
	}
}

func TestHelpDoesNotExposeOutOfScopeCommands(t *testing.T) {
	code, stdout, stderr := runCLI("--help")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%s", code, stderr)
	}

	for _, command := range []string{"history", "serve", "daemon", "gui", "sync", "resume", "login", "recv"} {
		if strings.Contains(stdout, command) {
			t.Fatalf("did not expect help to expose %q; help:\n%s", command, stdout)
		}
	}
}

func TestReceiveHelpUsesCanonicalCommand(t *testing.T) {
	code, stdout, stderr := runCLI("receive", "--help")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%s", code, stderr)
	}

	if !strings.Contains(stdout, "Usage: filepilot receive [session-id]") {
		t.Fatalf("expected canonical receive usage; help:\n%s", stdout)
	}
	if strings.Contains(stdout, "recv") {
		t.Fatalf("did not expect receive help to mention recv alias; help:\n%s", stdout)
	}
}

func TestMVPCommandsAcceptGlobalFlags(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.toml")
	backendPath := filepath.Join(root, "transfer-engine"+scriptExt())
	globalPayloadPath := filepath.Join(root, "payloads", "global.txt")
	if err := os.MkdirAll(filepath.Dir(globalPayloadPath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(globalPayloadPath, []byte("global payload"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	writeReceiveBackend(t, backendPath, filepath.Join(root, "global-backend-args.txt"), globalPayloadPath, 0)
	writeConfigFile(t, configPath, backendPath, filepath.Join(root, "downloads"))
	t.Setenv("FILEPILOT_CONFIG", configPath)
	t.Setenv("FILEPILOT_CACHE_DIR", filepath.Join(root, "cache"))
	t.Setenv("FILEPILOT_LOG_DIR", filepath.Join(root, "logs"))
	packSource := filepath.Join(root, "results")
	if err := os.MkdirAll(packSource, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	sendSource := filepath.Join(root, "sample.txt")
	if err := os.WriteFile(sendSource, []byte("sample"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	cases := [][]string{
		{"--json", "send", sendSource},
		{"--verbose", "receive", "FP-river-copper-lamp-7K2Q9M4XP8"},
		{"--quiet", "pack", packSource},
		{"doctor", "--json"},
		{"clean", "--verbose"},
		{"config", "show", "--quiet"},
		{"config", "set", "backend_path", "backend-bin", "--json"},
	}

	for _, args := range cases {
		code, _, stderr := runCLI(args...)
		if code != 0 {
			t.Fatalf("expected %q to accept global flags, got exit code %d; stderr=%s", strings.Join(args, " "), code, stderr)
		}
	}
}

func TestUnknownCommandReturnsArgumentError(t *testing.T) {
	code, stdout, stderr := runCLI("history")
	if code != 2 {
		t.Fatalf("expected exit code 2 for unknown command, got %d; stdout=%s stderr=%s", code, stdout, stderr)
	}
	if !strings.Contains(stderr, `Unknown command "history".`) {
		t.Fatalf("expected unknown command message; stderr=%s", stderr)
	}
}

func TestConfigShowDisplaysEffectiveConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("FILEPILOT_CONFIG", configPath)
	t.Setenv("FILEPILOT_CACHE_DIR", filepath.Join(t.TempDir(), "cache"))
	t.Setenv("FILEPILOT_LOG_DIR", filepath.Join(t.TempDir(), "logs"))

	code, stdout, stderr := runCLI("config", "show")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%s", code, stderr)
	}

	for _, want := range []string{"backend_path:", "download_dir:", "auto_unpack: true", "keep_packages: false", "json_output: false", "config_path:", "cache_dir:", "log_dir:"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected config show to contain %q; output:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "croc") {
		t.Fatalf("config show should use backend language, not croc-specific language; output:\n%s", stdout)
	}
}

func TestConfigSetUpdatesConfigToml(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("FILEPILOT_CONFIG", configPath)

	code, _, stderr := runCLI("config", "set", "backend_path", filepath.Join("tools", "backend-bin"))
	if code != 0 {
		t.Fatalf("expected config set exit code 0, got %d; stderr=%s", code, stderr)
	}

	code, stdout, stderr := runCLI("config", "show", "--json")
	if code != 0 {
		t.Fatalf("expected config show exit code 0, got %d; stderr=%s", code, stderr)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("config show --json did not return JSON: %v\n%s", err, stdout)
	}
	if payload["backend_path"] != filepath.Join("tools", "backend-bin") {
		t.Fatalf("backend_path mismatch in JSON output: %#v", payload)
	}
}

func TestConfigJsonOutputDefaultMakesShowMachineReadable(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("FILEPILOT_CONFIG", configPath)

	code, _, stderr := runCLI("config", "set", "json_output", "true")
	if code != 0 {
		t.Fatalf("expected config set exit code 0, got %d; stderr=%s", code, stderr)
	}

	code, stdout, stderr := runCLI("config", "show")
	if code != 0 {
		t.Fatalf("expected config show exit code 0, got %d; stderr=%s", code, stderr)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("json_output config should make config show JSON; err=%v output=%s", err, stdout)
	}
	if payload["json_output"] != true {
		t.Fatalf("json_output mismatch in payload: %#v", payload)
	}
}

func TestSendFileJSONUsesStableAgentShape(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.toml")
	backendPath := filepath.Join(root, "transfer-engine"+scriptExt())
	sourcePath := filepath.Join(root, "sample.txt")
	if err := os.WriteFile(sourcePath, []byte("payload"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	writeSendBackend(t, backendPath, filepath.Join(root, "backend-args.txt"), 0)
	writeConfigFile(t, configPath, backendPath, filepath.Join(root, "downloads"))
	t.Setenv("FILEPILOT_CONFIG", configPath)
	t.Setenv("FILEPILOT_CACHE_DIR", filepath.Join(root, "cache"))
	t.Setenv("FILEPILOT_LOG_DIR", filepath.Join(root, "logs"))

	code, stdout, stderr := runCLI("send", sourcePath, "--json")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%s", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("JSON mode should not write human text to stderr, got %q", stderr)
	}

	payload := decodeJSON(t, stdout)
	if payload["ok"] != true || payload["status"] != "completed" || payload["mode"] != "send" {
		t.Fatalf("missing stable success fields: %#v", payload)
	}
	if payload["payload_type"] != "file" || payload["input_path"] != sourcePath {
		t.Fatalf("missing send payload fields: %#v", payload)
	}
	if payload["session_id"] == "" || payload["session_id_redacted"] == "" {
		t.Fatalf("missing session fields: %#v", payload)
	}
	if _, ok := payload["error"]; !ok {
		t.Fatalf("JSON success response must include error field: %#v", payload)
	}
}

func TestSendFileInvokesBackendAndRecordsRedactedHistory(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.toml")
	logDir := filepath.Join(root, "logs")
	backendLog := filepath.Join(root, "backend-args.txt")
	backendPath := filepath.Join(root, "transfer-engine"+scriptExt())
	sourcePath := filepath.Join(root, "file.zip")
	if err := os.WriteFile(sourcePath, []byte("payload"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	writeSendBackend(t, backendPath, backendLog, 0)
	writeConfigFile(t, configPath, backendPath, filepath.Join(root, "downloads"))
	t.Setenv("FILEPILOT_CONFIG", configPath)
	t.Setenv("FILEPILOT_CACHE_DIR", filepath.Join(root, "cache"))
	t.Setenv("FILEPILOT_LOG_DIR", logDir)

	code, stdout, stderr := runCLI("send", sourcePath)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stdout=%s stderr=%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "filepilot receive ") {
		t.Fatalf("human output should show the FilePilot receive command; output:\n%s", stdout)
	}
	for _, forbidden := range []string{"--code", backendPath, "croc"} {
		if strings.Contains(stdout, forbidden) {
			t.Fatalf("human output exposed backend detail %q:\n%s", forbidden, stdout)
		}
	}

	sessionID := extractSessionID(t, stdout)
	rawBackendLog, err := os.ReadFile(backendLog)
	if err != nil {
		t.Fatalf("ReadFile backend log returned error: %v", err)
	}
	backendInvocation := string(rawBackendLog)
	wantInvocation := []string{"--ignore-stdin", "send", sourcePath}
	if runtime.GOOS == "windows" {
		wantInvocation = append(wantInvocation, "--code", sessionID)
	} else {
		wantInvocation = append(wantInvocation, "CROC_SECRET="+sessionID)
		if strings.Contains(backendInvocation, "--code") {
			t.Fatalf("Unix send should use CROC_SECRET instead of --code; got %q", backendInvocation)
		}
	}
	for _, want := range wantInvocation {
		if !strings.Contains(backendInvocation, want) {
			t.Fatalf("backend invocation missing %q; got %q", want, backendInvocation)
		}
	}

	historyPath := filepath.Join(logDir, "transfer-history.jsonl")
	rawHistory, err := os.ReadFile(historyPath)
	if err != nil {
		t.Fatalf("ReadFile history returned error: %v", err)
	}
	if strings.Contains(string(rawHistory), sessionID) {
		t.Fatalf("history leaked full session id:\n%s", rawHistory)
	}
	rows := readHistoryRowsForCLI(t, historyPath)
	if len(rows) != 1 {
		t.Fatalf("expected 1 history row, got %d", len(rows))
	}
	row := rows[0]
	if row["mode"] != "send" || row["status"] != "completed" || row["payload_type"] != "file" {
		t.Fatalf("history row mismatch: %#v", row)
	}
	if row["input_path"] != sourcePath || row["backend_source"] != "configured" {
		t.Fatalf("history path/backend mismatch: %#v", row)
	}
	if row["session_id_redacted"] == "" {
		t.Fatalf("history should include redacted session id: %#v", row)
	}
}

func TestSendDirectoryPackagesInvokesBackendAndCleansTemporaryPackage(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.toml")
	cacheDir := filepath.Join(root, "cache")
	logDir := filepath.Join(root, "logs")
	backendLog := filepath.Join(root, "backend-args.txt")
	backendPath := filepath.Join(root, "transfer-engine"+scriptExt())
	sourceDir := filepath.Join(root, "results")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "summary.txt"), []byte("done\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	writeSendBackend(t, backendPath, backendLog, 0)
	writeConfigFile(t, configPath, backendPath, filepath.Join(root, "downloads"))
	t.Setenv("FILEPILOT_CONFIG", configPath)
	t.Setenv("FILEPILOT_CACHE_DIR", cacheDir)
	t.Setenv("FILEPILOT_LOG_DIR", logDir)

	code, stdout, stderr := runCLI("send", sourceDir)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stdout=%s stderr=%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "filepilot receive ") {
		t.Fatalf("human output should show the FilePilot receive command; output:\n%s", stdout)
	}
	for _, forbidden := range []string{"--code", backendPath, "croc"} {
		if strings.Contains(stdout, forbidden) {
			t.Fatalf("human output exposed backend detail %q:\n%s", forbidden, stdout)
		}
	}

	rawBackendLog, err := os.ReadFile(backendLog)
	if err != nil {
		t.Fatalf("ReadFile backend log returned error: %v", err)
	}
	backendInvocation := string(rawBackendLog)
	packagePath := extractSentPath(t, backendInvocation)
	if filepath.Dir(packagePath) != cacheDir {
		t.Fatalf("package path = %q, want cache dir %q", packagePath, cacheDir)
	}
	if !strings.HasSuffix(packagePath, ".tar.gz") {
		t.Fatalf("directory send should pass a .tar.gz package to backend, got %q", packagePath)
	}
	if _, err := os.Stat(packagePath); !os.IsNotExist(err) {
		t.Fatalf("temporary package should be removed when keep_packages=false; stat err=%v", err)
	}

	rows := readHistoryRowsForCLI(t, filepath.Join(logDir, "transfer-history.jsonl"))
	if len(rows) != 1 {
		t.Fatalf("expected 1 history row, got %d", len(rows))
	}
	row := rows[0]
	if row["mode"] != "send" || row["status"] != "completed" || row["payload_type"] != "directory_package" {
		t.Fatalf("history row mismatch: %#v", row)
	}
	if row["input_path"] != sourceDir || row["package_path"] != packagePath || row["packed"] != true {
		t.Fatalf("history should record directory package metadata: %#v", row)
	}
}

func TestSendDirectoryJSONReportsPackageAndKeepsPackageWhenConfigured(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.toml")
	cacheDir := filepath.Join(root, "cache")
	backendLog := filepath.Join(root, "backend-args.txt")
	backendPath := filepath.Join(root, "transfer-engine"+scriptExt())
	sourceDir := filepath.Join(root, "outputs")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "metrics.json"), []byte(`{"ok":true}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	writeSendBackend(t, backendPath, backendLog, 0)
	writeConfigFileWithKeepPackages(t, configPath, backendPath, filepath.Join(root, "downloads"), true)
	t.Setenv("FILEPILOT_CONFIG", configPath)
	t.Setenv("FILEPILOT_CACHE_DIR", cacheDir)
	t.Setenv("FILEPILOT_LOG_DIR", filepath.Join(root, "logs"))

	code, stdout, stderr := runCLI("send", sourceDir, "--json")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stdout=%s stderr=%s", code, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("JSON mode should not write stderr, got %q", stderr)
	}

	payload := decodeJSON(t, stdout)
	if payload["ok"] != true || payload["status"] != "completed" || payload["mode"] != "send" {
		t.Fatalf("missing stable send JSON fields: %#v", payload)
	}
	if payload["payload_type"] != "directory_package" || payload["input_path"] != sourceDir {
		t.Fatalf("missing directory send JSON fields: %#v", payload)
	}
	packagePath, ok := payload["package_path"].(string)
	if !ok || packagePath == "" {
		t.Fatalf("package_path missing from directory send JSON: %#v", payload)
	}
	if filepath.Dir(packagePath) != cacheDir {
		t.Fatalf("package path = %q, want cache dir %q", packagePath, cacheDir)
	}
	if _, err := os.Stat(packagePath); err != nil {
		t.Fatalf("package should be retained when keep_packages=true: %v", err)
	}
	entries := readPackageEntries(t, packagePath)
	if _, ok := entries[".filepilot/manifest.json"]; !ok {
		t.Fatalf("retained package missing manifest; entries=%v", packageEntryNames(entries))
	}
	if string(entries["outputs/metrics.json"]) != `{"ok":true}` {
		t.Fatalf("retained package missing source content; entries=%v", packageEntryNames(entries))
	}

	rawBackendLog, err := os.ReadFile(backendLog)
	if err != nil {
		t.Fatalf("ReadFile backend log returned error: %v", err)
	}
	if sentPath := extractSentPath(t, string(rawBackendLog)); sentPath != packagePath {
		t.Fatalf("backend sent path = %q, want package path %q", sentPath, packagePath)
	}
}

func TestSendMissingFileReturnsStructuredPathError(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FILEPILOT_CONFIG", filepath.Join(root, "config.toml"))
	t.Setenv("FILEPILOT_CACHE_DIR", filepath.Join(root, "cache"))
	t.Setenv("FILEPILOT_LOG_DIR", filepath.Join(root, "logs"))

	code, stdout, stderr := runCLI("send", filepath.Join(root, "missing.zip"), "--json")
	if code != 3 {
		t.Fatalf("expected exit code 3, got %d; stdout=%s stderr=%s", code, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("JSON mode should not write stderr, got %q", stderr)
	}
	payload := decodeJSON(t, stdout)
	assertJSONError(t, payload, "send", "PATH_NOT_FOUND")
}

func TestSendBackendFailureRecordsRedactedFailedAttempt(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.toml")
	logDir := filepath.Join(root, "logs")
	backendPath := filepath.Join(root, "transfer-engine"+scriptExt())
	sourcePath := filepath.Join(root, "result.bin")
	if err := os.WriteFile(sourcePath, []byte("payload"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	writeSendBackend(t, backendPath, filepath.Join(root, "backend-args.txt"), 17)
	writeConfigFile(t, configPath, backendPath, filepath.Join(root, "downloads"))
	t.Setenv("FILEPILOT_CONFIG", configPath)
	t.Setenv("FILEPILOT_CACHE_DIR", filepath.Join(root, "cache"))
	t.Setenv("FILEPILOT_LOG_DIR", logDir)

	code, stdout, stderr := runCLI("send", sourcePath)
	if code != 6 {
		t.Fatalf("expected exit code 6, got %d; stdout=%s stderr=%s", code, stdout, stderr)
	}
	if strings.Contains(stderr, "--code") || strings.Contains(stderr, backendPath) {
		t.Fatalf("failure output exposed backend details; stderr=%s", stderr)
	}
	sessionID := extractSessionID(t, stdout)

	historyPath := filepath.Join(logDir, "transfer-history.jsonl")
	rawHistory, err := os.ReadFile(historyPath)
	if err != nil {
		t.Fatalf("ReadFile history returned error: %v", err)
	}
	if strings.Contains(string(rawHistory), sessionID) {
		t.Fatalf("history leaked full session id:\n%s", rawHistory)
	}
	rows := readHistoryRowsForCLI(t, historyPath)
	if len(rows) != 1 {
		t.Fatalf("expected 1 history row, got %d", len(rows))
	}
	row := rows[0]
	if row["status"] != "failed" || row["error_code"] != "TRANSFER_FAILED" {
		t.Fatalf("expected failed transfer row, got %#v", row)
	}
	if row["session_id_redacted"] == "" {
		t.Fatalf("history should include redacted session id: %#v", row)
	}
}

func TestSendTimeoutCancelsBackendAndRecordsLocalTimeout(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.toml")
	logDir := filepath.Join(root, "logs")
	backendLog := filepath.Join(root, "backend-args.txt")
	backendPath := filepath.Join(root, "transfer-engine"+scriptExt())
	sourcePath := filepath.Join(root, "result.bin")
	if err := os.WriteFile(sourcePath, []byte("payload"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	writeBlockingBackend(t, backendPath, backendLog, filepath.Join(root, "backend-finished.txt"))
	writeConfigFile(t, configPath, backendPath, filepath.Join(root, "downloads"))
	t.Setenv("FILEPILOT_CONFIG", configPath)
	t.Setenv("FILEPILOT_CACHE_DIR", filepath.Join(root, "cache"))
	t.Setenv("FILEPILOT_LOG_DIR", logDir)

	started := time.Now()
	code, stdout, stderr := runCLI("send", sourcePath, "--timeout", "75ms", "--json")
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("send timeout took too long: %s; stdout=%s stderr=%s", elapsed, stdout, stderr)
	}
	if code != 8 {
		t.Fatalf("expected exit code 8, got %d; stdout=%s stderr=%s", code, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("JSON timeout should not write stderr, got %q", stderr)
	}
	payload := decodeJSON(t, stdout)
	assertJSONError(t, payload, "send", "LOCAL_TIMEOUT")
	if _, err := os.Stat(backendLog); err != nil {
		t.Fatalf("backend should have been invoked before timeout: %v", err)
	}

	rows := readHistoryRowsForCLI(t, filepath.Join(logDir, "transfer-history.jsonl"))
	if len(rows) != 1 {
		t.Fatalf("expected 1 history row, got %d", len(rows))
	}
	row := rows[0]
	if row["mode"] != "send" || row["status"] != "timeout" || row["error_code"] != "LOCAL_TIMEOUT" {
		t.Fatalf("history should record local timeout: %#v", row)
	}
}

func TestSendCancellationStopsBackendAndRecordsCancelled(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.toml")
	logDir := filepath.Join(root, "logs")
	backendLog := filepath.Join(root, "backend-args.txt")
	backendPath := filepath.Join(root, "transfer-engine"+scriptExt())
	sourcePath := filepath.Join(root, "result.bin")
	if err := os.WriteFile(sourcePath, []byte("payload"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	writeBlockingBackend(t, backendPath, backendLog, filepath.Join(root, "backend-finished.txt"))
	writeConfigFile(t, configPath, backendPath, filepath.Join(root, "downloads"))
	t.Setenv("FILEPILOT_CONFIG", configPath)
	t.Setenv("FILEPILOT_CACHE_DIR", filepath.Join(root, "cache"))
	t.Setenv("FILEPILOT_LOG_DIR", logDir)

	ctx, cancel := context.WithCancel(context.Background())
	restore := overrideBaseContextForTest(func() (context.Context, context.CancelFunc) {
		return ctx, cancel
	})
	defer restore()
	go func() {
		for {
			if _, err := os.Stat(backendLog); err == nil {
				cancel()
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	code, stdout, stderr := runCLI("send", sourcePath, "--json")
	if code != 8 {
		t.Fatalf("expected exit code 8, got %d; stdout=%s stderr=%s", code, stdout, stderr)
	}
	payload := decodeJSON(t, stdout)
	assertJSONError(t, payload, "send", "CANCELLED")
	rows := readHistoryRowsForCLI(t, filepath.Join(logDir, "transfer-history.jsonl"))
	if len(rows) != 1 {
		t.Fatalf("expected 1 history row, got %d", len(rows))
	}
	row := rows[0]
	if row["mode"] != "send" || row["status"] != "cancelled" || row["error_code"] != "CANCELLED" {
		t.Fatalf("history should record cancellation: %#v", row)
	}
}

func TestReceiveTimeoutCancelsBackendAndRecordsLocalTimeout(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.toml")
	logDir := filepath.Join(root, "logs")
	backendPath := filepath.Join(root, "transfer-engine"+scriptExt())
	writeBlockingBackend(t, backendPath, filepath.Join(root, "backend-args.txt"), filepath.Join(root, "backend-finished.txt"))
	writeConfigFile(t, configPath, backendPath, filepath.Join(root, "downloads"))
	t.Setenv("FILEPILOT_CONFIG", configPath)
	t.Setenv("FILEPILOT_CACHE_DIR", filepath.Join(root, "cache"))
	t.Setenv("FILEPILOT_LOG_DIR", logDir)

	code, stdout, stderr := runCLI("receive", "FP-river-copper-lamp-7K2Q9M4XP8", "--timeout", "75ms", "--json")
	if code != 8 {
		t.Fatalf("expected exit code 8, got %d; stdout=%s stderr=%s", code, stdout, stderr)
	}
	payload := decodeJSON(t, stdout)
	assertJSONError(t, payload, "receive", "LOCAL_TIMEOUT")
	rows := readHistoryRowsForCLI(t, filepath.Join(logDir, "transfer-history.jsonl"))
	if len(rows) != 1 {
		t.Fatalf("expected 1 history row, got %d", len(rows))
	}
	row := rows[0]
	if row["mode"] != "receive" || row["status"] != "timeout" || row["error_code"] != "LOCAL_TIMEOUT" {
		t.Fatalf("history should record receive timeout: %#v", row)
	}
}

func TestReceiveFileInvokesBackendSavesPayloadAndRecordsRedactedHistory(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.toml")
	downloadDir := filepath.Join(root, "downloads")
	logDir := filepath.Join(root, "logs")
	backendLog := filepath.Join(root, "backend-args.txt")
	backendPath := filepath.Join(root, "transfer-engine"+scriptExt())
	payloadPath := filepath.Join(root, "payloads", "file.zip")
	if err := os.MkdirAll(filepath.Dir(payloadPath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(payloadPath, []byte("ordinary archive bytes"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	writeReceiveBackend(t, backendPath, backendLog, payloadPath, 0)
	writeConfigFile(t, configPath, backendPath, downloadDir)
	t.Setenv("FILEPILOT_CONFIG", configPath)
	t.Setenv("FILEPILOT_CACHE_DIR", filepath.Join(root, "cache"))
	t.Setenv("FILEPILOT_LOG_DIR", logDir)
	sessionID := "FP-river-copper-lamp-7K2Q9M4XP8"

	code, stdout, stderr := runCLI("receive", sessionID)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stdout=%s stderr=%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "FilePilot receive completed.") {
		t.Fatalf("expected human receive completion output, got:\n%s", stdout)
	}
	for _, forbidden := range []string{"--out", backendPath, "croc"} {
		if strings.Contains(stdout, forbidden) || strings.Contains(stderr, forbidden) {
			t.Fatalf("receive output exposed backend detail %q; stdout=%s stderr=%s", forbidden, stdout, stderr)
		}
	}

	receivedPath := filepath.Join(downloadDir, "file.zip")
	content, err := os.ReadFile(receivedPath)
	if err != nil {
		t.Fatalf("ReadFile received payload returned error: %v", err)
	}
	if string(content) != "ordinary archive bytes" {
		t.Fatalf("received file content mismatch: %q", content)
	}
	if _, err := os.Stat(filepath.Join(downloadDir, "file")); !os.IsNotExist(err) {
		t.Fatalf("ordinary archive should be saved unchanged, not unpacked; stat err=%v", err)
	}

	rawBackendLog, err := os.ReadFile(backendLog)
	if err != nil {
		t.Fatalf("ReadFile backend log returned error: %v", err)
	}
	backendInvocation := string(rawBackendLog)
	for _, want := range []string{"--ignore-stdin", "--yes", "--out", downloadDir, sessionID} {
		if !strings.Contains(backendInvocation, want) {
			t.Fatalf("backend invocation missing %q; got %q", want, backendInvocation)
		}
	}

	rawHistory, err := os.ReadFile(filepath.Join(logDir, "transfer-history.jsonl"))
	if err != nil {
		t.Fatalf("ReadFile history returned error: %v", err)
	}
	if strings.Contains(string(rawHistory), sessionID) {
		t.Fatalf("history leaked full session id:\n%s", rawHistory)
	}
	rows := readHistoryRowsForCLI(t, filepath.Join(logDir, "transfer-history.jsonl"))
	if len(rows) != 1 {
		t.Fatalf("expected 1 history row, got %d", len(rows))
	}
	row := rows[0]
	if row["mode"] != "receive" || row["status"] != "completed" || row["payload_type"] != "file" {
		t.Fatalf("history row mismatch: %#v", row)
	}
	if row["output_path"] != receivedPath || row["unpacked"] != false || row["session_id_redacted"] == "" {
		t.Fatalf("history should record received file metadata: %#v", row)
	}
}

func TestReceiveRejectsInvalidSessionIDBeforeBackend(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.toml")
	backendLog := filepath.Join(root, "backend-args.txt")
	backendPath := filepath.Join(root, "transfer-engine"+scriptExt())
	payloadPath := filepath.Join(root, "payload.txt")
	if err := os.WriteFile(payloadPath, []byte("payload"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	writeReceiveBackend(t, backendPath, backendLog, payloadPath, 0)
	writeConfigFile(t, configPath, backendPath, filepath.Join(root, "downloads"))
	t.Setenv("FILEPILOT_CONFIG", configPath)
	t.Setenv("FILEPILOT_CACHE_DIR", filepath.Join(root, "cache"))
	t.Setenv("FILEPILOT_LOG_DIR", filepath.Join(root, "logs"))

	code, stdout, stderr := runCLI("receive", "not-a-filepilot-session", "--json")
	if code != 7 {
		t.Fatalf("expected exit code 7, got %d; stdout=%s stderr=%s", code, stdout, stderr)
	}
	payload := decodeJSON(t, stdout)
	assertJSONError(t, payload, "receive", "INVALID_SESSION_ID")
	if _, err := os.Stat(backendLog); !os.IsNotExist(err) {
		t.Fatalf("backend should not be invoked for invalid session ID; stat err=%v", err)
	}
}

func TestReceiveUserSuppliedTarGzArchiveIsSavedUnchanged(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.toml")
	downloadDir := filepath.Join(root, "downloads")
	backendPath := filepath.Join(root, "transfer-engine"+scriptExt())
	payloadPath := filepath.Join(root, "payloads", "logs.tar.gz")
	if err := os.MkdirAll(filepath.Dir(payloadPath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(payloadPath, []byte("user archive bytes"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	writeReceiveBackend(t, backendPath, filepath.Join(root, "backend-args.txt"), payloadPath, 0)
	writeConfigFile(t, configPath, backendPath, downloadDir)
	t.Setenv("FILEPILOT_CONFIG", configPath)
	t.Setenv("FILEPILOT_CACHE_DIR", filepath.Join(root, "cache"))
	t.Setenv("FILEPILOT_LOG_DIR", filepath.Join(root, "logs"))

	code, stdout, stderr := runCLI("receive", "FP-river-copper-lamp-7K2Q9M4XP8", "--json")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stdout=%s stderr=%s", code, stdout, stderr)
	}
	payload := decodeJSON(t, stdout)
	if payload["payload_type"] != "file" || payload["unpacked"] != false {
		t.Fatalf("user archive should be reported as an ordinary file: %#v", payload)
	}
	receivedPath := filepath.Join(downloadDir, "logs.tar.gz")
	if string(mustReadFile(t, receivedPath)) != "user archive bytes" {
		t.Fatalf("user archive content changed")
	}
	if _, err := os.Stat(filepath.Join(downloadDir, "logs")); !os.IsNotExist(err) {
		t.Fatalf("user archive should not be unpacked; stat err=%v", err)
	}
}

func TestReceiveDirectoryPackageUnpacksToNonConflictingDestination(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.toml")
	downloadDir := filepath.Join(root, "downloads")
	logDir := filepath.Join(root, "logs")
	backendPath := filepath.Join(root, "transfer-engine"+scriptExt())
	sourceDir := filepath.Join(root, "results")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "summary.txt"), []byte("done\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	packageResult, err := packaging.CreateDirectoryPackage(packaging.Request{
		SourceDir:  sourceDir,
		OutputPath: filepath.Join(root, "payloads", "results.tar.gz"),
	})
	if err != nil {
		t.Fatalf("CreateDirectoryPackage returned error: %v", err)
	}
	existingDir := filepath.Join(downloadDir, "results")
	if err := os.MkdirAll(existingDir, 0o755); err != nil {
		t.Fatalf("MkdirAll existing dir returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(existingDir, "existing.txt"), []byte("keep\n"), 0o644); err != nil {
		t.Fatalf("WriteFile existing returned error: %v", err)
	}
	writeReceiveBackend(t, backendPath, filepath.Join(root, "backend-args.txt"), packageResult.PackagePath, 0)
	writeConfigFile(t, configPath, backendPath, downloadDir)
	t.Setenv("FILEPILOT_CONFIG", configPath)
	t.Setenv("FILEPILOT_CACHE_DIR", filepath.Join(root, "cache"))
	t.Setenv("FILEPILOT_LOG_DIR", logDir)
	sessionID := "FP-river-copper-lamp-7K2Q9M4XP8"

	code, stdout, stderr := runCLI("receive", sessionID)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stdout=%s stderr=%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "FilePilot receive completed.") {
		t.Fatalf("expected receive completion output, got:\n%s", stdout)
	}
	if string(mustReadFile(t, filepath.Join(existingDir, "existing.txt"))) != "keep\n" {
		t.Fatalf("existing destination was overwritten")
	}
	unpackedPath := filepath.Join(downloadDir, "results-1")
	if string(mustReadFile(t, filepath.Join(unpackedPath, "summary.txt"))) != "done\n" {
		t.Fatalf("directory package was not unpacked into non-conflicting destination")
	}
	if _, err := os.Stat(filepath.Join(downloadDir, "results.tar.gz")); err != nil {
		t.Fatalf("received package should remain available after unpack: %v", err)
	}

	rows := readHistoryRowsForCLI(t, filepath.Join(logDir, "transfer-history.jsonl"))
	if len(rows) != 1 {
		t.Fatalf("expected 1 history row, got %d", len(rows))
	}
	row := rows[0]
	if row["mode"] != "receive" || row["status"] != "completed" || row["payload_type"] != "directory_package" {
		t.Fatalf("history row mismatch: %#v", row)
	}
	if row["output_path"] != unpackedPath || row["package_path"] != filepath.Join(downloadDir, "results.tar.gz") || row["unpacked"] != true {
		t.Fatalf("history should record unpacked directory package metadata: %#v", row)
	}
}

func TestCleanDryRunListsOnlyFilePilotCacheFilesWithoutDeleting(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.toml")
	cacheDir := filepath.Join(root, "cache")
	downloadDir := filepath.Join(root, "downloads")
	logDir := filepath.Join(root, "logs")
	sourceDir := filepath.Join(root, "results")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "summary.txt"), []byte("done\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	packagePath := filepath.Join(cacheDir, "results-filepilot.tar.gz")
	if _, err := packaging.CreateDirectoryPackage(packaging.Request{SourceDir: sourceDir, OutputPath: packagePath}); err != nil {
		t.Fatalf("CreateDirectoryPackage returned error: %v", err)
	}
	ordinaryArchive := filepath.Join(cacheDir, "user-archive.tar.gz")
	if err := os.WriteFile(ordinaryArchive, []byte("not a FilePilot package"), 0o644); err != nil {
		t.Fatalf("WriteFile ordinary archive returned error: %v", err)
	}
	downloadFile := filepath.Join(downloadDir, "downloaded.txt")
	if err := os.MkdirAll(downloadDir, 0o755); err != nil {
		t.Fatalf("MkdirAll download returned error: %v", err)
	}
	if err := os.WriteFile(downloadFile, []byte("keep download"), 0o644); err != nil {
		t.Fatalf("WriteFile download returned error: %v", err)
	}
	historyPath := filepath.Join(logDir, "transfer-history.jsonl")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatalf("MkdirAll log returned error: %v", err)
	}
	if err := os.WriteFile(historyPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile history returned error: %v", err)
	}
	writeConfigFile(t, configPath, "", downloadDir)
	t.Setenv("FILEPILOT_CONFIG", configPath)
	t.Setenv("FILEPILOT_CACHE_DIR", cacheDir)
	t.Setenv("FILEPILOT_LOG_DIR", logDir)

	code, stdout, stderr := runCLI("clean", "--dry-run")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stdout=%s stderr=%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "Would remove 1 FilePilot cache file") || !strings.Contains(stdout, packagePath) {
		t.Fatalf("dry-run output should list planned FilePilot cache deletion; output:\n%s", stdout)
	}
	if strings.Contains(stdout, ordinaryArchive) || strings.Contains(stdout, downloadFile) || strings.Contains(stdout, historyPath) || strings.Contains(stdout, configPath) {
		t.Fatalf("dry-run output included non-cache or non-owned path:\n%s", stdout)
	}
	for _, path := range []string{packagePath, ordinaryArchive, downloadFile, historyPath, configPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("dry-run should not delete %s: %v", path, err)
		}
	}
}

func TestCleanDeletesOnlyFilePilotOwnedCacheFiles(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.toml")
	cacheDir := filepath.Join(root, "cache")
	downloadDir := filepath.Join(root, "downloads")
	logDir := filepath.Join(root, "logs")
	packagePath := createTestDirectoryPackage(t, root, filepath.Join(cacheDir, "owned.tar.gz"))
	ordinaryCacheFile := filepath.Join(cacheDir, "keep.txt")
	if err := os.WriteFile(ordinaryCacheFile, []byte("keep cache"), 0o644); err != nil {
		t.Fatalf("WriteFile ordinary cache returned error: %v", err)
	}
	downloadFile := filepath.Join(downloadDir, "downloaded.txt")
	if err := os.MkdirAll(downloadDir, 0o755); err != nil {
		t.Fatalf("MkdirAll download returned error: %v", err)
	}
	if err := os.WriteFile(downloadFile, []byte("keep download"), 0o644); err != nil {
		t.Fatalf("WriteFile download returned error: %v", err)
	}
	historyPath := filepath.Join(logDir, "transfer-history.jsonl")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatalf("MkdirAll log returned error: %v", err)
	}
	if err := os.WriteFile(historyPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile history returned error: %v", err)
	}
	writeConfigFile(t, configPath, "", downloadDir)
	t.Setenv("FILEPILOT_CONFIG", configPath)
	t.Setenv("FILEPILOT_CACHE_DIR", cacheDir)
	t.Setenv("FILEPILOT_LOG_DIR", logDir)

	code, stdout, stderr := runCLI("clean")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stdout=%s stderr=%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "Removed 1 FilePilot cache file") || !strings.Contains(stdout, packagePath) {
		t.Fatalf("clean output should report removed FilePilot cache file; output:\n%s", stdout)
	}
	if _, err := os.Stat(packagePath); !os.IsNotExist(err) {
		t.Fatalf("clean should delete FilePilot cache package; stat err=%v", err)
	}
	for _, path := range []string{ordinaryCacheFile, downloadFile, historyPath, configPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("clean should not delete %s: %v", path, err)
		}
	}
}

func TestCleanOlderThanFiltersByAgeAndReportsJSON(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.toml")
	cacheDir := filepath.Join(root, "cache")
	oldPackage := createTestDirectoryPackage(t, root, filepath.Join(cacheDir, "old.tar.gz"))
	newPackage := createTestDirectoryPackage(t, root, filepath.Join(cacheDir, "new.tar.gz"))
	oldTime := time.Now().Add(-3 * time.Hour)
	newTime := time.Now().Add(-10 * time.Minute)
	if err := os.Chtimes(oldPackage, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes old package returned error: %v", err)
	}
	if err := os.Chtimes(newPackage, newTime, newTime); err != nil {
		t.Fatalf("Chtimes new package returned error: %v", err)
	}
	writeConfigFile(t, configPath, "", filepath.Join(root, "downloads"))
	t.Setenv("FILEPILOT_CONFIG", configPath)
	t.Setenv("FILEPILOT_CACHE_DIR", cacheDir)
	t.Setenv("FILEPILOT_LOG_DIR", filepath.Join(root, "logs"))

	code, stdout, stderr := runCLI("clean", "--older-than", "1h", "--json")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stdout=%s stderr=%s", code, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("JSON clean should not write stderr, got %q", stderr)
	}
	payload := decodeJSON(t, stdout)
	if payload["ok"] != true || payload["status"] != "completed" || payload["mode"] != "clean" {
		t.Fatalf("missing clean JSON fields: %#v", payload)
	}
	removed, ok := payload["removed"].([]any)
	if !ok || len(removed) != 1 {
		t.Fatalf("expected one removed item in JSON, got %#v", payload)
	}
	removedItem, ok := removed[0].(map[string]any)
	if !ok || removedItem["path"] != oldPackage {
		t.Fatalf("removed item mismatch: %#v", payload)
	}
	if _, err := os.Stat(oldPackage); !os.IsNotExist(err) {
		t.Fatalf("old package should be deleted; stat err=%v", err)
	}
	if _, err := os.Stat(newPackage); err != nil {
		t.Fatalf("new package should remain because of older-than filter: %v", err)
	}
}

func TestReceiveJSONWithoutSessionIDReturnsMissingSessionIDWithoutBackend(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.toml")
	backendLog := filepath.Join(root, "backend-args.txt")
	backendPath := filepath.Join(root, "transfer-engine"+scriptExt())
	payloadPath := filepath.Join(root, "payload.txt")
	if err := os.WriteFile(payloadPath, []byte("payload"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	writeReceiveBackend(t, backendPath, backendLog, payloadPath, 0)
	writeConfigFile(t, configPath, backendPath, filepath.Join(root, "downloads"))
	t.Setenv("FILEPILOT_CONFIG", configPath)
	t.Setenv("FILEPILOT_CACHE_DIR", filepath.Join(root, "cache"))
	t.Setenv("FILEPILOT_LOG_DIR", filepath.Join(root, "logs"))

	code, stdout, stderr := runCLI("receive", "--json")
	if code != 7 {
		t.Fatalf("expected exit code 7, got %d; stdout=%s stderr=%s", code, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("JSON missing session should not write stderr, got %q", stderr)
	}
	payload := decodeJSON(t, stdout)
	assertJSONError(t, payload, "receive", "MISSING_SESSION_ID")
	if _, err := os.Stat(backendLog); !os.IsNotExist(err) {
		t.Fatalf("backend should not be invoked when JSON receive is missing a session ID; stat err=%v", err)
	}
}

func TestReceiveConfigJSONOutputWithoutSessionIDDoesNotPrompt(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.toml")
	backendLog := filepath.Join(root, "backend-args.txt")
	backendPath := filepath.Join(root, "transfer-engine"+scriptExt())
	payloadPath := filepath.Join(root, "payload.txt")
	if err := os.WriteFile(payloadPath, []byte("payload"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	writeReceiveBackend(t, backendPath, backendLog, payloadPath, 0)
	writeConfigFileWithOptions(t, configPath, backendPath, filepath.Join(root, "downloads"), false, true)
	t.Setenv("FILEPILOT_CONFIG", configPath)
	t.Setenv("FILEPILOT_CACHE_DIR", filepath.Join(root, "cache"))
	t.Setenv("FILEPILOT_LOG_DIR", filepath.Join(root, "logs"))

	code, stdout, stderr := runCLI("receive")
	if code != 7 {
		t.Fatalf("expected exit code 7, got %d; stdout=%s stderr=%s", code, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("config-driven JSON mode should not prompt or write stderr, got %q", stderr)
	}
	payload := decodeJSON(t, stdout)
	assertJSONError(t, payload, "receive", "MISSING_SESSION_ID")
	if _, err := os.Stat(backendLog); !os.IsNotExist(err) {
		t.Fatalf("backend should not be invoked when JSON receive is missing a session ID; stat err=%v", err)
	}
}

func TestReceiveHumanWithoutSessionIDPromptsAndUsesEnteredSessionID(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.toml")
	downloadDir := filepath.Join(root, "downloads")
	backendLog := filepath.Join(root, "backend-args.txt")
	backendPath := filepath.Join(root, "transfer-engine"+scriptExt())
	payloadPath := filepath.Join(root, "payloads", "prompted.txt")
	if err := os.MkdirAll(filepath.Dir(payloadPath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(payloadPath, []byte("prompted payload"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	writeReceiveBackend(t, backendPath, backendLog, payloadPath, 0)
	writeConfigFile(t, configPath, backendPath, downloadDir)
	t.Setenv("FILEPILOT_CONFIG", configPath)
	t.Setenv("FILEPILOT_CACHE_DIR", filepath.Join(root, "cache"))
	t.Setenv("FILEPILOT_LOG_DIR", filepath.Join(root, "logs"))
	sessionID := "FP-river-copper-lamp-7K2Q9M4XP8"
	withStdin(t, sessionID+"\n", func() {
		code, stdout, stderr := runCLI("receive")
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d; stdout=%s stderr=%s", code, stdout, stderr)
		}
		if !strings.Contains(stderr, "FilePilot Session ID:") {
			t.Fatalf("expected human receive prompt on stderr, got %q", stderr)
		}
		if !strings.Contains(stdout, "FilePilot receive completed.") {
			t.Fatalf("expected receive completion output, got %q", stdout)
		}
	})
	if string(mustReadFile(t, filepath.Join(downloadDir, "prompted.txt"))) != "prompted payload" {
		t.Fatalf("receive did not save payload after prompted session ID")
	}
	rawBackendLog := string(mustReadFile(t, backendLog))
	if !strings.Contains(rawBackendLog, sessionID) {
		t.Fatalf("backend invocation did not receive prompted session ID: %q", rawBackendLog)
	}
}

func TestJsonUnknownCommandReturnsStructuredError(t *testing.T) {
	code, stdout, stderr := runCLI("--json", "history")
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d; stdout=%s stderr=%s", code, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("JSON mode should not write human text to stderr, got %q", stderr)
	}

	payload := decodeJSON(t, stdout)
	assertJSONError(t, payload, "cli", "INVALID_ARGUMENT")
}

func TestJsonConfigSetUnsupportedKeyReturnsStructuredError(t *testing.T) {
	t.Setenv("FILEPILOT_CONFIG", filepath.Join(t.TempDir(), "config.toml"))

	code, stdout, stderr := runCLI("config", "set", "unknown_key", "value", "--json")
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d; stdout=%s stderr=%s", code, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("JSON mode should not write human text to stderr, got %q", stderr)
	}

	payload := decodeJSON(t, stdout)
	assertJSONError(t, payload, "config set", "INVALID_ARGUMENT")
}

func TestJsonInvalidConfigFileReturnsStructuredConfigError(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	t.Setenv("FILEPILOT_CONFIG", configPath)
	if err := os.WriteFile(configPath, []byte("not valid toml\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	code, stdout, stderr := runCLI("config", "show", "--json")
	if code != 10 {
		t.Fatalf("expected exit code 10, got %d; stdout=%s stderr=%s", code, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("JSON mode should not write human text to stderr, got %q", stderr)
	}

	payload := decodeJSON(t, stdout)
	assertJSONError(t, payload, "config show", "CONFIG_ERROR")
}

func decodeJSON(t *testing.T, stdout string) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout)
	}
	return payload
}

func assertJSONError(t *testing.T, payload map[string]any, mode string, code string) {
	t.Helper()
	if payload["ok"] != false || payload["status"] != "failed" || payload["mode"] != mode {
		t.Fatalf("missing stable failure fields: %#v", payload)
	}
	errorPayload, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("error field is not object: %#v", payload)
	}
	if errorPayload["code"] != code {
		t.Fatalf("error code = %#v, want %q; payload=%#v", errorPayload["code"], code, payload)
	}
	if errorPayload["message"] == "" {
		t.Fatalf("error message missing: %#v", payload)
	}
}

func TestPackCommandCreatesDirectoryPackageInCacheByDefault(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "results")
	cacheDir := filepath.Join(root, "cache")
	t.Setenv("FILEPILOT_CONFIG", filepath.Join(root, "config.toml"))
	t.Setenv("FILEPILOT_CACHE_DIR", cacheDir)
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "summary.txt"), []byte("done\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	code, stdout, stderr := runCLI("pack", source)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stdout=%s stderr=%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "Created FilePilot Directory Package:") {
		t.Fatalf("expected human pack output, got %q", stdout)
	}

	packagePath := strings.TrimSpace(strings.TrimPrefix(stdout, "Created FilePilot Directory Package:"))
	if filepath.Dir(packagePath) != cacheDir {
		t.Fatalf("package path = %q, want cache dir %q", packagePath, cacheDir)
	}
	entries := readPackageEntries(t, packagePath)
	if _, ok := entries[".filepilot/manifest.json"]; !ok {
		t.Fatalf("package missing manifest; entries=%v", packageEntryNames(entries))
	}
	if string(entries["results/summary.txt"]) != "done\n" {
		t.Fatalf("package missing source file; entries=%v", packageEntryNames(entries))
	}
}

func TestPackCommandSupportsOutputPathAndJSON(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "results")
	outputPath := filepath.Join(root, "out", "custom.tar.gz")
	t.Setenv("FILEPILOT_CONFIG", filepath.Join(root, "config.toml"))
	t.Setenv("FILEPILOT_CACHE_DIR", filepath.Join(root, "cache"))
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "summary.txt"), []byte("done\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	code, stdout, stderr := runCLI("pack", source, "--output", outputPath, "--json")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stdout=%s stderr=%s", code, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("JSON mode should not write stderr, got %q", stderr)
	}

	payload := decodeJSON(t, stdout)
	if payload["ok"] != true || payload["status"] != "completed" || payload["mode"] != "pack" {
		t.Fatalf("missing stable pack JSON fields: %#v", payload)
	}
	if payload["package_path"] != outputPath {
		t.Fatalf("package_path = %#v, want %q; payload=%#v", payload["package_path"], outputPath, payload)
	}
	if payload["payload_type"] != "directory_package" {
		t.Fatalf("payload_type mismatch: %#v", payload)
	}
	manifest, ok := payload["manifest"].(map[string]any)
	if !ok {
		t.Fatalf("manifest field missing or invalid: %#v", payload)
	}
	if manifest["schema_version"] != float64(1) || manifest["source_name"] != "results" || manifest["created_by"] != "filepilot" {
		t.Fatalf("manifest summary mismatch: %#v", manifest)
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("expected package at output path: %v", err)
	}
}

func TestDoctorReportsConfiguredBackendHuman(t *testing.T) {
	root := t.TempDir()
	backendPath := filepath.Join(root, "transfer-engine"+scriptExt())
	writeDoctorBackend(t, backendPath, "FilePilot test backend 1.0")
	t.Setenv("FILEPILOT_CONFIG", filepath.Join(root, "config.toml"))
	t.Setenv("FILEPILOT_CACHE_DIR", filepath.Join(root, "cache"))
	t.Setenv("FILEPILOT_LOG_DIR", filepath.Join(root, "logs"))
	writeConfigFile(t, filepath.Join(root, "config.toml"), backendPath, filepath.Join(root, "downloads"))

	code, stdout, stderr := runCLI("doctor")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stdout=%s stderr=%s", code, stdout, stderr)
	}
	for _, want := range []string{"Backend source: configured", "Backend version: FilePilot test backend 1.0", "Directory cache: writable", "Directory log: writable", "Directory download: writable"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected doctor output to contain %q; output:\n%s", want, stdout)
		}
	}
	if strings.Contains(strings.ToLower(stdout), "croc") {
		t.Fatalf("doctor output should use backend language, not croc-specific language:\n%s", stdout)
	}
}

func TestDoctorJsonMissingBackendReturnsStructuredFatal(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FILEPILOT_CONFIG", filepath.Join(root, "config.toml"))
	t.Setenv("FILEPILOT_CACHE_DIR", filepath.Join(root, "cache"))
	t.Setenv("FILEPILOT_LOG_DIR", filepath.Join(root, "logs"))
	t.Setenv("PATH", "")
	writeConfigFile(t, filepath.Join(root, "config.toml"), "", filepath.Join(root, "downloads"))

	code, stdout, stderr := runCLI("doctor", "--json")
	if code != 4 {
		t.Fatalf("expected exit code 4, got %d; stdout=%s stderr=%s", code, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("JSON mode should not write stderr, got %q", stderr)
	}

	payload := decodeJSON(t, stdout)
	assertJSONError(t, payload, "doctor", "BACKEND_NOT_FOUND")
	if _, ok := payload["warnings"]; !ok {
		t.Fatalf("doctor JSON should include warnings separately: %#v", payload)
	}
	if _, ok := payload["fatal"]; !ok {
		t.Fatalf("doctor JSON should include fatal separately: %#v", payload)
	}
}

func TestDoctorJsonProxyWarningDoesNotFail(t *testing.T) {
	root := t.TempDir()
	backendPath := filepath.Join(root, "transfer-engine"+scriptExt())
	writeDoctorBackend(t, backendPath, "FilePilot test backend 1.0")
	t.Setenv("FILEPILOT_CONFIG", filepath.Join(root, "config.toml"))
	t.Setenv("FILEPILOT_CACHE_DIR", filepath.Join(root, "cache"))
	t.Setenv("FILEPILOT_LOG_DIR", filepath.Join(root, "logs"))
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:7890")
	writeConfigFile(t, filepath.Join(root, "config.toml"), backendPath, filepath.Join(root, "downloads"))

	code, stdout, stderr := runCLI("doctor", "--json")
	if code != 0 {
		t.Fatalf("expected exit code 0 for warning-only doctor, got %d; stdout=%s stderr=%s", code, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("JSON mode should not write stderr, got %q", stderr)
	}

	payload := decodeJSON(t, stdout)
	if payload["ok"] != true || payload["status"] != "completed" || payload["mode"] != "doctor" {
		t.Fatalf("missing doctor success fields: %#v", payload)
	}
	warnings, ok := payload["warnings"].([]any)
	if !ok || len(warnings) == 0 {
		t.Fatalf("expected warning list in doctor JSON: %#v", payload)
	}
}

func readPackageEntries(t *testing.T, path string) map[string][]byte {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("NewReader returned error: %v", err)
	}
	defer gzipReader.Close()

	entries := map[string][]byte{}
	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar Next returned error: %v", err)
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		content, err := io.ReadAll(tarReader)
		if err != nil {
			t.Fatalf("ReadAll returned error: %v", err)
		}
		entries[header.Name] = content
	}
	return entries
}

func writeDoctorBackend(t *testing.T, path string, version string) {
	t.Helper()
	content := "@echo off\r\necho " + version + "\r\n"
	if filepath.Ext(path) != ".cmd" {
		content = "#!/bin/sh\necho '" + version + "'\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}

func writeSendBackend(t *testing.T, path string, logPath string, exitCode int) {
	t.Helper()
	var content string
	if runtime.GOOS == "windows" {
		content = strings.Join([]string{
			"@echo off",
			`if "%1"=="--version" echo FilePilot fake backend 1.0& exit /b 0`,
			`echo %*>>"` + logPath + `"`,
			fmt.Sprintf("exit /b %d", exitCode),
			"",
		}, "\r\n")
	} else {
		content = strings.Join([]string{
			"#!/bin/sh",
			`if [ "$1" = "--version" ]; then echo "FilePilot fake backend 1.0"; exit 0; fi`,
			`printf '%s\n' "$*" >> "` + logPath + `"`,
			`printf 'CROC_SECRET=%s\n' "$CROC_SECRET" >> "` + logPath + `"`,
			fmt.Sprintf("exit %d", exitCode),
			"",
		}, "\n")
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}

func writeReceiveBackend(t *testing.T, path string, logPath string, payloadPath string, exitCode int) {
	t.Helper()
	payloadName := filepath.Base(payloadPath)
	var content string
	if runtime.GOOS == "windows" {
		content = strings.Join([]string{
			"@echo off",
			`if "%1"=="--version" echo FilePilot fake backend 1.0& exit /b 0`,
			`echo %*>>"` + logPath + `"`,
			`if not "%2"=="--yes" exit /b ` + fmt.Sprintf("%d", exitCode),
			`if not exist "%4" mkdir "%4"`,
			`copy /Y "` + payloadPath + `" "%4\` + payloadName + `" >nul`,
			fmt.Sprintf("exit /b %d", exitCode),
			"",
		}, "\r\n")
	} else {
		content = strings.Join([]string{
			"#!/bin/sh",
			`if [ "$1" = "--version" ]; then echo "FilePilot fake backend 1.0"; exit 0; fi`,
			`printf '%s\n' "$*" >> "` + logPath + `"`,
			`if [ "$2" != "--yes" ]; then exit ` + fmt.Sprintf("%d", exitCode) + `; fi`,
			`mkdir -p "$4"`,
			`cp "` + payloadPath + `" "$4/` + payloadName + `"`,
			fmt.Sprintf("exit %d", exitCode),
			"",
		}, "\n")
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}

func writeBlockingBackend(t *testing.T, path string, logPath string, finishedPath string) {
	t.Helper()
	var content string
	if runtime.GOOS == "windows" {
		content = strings.Join([]string{
			"@echo off",
			`if "%1"=="--version" echo FilePilot fake backend 1.0& exit /b 0`,
			`echo %*>>"` + logPath + `"`,
			`ping -n 6 127.0.0.1 >nul`,
			`echo finished>"` + finishedPath + `"`,
			"exit /b 0",
			"",
		}, "\r\n")
	} else {
		content = strings.Join([]string{
			"#!/bin/sh",
			`if [ "$1" = "--version" ]; then echo "FilePilot fake backend 1.0"; exit 0; fi`,
			`printf '%s\n' "$*" >> "` + logPath + `"`,
			"sleep 5",
			`printf 'finished\n' > "` + finishedPath + `"`,
			"exit 0",
			"",
		}, "\n")
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}

func writeConfigFile(t *testing.T, path string, backendPath string, downloadDir string) {
	t.Helper()
	writeConfigFileWithKeepPackages(t, path, backendPath, downloadDir, false)
}

func writeConfigFileWithKeepPackages(t *testing.T, path string, backendPath string, downloadDir string, keepPackages bool) {
	t.Helper()
	writeConfigFileWithOptions(t, path, backendPath, downloadDir, keepPackages, false)
}

func writeConfigFileWithOptions(t *testing.T, path string, backendPath string, downloadDir string, keepPackages bool, jsonOutput bool) {
	t.Helper()
	content := strings.Join([]string{
		`backend_path = "` + strings.ReplaceAll(backendPath, `\`, `\\`) + `"`,
		`download_dir = "` + strings.ReplaceAll(downloadDir, `\`, `\\`) + `"`,
		"auto_unpack = true",
		fmt.Sprintf("keep_packages = %t", keepPackages),
		fmt.Sprintf("json_output = %t", jsonOutput),
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}

func packageEntryNames(entries map[string][]byte) []string {
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	return names
}

func scriptExt() string {
	if runtime.GOOS == "windows" {
		return ".cmd"
	}
	return ""
}

func extractSessionID(t *testing.T, output string) string {
	t.Helper()
	matches := regexp.MustCompile(`filepilot receive (FP-[^\s]+)`).FindStringSubmatch(output)
	if len(matches) != 2 {
		t.Fatalf("could not find FilePilot Session ID in output:\n%s", output)
	}
	return matches[1]
}

func extractSentPath(t *testing.T, backendInvocation string) string {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(backendInvocation), "\n")
	fields := strings.Fields(strings.TrimSpace(lines[0]))
	if len(fields) == 0 {
		t.Fatalf("backend invocation was empty")
	}
	return fields[len(fields)-1]
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) returned error: %v", path, err)
	}
	return content
}

func createTestDirectoryPackage(t *testing.T, root string, outputPath string) string {
	t.Helper()
	sourceDir := filepath.Join(root, "package-source-"+filepath.Base(outputPath))
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("MkdirAll source returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "summary.txt"), []byte("done\n"), 0o644); err != nil {
		t.Fatalf("WriteFile source returned error: %v", err)
	}
	result, err := packaging.CreateDirectoryPackage(packaging.Request{
		SourceDir:  sourceDir,
		OutputPath: outputPath,
	})
	if err != nil {
		t.Fatalf("CreateDirectoryPackage returned error: %v", err)
	}
	return result.PackagePath
}

func withStdin(t *testing.T, input string, run func()) {
	t.Helper()
	original := os.Stdin
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe returned error: %v", err)
	}
	if _, err := writer.WriteString(input); err != nil {
		t.Fatalf("WriteString returned error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close writer returned error: %v", err)
	}
	os.Stdin = reader
	defer func() {
		os.Stdin = original
		_ = reader.Close()
	}()
	run()
}

func overrideBaseContextForTest(fn func() (context.Context, context.CancelFunc)) func() {
	original := newBaseContext
	newBaseContext = fn
	return func() {
		newBaseContext = original
	}
}

func readHistoryRowsForCLI(t *testing.T, path string) []map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	rows := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		var row map[string]any
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatalf("history row is not JSON: %v\n%s", err, line)
		}
		rows = append(rows, row)
	}
	return rows
}
