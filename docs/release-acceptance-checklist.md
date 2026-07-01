# FilePilot Release Acceptance Checklist

Use this checklist for the first portable FilePilot release artifacts after package generation. It separates repeatable package checks from manual transfer, GUI, and human-review evidence.

## Automated Package Checks

Run the package checker against each extracted package directory.

Windows package on Windows:

```powershell
.\scripts\check-release-package.ps1 `
  -PackageDir .\release\staging\windows-amd64\FilePilot `
  -Platform windows-amd64
```

Linux package on Linux:

```bash
sh ./scripts/check-release-package.sh \
  --package-dir ./release/staging/linux-amd64/FilePilot \
  --platform linux-amd64
```

When checking a package on the opposite host platform, use the same command with executable checks disabled. This still verifies package contents, checksums, and manifest fields, but it does not prove `doctor` or help output on the target OS.

Windows PowerShell:

```powershell
.\scripts\check-release-package.ps1 `
  -PackageDir .\release\staging\linux-amd64\FilePilot `
  -Platform linux-amd64 `
  -SkipExecutableChecks
```

Linux shell:

```bash
sh ./scripts/check-release-package.sh \
  --package-dir ./release/staging/windows-amd64/FilePilot \
  --platform windows-amd64 \
  --skip-executable-checks
```

The same-platform checks verify:

- required release package files,
- `release-manifest.json` required fields,
- manifest backend package path, license fields, and backend checksum,
- manifest file records for required package files,
- `checksums.txt` entries and SHA-256 values,
- `LICENSE`, `NOTICE.md`, `THIRD_PARTY_NOTICES.md`, and `licenses/` third-party license files,
- `filepilot doctor` reports `Backend source: bundled`,
- `filepilot --help` and `fp --help` run successfully.

These scripts do not approve backend provenance, licenses, notices, final package publication, cross-machine transfers, or GUI launch behavior.

Before setting `release_acceptance_status` to `passed`, complete [release-backend-provenance.md](./release-backend-provenance.md) and use an explicitly reviewed `NOTICE.md` through the packaging script's notice path option.

## Manual Verification Record

Create one record per scenario.

```text
Scenario:
Date:
Tester:
Sender OS:
Receiver OS:
FilePilot package:
FilePilot version:
Git commit:
Backend source:
Backend version:
Backend license:
Backend license URL:
Backend SHA-256:
NOTICE review status:
Command or GUI flow:
Expected result:
Actual result:
Status: pass / fail / blocked
Notes:
```

## Required Manual Scenarios

Record each scenario before publication:

- GUI starts without immediately exiting on Windows.
- GUI starts without immediately exiting on Linux.
- Optional CLI registration makes `filepilot` and `fp` available from a new Windows terminal.
- Optional CLI registration makes `filepilot` and `fp` available from a new Linux terminal.
- Linux sender to Windows receiver with a small file.
- Linux sender to Windows receiver with a directory.
- Windows sender to Linux receiver with a small file.
- Transfer path with spaces.
- Transfer path with non-ASCII names.
- Invalid FilePilot Session ID behavior is understandable.
- Missing backend behavior through `doctor` and transfer commands is understandable.
- Local timeout behavior for send or receive is understandable.
- Backend source, license, notices, checksums, and final package contents are human-reviewed before publication.

## Optional Confidence Scenarios

Record these when time permits:

- Large file transfer, such as 500 MB or 1 GB.
- Repeated receive into a location with existing names.
- User cancellation with Ctrl+C.
- Receiver never joins.
- Interrupted network or backend failure.
- User-supplied `.zip` or `.tar.gz` remains a File Payload and is not auto-unpacked.
