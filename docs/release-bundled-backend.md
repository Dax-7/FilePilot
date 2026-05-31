# Bundled Backend Release Layout

FilePilot resolves a bundled transfer backend from a directory next to the FilePilot executable:

```text
<install-dir>/
  filepilot[.exe]
  backend/
    linux-amd64/
      filepilot-backend
    linux-arm64/
      filepilot-backend
    darwin-amd64/
      filepilot-backend
    darwin-arm64/
      filepilot-backend
    windows-amd64/
      filepilot-backend.exe
```

The resolver also accepts platform-native script or batch extensions during tests. Release artifacts should prefer `filepilot-backend` on Unix-like platforms and `filepilot-backend.exe` on Windows.

Resolution order remains:

1. `backend_path` from config.
2. Bundled backend at `backend/<goos>-<goarch>/filepilot-backend`.
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
