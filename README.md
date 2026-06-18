# FilePilot

FilePilot is a cross-platform CLI and desktop GUI for moving files and directories between developer machines, lab servers, edge devices, and Agent workflows.

It provides a simple FilePilot command surface over a bundled croc-compatible transfer backend. Users do not need to call croc directly or install croc manually for the default workflow.

## Status

FilePilot has a runnable Public MVP path for sending and receiving files or directories between Linux and Windows using FilePilot commands.

The project is now in MVP Hardening Stage: validating real cross-machine transfer scenarios, stabilizing error boundaries, aligning documentation with the runnable CLI and GUI, and preparing release packaging. It is not yet a finalized Release Artifact.

See [docs/mvp-hardening-plan.md](./docs/mvp-hardening-plan.md) for the hardening gate and [docs/release-bundled-backend.md](./docs/release-bundled-backend.md) for release packaging policy.

## Quick Start

Sender:

```bash
filepilot send ./results
```

Receiver:

```bash
filepilot receive FP-river-copper-lamp-7K2Q9M4XP8
```

FilePilot generates a one-time FilePilot Session ID and passes it to the backend as a controlled passphrase. Users share the FilePilot Session ID through a trusted channel.

## Build From Source

Windows PowerShell:

```powershell
go build -o bin\filepilot.exe .\cmd\filepilot
Copy-Item bin\filepilot.exe bin\fp.exe
```

Linux or macOS:

```bash
go build -o bin/filepilot ./cmd/filepilot
cp bin/filepilot bin/fp
```

Add the `bin` directory to your `PATH` if you want to run `filepilot` or `fp` from any directory. Without that, run the local binary path directly, such as `.\bin\filepilot.exe receive <session-id>` on Windows.

## Desktop GUI

The desktop GUI is a human-facing Wails entrypoint. It does not replace the CLI: `filepilot send`, `filepilot receive`, JSON Agent API behavior, and release entrypoint semantics stay the same.

The GUI calls the shared Go transfer layer in `internal/transfer`; it does not construct shell commands or invoke the CLI by string concatenation. Receive saves default to `<home>/Downloads/FilePilot`, and users can choose another folder in the GUI.

Install Wails once:

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.2
```

Build the GUI on Windows:

```powershell
.\scripts\build-gui-windows.ps1
```

Build the GUI on Linux:

```bash
sh ./scripts/build-gui-linux.sh
```

On Ubuntu 24.04, Wails may need the newer WebKit tag if `webkit2gtk-4.0` is unavailable:

```bash
cd cmd/filepilot-gui
wails build -tags webkit2_41
```

Local GUI development:

```powershell
Set-Location .\cmd\filepilot-gui
wails dev
Set-Location ..\..
```

Before packaging release artifacts, prepare the bundled backend according to the release policy and verify local backend availability with `filepilot doctor` or a real GUI send/receive flow. By default the backend resolver looks for `filepilot.exe` or `croc.exe` on Windows, and `filepilot` or `croc` on Linux.

## Short Command

`filepilot` is the canonical command used in documentation and help text.

Release packages should also provide `fp` as a short executable name:

```bash
fp send ./results
fp receive FP-river-copper-lamp-7K2Q9M4XP8
```

`fp` invokes the same FilePilot CLI. It is not a separate product or command set.

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

The canonical receive subcommand is `receive`.

## What FilePilot Does

- Sends files and directories across machines without requiring direct IP reachability.
- Packages directories automatically before transfer.
- Unpacks only FilePilot-created directory packages.
- Uses a bundled croc-compatible backend by default.
- Provides stable JSON output for Agents.
- Provides a Wails desktop GUI for human send/receive workflows.
- Records transfer attempts with session IDs redacted.
- Runs local diagnostics with `filepilot doctor`.

## What FilePilot Does Not Do in the MVP

- It does not provide an official hosted rendezvous or relay service.
- It does not require users to deploy a separate server.
- It does not download backend binaries at runtime.
- It does not install packages through apt, brew, winget, scoop, or choco.
- It does not implement an independent encryption protocol.
- It does not include daemon, sync, resume, login, or history commands.

## Agent Usage

Use `--json` when FilePilot is called from automation.

```bash
filepilot send ./outputs --json
```

JSON mode is non-interactive. For example, this returns a structured error rather than prompting:

```bash
filepilot receive --json
```

## Directory Packages

When sending a directory, FilePilot creates a `.tar.gz` package with a `.filepilot/manifest.json` file.

Only FilePilot-created directory packages are auto-unpacked on receive. A user-supplied archive such as `file.zip` or `logs.tar.gz` is treated as a normal file.

## Diagnostics

Run local diagnostics before transfer attempts:

```bash
filepilot doctor
```

FilePilot resolves the transfer backend in this order:

1. `backend_path` from config.
2. Bundled backend binary from the FilePilot installation.
3. Compatible backend from system `PATH`.
4. `BACKEND_NOT_FOUND` if none is usable.

## Security Boundary

FilePilot does not implement its own encryption protocol in the MVP.

Transport security is provided by the configured backend. FilePilot is responsible for safe session code generation, user guidance, log redaction, local timeout or cancellation, and hiding backend-specific commands from the normal workflow.

FilePilot Session IDs are visible to sender and receiver during active transfer, but they are treated as sensitive while valid and are redacted from persistent history.

## Development

Run tests with a local Go cache:

```powershell
$env:GOCACHE='D:\Dax_Projects\FilePilot\.gocache'
go test ./...
```

The Public MVP is defined in [PRD.md](./PRD.md). Implementation decisions are recorded in [docs/adr](./docs/adr), and historical implementation slices are listed in [ISSUES.md](./ISSUES.md).
