# MVP Hardening Plan

FilePilot has a runnable Public MVP path. The purpose of this stage is to decide whether that implementation is stable enough to package for users, without expanding MVP scope.

## Stage Boundary

MVP Hardening Stage validates the implemented Public MVP against:

- real cross-machine transfer scenarios,
- stable FilePilot error boundaries,
- Agent API regression behavior,
- command and documentation accuracy,
- release packaging prerequisites.

It is not a new feature phase, protocol redesign, or release artifact by itself.

## Release Readiness Standard

Before producing Release Artifacts:

- the Required Stability Gate must pass,
- Regression Boundary Tests must pass,
- README must describe the runnable MVP workflow,
- `fp` must be available as a short executable entrypoint in release packaging,
- Release Packaging Design must document backend source, license, checksum, and package contents for human review.
- First-release user instructions must clearly explain common failure points: keep the sender running until receive completes, copy the full FilePilot Session ID exactly, choose a writable receive folder, and check VPN, proxy, firewall, or restricted-network behavior when pairing fails.
- Release acceptance should be quick to execute: automate repeatable package-content, `doctor`, help-output, build, and checksum checks; reserve manual steps for real cross-machine transfers, GUI launch behavior, and human review of backend provenance and notices.

## First Release Acceptance

The first Release Artifact acceptance pass should cover:

- `go test ./...`.
- Windows GUI build succeeds.
- Linux GUI build succeeds.
- Each package contains `filepilot[.exe]`, `fp[.exe]`, `filepilot-gui[.exe]`, bundled `backend/<goos>-<goarch>/croc[.exe]`, user instructions, license or notice files, checksums, and `release-manifest.json`.
- From each extracted package, `filepilot doctor` reports the backend source as `bundled`.
- From each extracted package, `filepilot --help` and `fp --help` run successfully.
- The GUI starts without immediately exiting.
- The optional CLI registration step makes `filepilot` and `fp` available from a new terminal.
- Real Windows-to-Linux and Linux-to-Windows transfers pass for small files and directories, including paths with spaces and non-ASCII names.
- Invalid Session ID, missing backend, timeout, and cancellation behavior are understandable to users.

## First Release Scope

The first Release stage should deliver:

- Windows amd64 and Linux amd64 Release Artifacts.
- A single extracted package per platform containing the Desktop GUI, Canonical Command Name, Short Executable Name, bundled croc-compatible backend, user instructions, license or notice files, checksums, and `release-manifest.json`.
- A machine-readable `release-manifest.json` that records FilePilot version, target platform, package file list, backend name, backend version, backend source, backend checksum, build time, git commit, and release acceptance status.
- Explicit CLI registration and unregistration scripts so users can opt in to running `filepilot` and `fp` from any terminal working directory.
- CLI registration must be user-scoped and reversible. On Windows it may add the extracted FilePilot directory to the current user's PATH and must not require administrator privileges or edit the system PATH. On Linux it may create `filepilot` and `fp` symlinks in `~/.local/bin`, must not use `sudo`, must not write `/usr/local/bin`, and should only prompt when `~/.local/bin` is not already on PATH.
- A quick release acceptance flow that automates repeatable checks where practical.
- A concise user guide at `docs/release-user-guide.md` for GUI and CLI usage, including common failure points. README should link to it rather than carrying the full release-user workflow.
- GitHub Releases-ready artifact names and release notes.

The first Release stage should not include:

- Windows installers such as MSI or NSIS, or package-manager publication through winget, choco, or scoop.
- Linux distro packages such as `.deb` or `.rpm`, AppImage, Snap, or Flatpak.
- Automatic updates.
- Account, login, device-binding, sync, resume, daemon, or cloud-service features.
- An official FilePilot-hosted rendezvous or relay service.
- GUI-driven PATH mutation or GUI-driven CLI installation.
- Official macOS, Windows ARM, or Linux ARM Release Artifacts.

## Required Stability Gate

These live scenarios must pass before release packaging:

- Linux sender to Windows receiver with a small file.
- Linux sender to Windows receiver with a directory.
- Windows sender to Linux receiver with a small file.
- Path with spaces.
- Non-ASCII file or directory name.
- Invalid FilePilot Session ID.
- Missing backend behavior through `doctor` and transfer commands.
- Local timeout behavior for send or receive.

## Recommended Stability Evidence

These scenarios improve confidence but do not automatically block release packaging:

- Large file transfer, such as 500 MB or 1 GB.
- Repeated receive into a location with existing names.
- User cancellation with Ctrl+C.
- Receiver never joins.
- Interrupted network or backend failure.
- User-supplied `.zip` or `.tar.gz` remains a File Payload and is not auto-unpacked.

## Regression Boundary Tests

Automated tests should guard:

- stable JSON error shape,
- `BACKEND_NOT_FOUND`,
- `INVALID_SESSION_ID`,
- `MISSING_SESSION_ID`,
- `LOCAL_TIMEOUT`,
- `CANCELLED`,
- `PATH_NOT_FOUND`,
- `PERMISSION_DENIED`,
- `TRANSFER_FAILED`,
- backend resolution order,
- FilePilot Session ID redaction,
- safe receive and unpack behavior.

Current coverage snapshot:

- Covered by existing automated tests: `BACKEND_NOT_FOUND`, `INVALID_SESSION_ID`, `MISSING_SESSION_ID`, `LOCAL_TIMEOUT`, `CANCELLED`, `PATH_NOT_FOUND`, `TRANSFER_FAILED`, backend resolution order, FilePilot Session ID redaction, safe receive and unpack behavior.
- Known gap: add an explicit CLI-level regression test for `PERMISSION_DENIED` if the test environment can model unreadable paths reliably across Windows and Unix-like platforms.
- Known gap: review human error output for actionable hints and backend terminology leakage without making full text byte-stable.

## Output Stability Levels

Agent API JSON output is a strong contract. Tests should verify stable fields, `error.code`, `status`, `mode`, and stdout/stderr behavior.

Human output is a quality gate rather than a byte-for-byte contract. Tests and review should verify that human errors include actionable guidance, do not expose backend raw commands, do not leak full FilePilot Session IDs outside the active transfer prompt, and do not require users to understand backend-specific terminology.

## Execution Record

Use this template for each live scenario:

```text
Scenario:
Date:
Sender OS:
Receiver OS:
FilePilot build:
Backend source:
Command:
Expected result:
Actual result:
Status: pass / fail / blocked
Notes:
```
