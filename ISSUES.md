# FilePilot MVP Issue Drafts

These are local issue drafts because no issue tracker is configured in this workspace. They are ordered roughly by dependency and can be copied to GitHub, GitLab, or another tracker.

## Proposed Breakdown

1. Bootstrap FilePilot CLI with JSON-safe command plumbing
2. Add platform paths and TOML config
3. Implement FilePilot error model and Agent JSON output
4. Implement Directory Package creation with manifest
5. Implement transfer history with session redaction
6. Resolve bundled/configured/PATH backend and doctor checks
7. Implement session ID generation and redaction
8. Implement file send path through CrocBackend
9. Implement directory send path through CrocBackend
10. Implement receive path with safe unpacking
11. Implement non-interactive JSON receive errors
12. Implement cache cleaning with dry-run safeguards
13. Add local timeout, cancellation, and failure recording
14. Package bundled backend layout and release smoke tests

---

## Issue 1: Bootstrap FilePilot CLI with JSON-safe command plumbing

**Type**: AFK

**Blocked by**: None - can start immediately

**User stories covered**: All command-line user stories

### What to build

Create the initial Go CLI application with the Public MVP command surface. The CLI should expose `send`, `receive`, `pack`, `doctor`, `clean`, `config show`, and `config set`.

Global options should include `--json`, `--verbose`, and `--quiet`. Commands may return placeholder behavior where implementation is not ready, but the command names and routing must be stable.

### Acceptance criteria

- [ ] `filepilot --help` lists the MVP commands.
- [ ] `filepilot receive --help` uses `receive` as the canonical command.
- [ ] Unsupported commands such as `history`, `serve`, `daemon`, `gui`, `sync`, `resume`, and `login` are not exposed.
- [ ] Global flags are accepted consistently by MVP commands.
- [ ] Basic CLI tests cover command registration and unknown command behavior.

---

## Issue 2: Add platform paths and TOML config

**Type**: AFK

**Blocked by**: Issue 1

**User stories covered**: Configurable backend path, download directory, cache, and Agent default JSON mode

### What to build

Implement platform-specific config, cache, log, and download path resolution. Add `config.toml` loading and writing for MVP fields.

Environment variables should override config, cache, and log locations where specified by the PRD.

### Acceptance criteria

- [ ] Linux, macOS, and Windows path rules are represented in code.
- [ ] `filepilot config show` displays effective config without leaking sensitive values.
- [ ] `filepilot config set <key> <value>` updates supported MVP keys.
- [ ] `FILEPILOT_CONFIG`, `FILEPILOT_CACHE_DIR`, and `FILEPILOT_LOG_DIR` override defaults.
- [ ] Tests cover default path resolution and config set/show behavior.

---

## Issue 3: Implement FilePilot error model and Agent JSON output

**Type**: AFK

**Blocked by**: Issue 1

**User stories covered**: Agent automation, stable error handling

### What to build

Implement stable exit codes, machine error codes, and JSON response helpers. JSON mode must write only structured JSON to stdout and must not include human-readable progress text.

All command implementations should be able to return a FilePilot error with code, message, hint, and exit code.

### Acceptance criteria

- [ ] JSON responses include stable `ok`, `status`, `mode`, and `error` fields.
- [ ] Failures include stable `error.code`, `error.message`, and optional `error.hint`.
- [ ] Exit codes match the PRD table.
- [ ] Human output and JSON output are separated.
- [ ] Tests cover at least one success response and three error responses.

---

## Issue 4: Implement Directory Package creation with manifest

**Type**: AFK

**Blocked by**: Issue 2, Issue 3

**User stories covered**: Send a directory, package outputs for later transfer

### What to build

Implement `filepilot pack <dir>` as a public auxiliary command. It should create a `.tar.gz` Directory Package in the FilePilot cache by default and include `.filepilot/manifest.json`.

The same packaging function should be reusable by `send <dir>`.

### Acceptance criteria

- [ ] `filepilot pack <dir>` creates a `.tar.gz` package.
- [ ] The package contains `.filepilot/manifest.json`.
- [ ] Manifest includes schema version, payload type, source name, creator, and creation time.
- [ ] `--output <path>` writes to the requested destination.
- [ ] `--json` returns package path, payload type, and manifest summary.
- [ ] Tests verify package contents and cache default behavior.

---

## Issue 5: Implement transfer history with session redaction

**Type**: AFK

**Blocked by**: Issue 2, Issue 3

**User stories covered**: Debug transfer attempts, avoid leaking sensitive session codes

### What to build

Implement JSONL transfer history for local send and receive attempts. Each attempt should record success, failure, or cancellation with redacted session IDs.

History must not store full FilePilot Session IDs, backend raw commands, backend credentials, or file contents.

### Acceptance criteria

- [ ] A history writer appends one JSONL row per transfer attempt.
- [ ] Rows include mode, status, payload type, backend source, duration, paths, and error information.
- [ ] Session IDs are redacted before persistence.
- [ ] Full session IDs and backend raw commands are not written.
- [ ] Tests cover redaction and failed attempt recording.

---

## Issue 6: Resolve bundled/configured/PATH backend and doctor checks

**Type**: AFK

**Blocked by**: Issue 2, Issue 3

**User stories covered**: Use FilePilot without manually installing croc, diagnose local blockers

### What to build

Implement backend resolution in the PRD order: configured `backend_path`, bundled backend, system PATH fallback, then `BACKEND_NOT_FOUND`.

Implement `filepilot doctor` as local diagnostics. It should report backend source, backend version, config status, writable directories, proxy variables, and suspected Fake-IP warnings.

### Acceptance criteria

- [ ] Backend resolution follows the configured, bundled, PATH, missing order.
- [ ] Doctor returns fatal error when no usable backend exists.
- [ ] Doctor warning-only results exit with code 0.
- [ ] Doctor reports backend source and version when available.
- [ ] JSON doctor output separates warnings from fatal errors.
- [ ] Tests cover each backend resolution branch.

---

## Issue 7: Implement session ID generation and redaction

**Type**: AFK

**Blocked by**: Issue 3, Issue 5

**User stories covered**: Share a FilePilot Session ID safely, avoid low-entropy codes

### What to build

Generate FilePilot Session IDs suitable for use as backend passphrases. The format should be readable, prefixed with `FP-`, and provide sufficient entropy.

Add validation and redaction helpers used by logs, JSON responses, and history.

### Acceptance criteria

- [ ] Generated IDs use the agreed `FP-...` shape.
- [ ] Generated IDs meet the entropy requirement defined by implementation notes or tests.
- [ ] Validation rejects malformed or unsafe IDs.
- [ ] Redaction hides the sensitive middle portion.
- [ ] Tests cover generation, validation, and redaction.

---

## Issue 8: Implement file send path through CrocBackend

**Type**: AFK

**Blocked by**: Issue 3, Issue 5, Issue 6, Issue 7

**User stories covered**: Send one file using only FilePilot commands

### What to build

Implement `filepilot send <file>` for File Payloads. FilePilot should generate a FilePilot Session ID, invoke CrocBackend with that ID as the controlled passphrase, display the FilePilot receive command, and block until completion or failure.

The implementation should record a transfer attempt and avoid exposing backend raw commands.

### Acceptance criteria

- [ ] `send <file>` validates existence and permissions.
- [ ] FilePilot invokes the selected backend through the `TransferBackend` abstraction.
- [ ] Human output shows `filepilot receive <session-id>`.
- [ ] Backend raw commands are not shown in normal output.
- [ ] Success and failure attempts are recorded with redacted session IDs.
- [ ] Tests use a fake backend to verify send lifecycle behavior.

---

## Issue 9: Implement directory send path through CrocBackend

**Type**: AFK

**Blocked by**: Issue 4, Issue 8

**User stories covered**: Send a directory with automatic packaging

### What to build

Implement `filepilot send <dir>` by creating a Directory Package, sending that package through CrocBackend, and cleaning or retaining the package according to config.

The directory path should share the same send lifecycle and JSON response shape as file send.

### Acceptance criteria

- [ ] `send <dir>` creates a Directory Package using the shared packer.
- [ ] The package is sent through the backend abstraction.
- [ ] Temporary package cleanup respects `keep_packages`.
- [ ] Human output still shows only FilePilot commands and states.
- [ ] JSON output reports `payload_type: directory_package`.
- [ ] Tests cover successful packaging, send call, and cleanup behavior.

---

## Issue 10: Implement receive path with safe unpacking

**Type**: AFK

**Blocked by**: Issue 4, Issue 6, Issue 7

**User stories covered**: Receive files unchanged, receive directories unpacked safely

### What to build

Implement `filepilot receive <session-id>` through CrocBackend. Received File Payloads should be saved as-is. Received Directory Packages should be identified by manifest and unpacked when `auto_unpack` is enabled.

Unpacking must avoid overwriting existing directories.

### Acceptance criteria

- [ ] `receive <session-id>` validates the session ID and invokes the backend abstraction.
- [ ] User-supplied archives are saved unchanged.
- [ ] FilePilot Directory Packages are detected by manifest.
- [ ] Directory Packages unpack into a non-conflicting destination.
- [ ] Receive attempts are recorded with redacted session IDs.
- [ ] Tests cover ordinary archive files and FilePilot Directory Packages.

---

## Issue 11: Implement non-interactive JSON receive errors

**Type**: AFK

**Blocked by**: Issue 3, Issue 10

**User stories covered**: Agent-safe receive behavior

### What to build

Finalize receive behavior when no session ID is provided. Human mode may prompt for input. JSON mode must never prompt and must return `MISSING_SESSION_ID`.

This issue should also verify that all JSON-mode commands avoid interactive prompts.

### Acceptance criteria

- [ ] `filepilot receive` prompts for a session ID in human mode.
- [ ] `filepilot receive --json` returns `MISSING_SESSION_ID`.
- [ ] JSON mode writes only JSON to stdout.
- [ ] JSON mode does not block waiting for stdin.
- [ ] Tests cover both human-mode prompt routing and JSON-mode failure.

---

## Issue 12: Implement cache cleaning with dry-run safeguards

**Type**: AFK

**Blocked by**: Issue 2, Issue 4

**User stories covered**: Clean FilePilot cache safely

### What to build

Implement `filepilot clean`, `filepilot clean --dry-run`, and `filepilot clean --older-than <duration>`.

The command must only remove FilePilot-owned temporary files under FilePilot cache locations. It must not remove downloads, source files, arbitrary archives, transfer history, or config.

### Acceptance criteria

- [ ] `clean --dry-run` lists planned deletions without deleting files.
- [ ] `clean` deletes only FilePilot-owned cache files.
- [ ] `--older-than` filters by age.
- [ ] Deletion is refused for paths outside the FilePilot cache root.
- [ ] Tests cover dry-run, deletion, age filtering, and outside-root refusal.

---

## Issue 13: Add local timeout, cancellation, and failure recording

**Type**: AFK

**Blocked by**: Issue 8, Issue 9, Issue 10

**User stories covered**: Cancel blocked transfers, record failed attempts

### What to build

Add sender-side local timeout and cancellation handling for send and receive processes. FilePilot should terminate the backend process when the local timeout expires or the user cancels.

Timeout and cancellation should map to stable statuses and be recorded in transfer history.

### Acceptance criteria

- [ ] `send --timeout <duration>` cancels a transfer after the duration.
- [ ] Ctrl+C or equivalent cancellation stops the backend process.
- [ ] Timeout maps to `LOCAL_TIMEOUT`.
- [ ] User cancellation maps to `CANCELLED`.
- [ ] History records timeout and cancellation attempts.
- [ ] Tests use a fake backend process to verify cancellation.

---

## Issue 14: Package bundled backend layout and release smoke tests

**Type**: HITL

**Blocked by**: Issue 6, Issue 8, Issue 10

**User stories covered**: Use FilePilot without manually installing croc

### What to build

Define and implement the release layout for shipping FilePilot with a bundled croc-compatible backend on Linux, macOS, and Windows.

This issue is marked HITL because the exact backend binary source, licensing review, checksum policy, and release artifact layout require human confirmation before publication.

### Acceptance criteria

- [ ] Release layout defines where bundled backend binaries live on each platform.
- [ ] `doctor` can find the bundled backend in release layout.
- [ ] Smoke tests run without system croc installed.
- [ ] Backend binary source and license are documented.
- [ ] Checksums or equivalent integrity metadata are documented.
- [ ] Human review approves the release packaging policy.
