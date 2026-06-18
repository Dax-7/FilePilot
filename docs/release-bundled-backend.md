# Bundled Backend Release Layout

FilePilot resolves a bundled transfer backend from a directory next to the FilePilot executable:

```text
<install-dir>/
  filepilot[.exe]
  fp[.exe]
  backend/
    linux-amd64/
      filepilot
    linux-arm64/
      filepilot
    darwin-amd64/
      filepilot
    darwin-arm64/
      filepilot
    windows-amd64/
      filepilot.exe
```

Release artifacts may bundle either a FilePilot-compatible transfer backend named `filepilot[.exe]` or a croc-compatible backend named `croc[.exe]`. The resolver derives the executable suffix from the runtime platform instead of hard-coding Windows paths into callers.

`filepilot` is the canonical executable name. Release artifacts should also provide `fp` as a short executable name that invokes the same CLI and command surface.

Resolution order remains:

1. `backend_path` from config.
2. Bundled backend at `backend/<goos>-<goarch>/filepilot[.exe]` or `backend/<goos>-<goarch>/croc[.exe]`.
3. Compatible backend from system `PATH`.
4. `BACKEND_NOT_FOUND`.

The bundled backend is invoked through FilePilot's `TransferBackend` abstraction. User-facing commands and normal output must continue to use FilePilot and transfer-backend terminology rather than backend-specific commands.

## Smoke Test

Automated smoke tests cover the release-layout contract with fake backend binaries:

```powershell
$env:GOCACHE='D:\Dax_Projects\FilePilot\.gocache'; go test ./internal/backend ./internal/doctor
```

The doctor smoke test clears PATH fallback inputs and verifies that a backend under the platform directory is reported with `source: bundled`.

## HITL Release Requirements

Before shipping real release artifacts, a human reviewer must approve:

- The exact backend binary source and version for each platform.
- The backend license and any notices that must ship with FilePilot.
- Checksums or equivalent integrity metadata for each backend artifact.
- The final release package contents for every supported platform.

FilePilot must not download backend binaries at runtime, modify PATH, or call package managers.
