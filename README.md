# FilePilot

FilePilot is a cross-platform CLI for moving files and directories between developer machines, lab servers, edge devices, and Agent workflows.

It provides a simple FilePilot command surface over a bundled croc-compatible transfer backend. Users do not need to call croc directly or install croc manually for the default workflow.

## Status

This repository is currently at the requirements and planning stage.

The Public MVP is defined in [PRD.md](./PRD.md). The implementation decisions are recorded in [docs/adr](./docs/adr).

## MVP Experience

Sender:

```bash
filepilot send ./results
```

Receiver:

```bash
filepilot receive FP-river-copper-lamp-7K2Q
```

FilePilot generates a one-time FilePilot Session ID and passes it to the backend as a controlled passphrase. Users share the FilePilot Session ID through a trusted channel.

## What FilePilot Does

- Sends files and directories across machines without requiring direct IP reachability.
- Packages directories automatically before transfer.
- Unpacks only FilePilot-created directory packages.
- Uses a bundled croc-compatible backend by default.
- Provides stable JSON output for Agents.
- Records transfer attempts with session IDs redacted.
- Runs local diagnostics with `filepilot doctor`.

## What FilePilot Does Not Do in the MVP

- It does not provide an official hosted rendezvous or relay service.
- It does not require users to deploy a separate server.
- It does not download backend binaries at runtime.
- It does not install packages through apt, brew, winget, scoop, or choco.
- It does not implement an independent encryption protocol.
- It does not include GUI, daemon, sync, resume, login, or history commands.

## MVP Commands

```bash
filepilot send <path>
filepilot receive [session-id]
filepilot pack <dir>
filepilot doctor
filepilot clean
filepilot config show
filepilot config set <key> <value>
```

The canonical receive command is `receive`.

## Agent Usage

Use `--json` when FilePilot is called from automation.

```bash
filepilot send ./outputs --json
```

JSON mode is non-interactive. For example, this must return a structured error rather than prompt:

```bash
filepilot receive --json
```

## Directory Packages

When sending a directory, FilePilot creates a `.tar.gz` package with a `.filepilot/manifest.json` file.

Only FilePilot-created directory packages are auto-unpacked on receive. A user-supplied archive such as `file.zip` or `logs.tar.gz` is treated as a normal file.

## Backend Resolution

FilePilot resolves the transfer backend in this order:

1. `backend_path` from config.
2. Bundled backend binary from the FilePilot installation.
3. Compatible backend from system `PATH`.
4. `BACKEND_NOT_FOUND` if none is usable.

## Security Boundary

FilePilot does not implement its own encryption protocol in the MVP.

Transport security is provided by the configured backend. FilePilot is responsible for safe session code generation, user guidance, log redaction, local timeout or cancellation, and hiding backend-specific commands from the normal workflow.

## Development Direction

The main CLI will be implemented in Go.

Expected technical shape:

```text
Go, Cobra, Cross-platform CLI, TransferBackend abstraction,
tar/gzip, TOML config, JSONL history, os/exec backend invocation
```

See [ISSUES.md](./ISSUES.md) for the proposed implementation slices.
