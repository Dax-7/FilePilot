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
