# FilePilot PRD

## 1. Summary

FilePilot is a cross-platform CLI for developer-oriented file transfer. It wraps a bundled croc-compatible backend behind a stable FilePilot workflow for sending files, packaging directories, receiving payloads, recording transfer attempts, and returning Agent-friendly JSON.

The Public MVP focuses on one reliable end-to-end path: a sender runs `filepilot send <path>`, a receiver runs `filepilot receive <session-id>`, and neither side needs to invoke croc directly.

## 2. Problem

Developers often need to move experiment outputs, logs, model artifacts, and data slices from Linux servers or edge devices to a local machine. Direct SSH, inbound ports, file shares, and cloud drives are often unavailable or inconvenient.

Existing relay-assisted tools solve the transport problem, but they expose backend-specific commands and code phrases. That makes the flow harder to automate, harder to explain, and less suitable for Agent workflows.

## 3. Goals

- Provide a simple two-command transfer flow for files and directories.
- Ship with a supported backend so ordinary users do not install croc manually.
- Hide backend commands and backend-specific terminology from normal users.
- Generate FilePilot Session IDs that can be shared with receivers.
- Automatically package directories and safely unpack only FilePilot-created packages.
- Provide stable JSON output for Agent automation.
- Record transfer attempts with sensitive session codes redacted.
- Keep the backend replaceable through a `TransferBackend` abstraction.

## 4. Non-Goals

- No official hosted FilePilot rendezvous or relay service in the MVP.
- No requirement for users to deploy a separate rendezvous or relay server.
- No runtime backend downloader or system package installer.
- No PATH modification, package manager invocation, or automatic croc installation.
- No independent encryption protocol, account system, device pairing, or permission system.
- No GUI, daemon, sync, resume, login, or history command in the MVP.
- No commitment to exact progress percentages or real-time transfer speed.

## 5. Target Users

Primary users are developers transferring results from lab servers to personal machines. They need commands that work from terminals and do not assume direct IP reachability.

Secondary users are Agent workflows that need to package and send task outputs without parsing human-readable text.

Edge-device users are also in scope when both endpoints can access the selected backend rendezvous and relay.

## 6. Core User Stories

As a developer, I can send one file from a Linux server and receive it on Windows using only FilePilot commands.

As a developer, I can send a directory and have FilePilot package it, transfer it, and unpack it safely on receive.

As an Agent, I can run FilePilot with `--json` and receive stable fields and stable error codes.

As a user, I can run `filepilot doctor` to understand local blockers before trying a transfer.

As a user, I can clean FilePilot cache without risking source files, downloads, or transfer history.

## 7. MVP Commands

The Public MVP includes:

```bash
filepilot send <path>
filepilot receive [session-id]
filepilot pack <dir>
filepilot doctor
filepilot clean
filepilot config show
filepilot config set <key> <value>
```

The canonical receive command is `receive`. Short aliases such as `recv` are not part of the MVP requirement.

All core commands support `--json`, `--verbose`, and `--quiet` where applicable. JSON mode must not prompt for input.

## 8. Send Flow

`filepilot send <path>` validates the path, resolves a backend, creates a FilePilot Session ID, and starts a blocking transfer.

If the path is a file, FilePilot sends it as a File Payload. Archive-looking files such as `.zip` or `.tar.gz` remain ordinary files.

If the path is a directory, FilePilot creates a Directory Package in the cache, includes a FilePilot manifest, and sends the package.

The sender process stays alive until the receiver joins and the transfer completes, fails, is cancelled, or reaches a local timeout.

## 9. Receive Flow

`filepilot receive <session-id>` validates the session ID, resolves a backend, and receives the payload into the configured download directory.

If the received payload is a FilePilot Directory Package, FilePilot may unpack it according to configuration. If the payload is a File Payload, FilePilot saves it as-is.

`filepilot receive` without a session ID may prompt a human user. `filepilot receive --json` without a session ID must return a structured error.

## 10. Packaging Rules

`filepilot pack <dir>` creates a Directory Package without starting a transfer.

Directory Packages are `.tar.gz` archives that contain `.filepilot/manifest.json`. The manifest identifies the payload as a FilePilot-created directory package.

Generated packages default to the FilePilot cache directory. `--output <path>` may override the destination.

## 11. Backend Rules

The MVP implements only `CrocBackend`, but code must depend on the `TransferBackend` interface.

Backend resolution order is:

1. `backend_path` from config.
2. Bundled backend binary from the FilePilot installation.
3. Compatible backend from system `PATH`.
4. `BACKEND_NOT_FOUND` if none is usable.

FilePilot does not download backend binaries at runtime and does not call platform package managers.

## 12. Security Boundary

FilePilot does not implement a stronger independent security model than the configured backend.

The FilePilot Session ID is a visible one-time session code. Users may copy it through a trusted channel, but FilePilot treats it as sensitive while valid.

Persistent logs, crash reports, and transfer history must redact the full session ID. Backend raw commands and backend credentials must not be recorded.

## 13. Agent API

`--json` is a stable Agent API.

Successful output includes stable fields such as `ok`, `status`, `mode`, `session_id`, `session_id_redacted`, `payload_type`, `backend`, and `error`.

Failed output includes `ok: false`, `status: failed`, and an `error` object with stable `code`, human-readable `message`, and optional `hint`.

## 14. Diagnostics

`filepilot doctor` performs local diagnostics only. It does not guarantee that a future transfer will succeed end to end.

Fatal failures include a missing backend, an unusable backend, unwritable required directories, or invalid config.

Warnings include proxy variables and suspected Fake-IP or TUN behavior. Warnings do not produce a non-zero exit code by themselves.

## 15. Config

FilePilot uses platform-specific config, cache, and log locations.

The config file is `config.toml`. MVP fields include:

```toml
backend_path = ""
download_dir = ""
auto_unpack = true
keep_packages = false
json_output = false
```

Environment overrides:

```text
FILEPILOT_CONFIG
FILEPILOT_CACHE_DIR
FILEPILOT_LOG_DIR
```

## 16. Transfer History

Transfer history is JSONL.

Each send or receive attempt is recorded, including success, failure, and cancellation.

History records include redacted session IDs, status, mode, payload type, paths, backend source, duration, and error information.

History records exclude full session IDs, backend raw commands, backend credentials, and file contents.

## 17. Acceptance Criteria

File transfer works with `filepilot send ./file.zip` and `filepilot receive <session-id>`. The receiver gets `file.zip` unchanged.

Directory transfer works with `filepilot send ./results` and `filepilot receive <session-id>`. The receiver gets an unpacked directory when the manifest identifies a Directory Package.

`filepilot receive --json` without a session ID returns `MISSING_SESSION_ID` and does not prompt.

`filepilot doctor` reports backend source and local directory readiness. Proxy warnings do not fail the command.

On a machine without system croc, FilePilot can still find the bundled backend.

`filepilot clean` only removes FilePilot-owned cache files and supports `--dry-run`.

## 18. Future Work

Future work may include JSON event streaming, exact progress, background transfers, a history command, GUI, self-hosted services, native backend, LAN-first routing, resume, incremental sync, and richer Agent workflows.
