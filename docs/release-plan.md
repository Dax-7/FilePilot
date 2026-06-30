# FilePilot Release Plan

This document is the working specification for the first FilePilot Release stage. It describes the release package shape, publisher responsibilities, user-facing entrypoints, and acceptance checks for the first downloadable artifacts.

## Goal

Produce downloadable FilePilot Release Artifacts that ordinary Windows and Linux users can extract and use through either the Desktop GUI or CLI without installing croc, knowing peer IP addresses, using SSH, or creating accounts.

The first release is a portable package release, not an installer ecosystem.

## Supported Platforms

The first release officially supports:

- Windows amd64
- Linux amd64

macOS, Windows ARM, Linux ARM, `.deb`, `.rpm`, AppImage, Snap, Flatpak, MSI, NSIS, winget, choco, and scoop are future packaging targets, not first-release acceptance criteria.

## Artifact Names

GitHub Releases artifacts should use platform-specific names:

```text
FilePilot-<version>-windows-amd64.zip
FilePilot-<version>-linux-amd64.tar.gz
```

Each archive should extract to a single top-level `FilePilot/` directory.

The first formal release version is `v0.1.0`:

```text
FilePilot-v0.1.0-windows-amd64.zip
FilePilot-v0.1.0-linux-amd64.tar.gz
```

The Git tag should be `v0.1.0`, the GitHub Release title should be `FilePilot v0.1.0`, and `release-manifest.json` should record the same version. Patch fixes should use `v0.1.1`, and later feature or platform additions should use later minor versions such as `v0.2.0`.

## Package Layout

Windows:

```text
FilePilot/
  filepilot-gui.exe
  filepilot.exe
  fp.exe
  install-cli.ps1
  uninstall-cli.ps1
  backend/
    windows-amd64/
      croc.exe
  QUICKSTART.md
  NOTICE.md
  checksums.txt
  release-manifest.json
```

Linux:

```text
FilePilot/
  filepilot-gui
  filepilot
  fp
  install-cli.sh
  uninstall-cli.sh
  backend/
    linux-amd64/
      croc
  QUICKSTART.md
  NOTICE.md
  checksums.txt
  release-manifest.json
```

`filepilot` is the Canonical Command Name. `fp` is the Short Executable Name and invokes the same CLI command surface.

## Backend Policy

First-release artifacts bundle a human-reviewed croc-compatible backend in the package.

Packaging scripts should accept local, already-reviewed backend binary paths as publisher inputs. Packaging scripts should not automatically download backend binaries.

This is a publisher-side requirement only. Users of the final Release Artifact do not install croc, provide backend paths, or understand backend layout details for the default workflow.

FilePilot must not download backend binaries at runtime, modify PATH automatically, or call platform package managers.

## GUI And CLI Entrypoints

The Desktop GUI, `filepilot`, and `fp` are peer official entrypoints in the same package.

- The GUI is for normal desktop send and receive workflows.
- The CLI is for terminals, servers, scripts, and Agent workflows.
- Both use the same FilePilot Session ID and transfer semantics.
- The GUI should not modify PATH or perform CLI registration in the first release.

## CLI Registration

Release packages should provide optional registration and unregistration scripts.

Windows:

- `install-cli.ps1` may add the extracted FilePilot directory to the current user's PATH.
- It must not require administrator privileges.
- It must not edit the system PATH.
- It should avoid duplicate PATH entries.
- It should tell the user to open a new terminal.
- `uninstall-cli.ps1` should remove the current FilePilot directory from the current user's PATH.

Linux:

- `install-cli.sh` may create `~/.local/bin/filepilot` and `~/.local/bin/fp` symlinks.
- It must not use `sudo`.
- It must not write `/usr/local/bin`.
- It should only prompt when `~/.local/bin` is not on PATH.
- `uninstall-cli.sh` should only remove symlinks that point to the current FilePilot directory.

## User Instructions

The release user guide should live at `docs/release-user-guide.md`. Each package should include a short `QUICKSTART.md` derived from that guide.

User instructions should keep the GUI flow short and make CLI usage explicit. They must explain common failure points:

- Keep the sender window or terminal open until receive completes.
- Copy the full FilePilot Session ID exactly.
- Choose a writable receive folder.
- VPN, proxy, firewall, company networks, campus networks, or restricted networks may prevent pairing or transfer.

Instructions should not expose croc commands or require users to understand backend terminology.

## Manifest And Checksums

Each package should include:

- `checksums.txt`
- `release-manifest.json`

`release-manifest.json` should record:

- FilePilot version
- target platform
- package file list
- backend name
- backend version
- backend source
- backend license
- backend license URL
- backend checksum
- build time
- git commit
- release acceptance status

## Minimum Acceptance

The first release acceptance pass should be quick to execute and automate repeatable checks where practical.

Automated or scriptable checks:

- `go test ./...`
- Windows GUI build succeeds.
- Linux GUI build succeeds.
- Package contents match this document.
- From each extracted package, `filepilot doctor` reports backend source as `bundled`.
- From each extracted package, `filepilot --help` and `fp --help` run successfully.
- Package checksums and `release-manifest.json` are present.

Manual checks:

- GUI starts without immediately exiting.
- Optional CLI registration makes `filepilot` and `fp` available from a new terminal.
- Windows-to-Linux and Linux-to-Windows transfers pass for small files and directories.
- Paths with spaces and non-ASCII names transfer correctly.
- Invalid Session ID, missing backend, timeout, and cancellation behavior are understandable.
- Backend source, license, notices, checksums, and final package contents are human-reviewed before publication.

## Implementation Tasks

Implement the first Release stage in small tasks:

1. Release package scripts
   - Windows and Linux packaging scripts.
   - Accept local, already-reviewed croc-compatible backend binary paths.
   - Build package directories, archives, checksums, and `release-manifest.json`.
2. CLI registration scripts
   - `install-cli.ps1` and `uninstall-cli.ps1`.
   - `install-cli.sh` and `uninstall-cli.sh`.
3. Release user guide
   - `docs/release-user-guide.md`.
   - Package-level `QUICKSTART.md` derived from the guide.
4. Release acceptance script or checklist
   - Fast checks for package contents, `doctor`, help output, checksums, and manifest fields.
   - Manual transfer and GUI verification record template.
5. Backend provenance and notices
   - Backend version, source URL, license, notices, and checksums.
6. `v0.1.0` release notes
   - GitHub Release description.
   - Supported platforms.
   - Known limits.
   - GUI and CLI quick start.

The recommended implementation order is 1, 2, 3, 4, 5, then 6.

## Packaging Script Entrypoints

Windows packaging:

```powershell
.\scripts\package-release-windows.ps1 `
  -Version v0.1.0 `
  -CrocPath C:\release-inputs\croc-windows-amd64.exe `
  -BackendSource "https://example.invalid/reviewed-croc-source" `
  -BackendVersion "<reviewed backend version>" `
  -BackendLicense "<reviewed backend license>" `
  -BackendLicenseUrl "https://example.invalid/reviewed-croc-license" `
  -NoticePath C:\release-inputs\FilePilot-NOTICE.md
```

Linux packaging:

```bash
sh ./scripts/package-release-linux.sh \
  --version v0.1.0 \
  --croc-path /release-inputs/croc-linux-amd64 \
  --backend-source https://example.invalid/reviewed-croc-source \
  --backend-version "<reviewed backend version>" \
  --backend-license "<reviewed backend license>" \
  --backend-license-url https://example.invalid/reviewed-croc-license \
  --notice-path /release-inputs/FilePilot-NOTICE.md
```

The scripts do not download backend binaries. `-CrocPath` and `--croc-path` must point to a local backend binary that has already been reviewed by the publisher. Backend version, source, license, license URL, notice file, and checksums remain publisher-supplied release inputs. By default, generated manifests use `release_acceptance_status: "pending"` until the release acceptance pass and human backend review are complete. Setting acceptance status to `passed` requires explicit reviewed backend version, license, license URL, and notice path inputs.

## Out Of Scope

The first release does not include:

- complex installers
- package manager publication
- automatic updates
- accounts or login
- device binding
- daemon mode
- sync
- resume
- GUI history
- official FilePilot-hosted rendezvous or relay services
- GUI-driven PATH mutation
- official macOS or ARM release artifacts
