# FilePilot v0.1.0 Release Notes

This is the publisher draft for the first FilePilot GitHub Release.

Do not publish this release text until:

- both platform packages have been generated with reviewed backend inputs,
- `release_acceptance_status` is set to `passed` only after package checks and human review,
- [release-acceptance-checklist.md](./release-acceptance-checklist.md) is complete,
- [release-backend-provenance.md](./release-backend-provenance.md) is complete for every bundled backend binary,
- the final package `NOTICE.md`, `checksums.txt`, and `release-manifest.json` files have been reviewed.

## GitHub Release Fields

Tag:

```text
v0.1.0
```

Title:

```text
FilePilot v0.1.0
```

Artifacts:

```text
FilePilot-v0.1.0-windows-amd64.zip
FilePilot-v0.1.0-linux-amd64.tar.gz
```

## GitHub Release Body

FilePilot v0.1.0 is the first portable package release for moving files and directories between two machines through a FilePilot Session ID.

This release includes the Desktop GUI, the canonical `filepilot` CLI, the short `fp` CLI entrypoint, optional user-scoped CLI registration scripts, bundled transfer support, quickstart instructions, notices, checksums, and a machine-readable release manifest.

## Supported Platforms

- Windows amd64
- Linux amd64

## Download

Choose the package for your machine:

- Windows: `FilePilot-v0.1.0-windows-amd64.zip`
- Linux: `FilePilot-v0.1.0-linux-amd64.tar.gz`

Each archive extracts to a single `FilePilot/` directory. FilePilot is portable for this release; there is no installer.

## Desktop GUI Quick Start

Windows:

```powershell
.\filepilot-gui.exe
```

Linux:

```bash
./filepilot-gui
```

To send, choose a file or directory, start the send, share the full FilePilot Session ID with the receiver, and keep the sender window open until receive completes.

To receive, paste the full FilePilot Session ID, choose a writable receive folder, and wait for FilePilot to finish.

## CLI Quick Start

Run from inside the extracted `FilePilot/` directory.

Windows:

```powershell
.\filepilot.exe doctor
.\filepilot.exe send C:\path\to\file-or-directory
.\filepilot.exe receive FP-river-copper-lamp-7K2Q9M4XP8
```

Linux:

```bash
./filepilot doctor
./filepilot send ./path/to/file-or-directory
./filepilot receive FP-river-copper-lamp-7K2Q9M4XP8
```

`fp` is the short executable name and runs the same CLI:

```bash
./fp send ./results
./fp receive FP-river-copper-lamp-7K2Q9M4XP8
```

## Optional CLI Registration

CLI registration is optional. It lets `filepilot` and `fp` run from any terminal working directory.

Windows:

```powershell
.\install-cli.ps1
```

Linux:

```bash
./install-cli.sh
```

Open a new terminal after registration, then run:

```bash
filepilot doctor
fp doctor
```

To unregister, run `.\uninstall-cli.ps1` on Windows or `./uninstall-cli.sh` on Linux from the same extracted package directory.

## What Is Included

- Desktop GUI for normal send and receive workflows.
- `filepilot` canonical CLI.
- `fp` short CLI entrypoint.
- Optional user-scoped CLI registration and unregistration scripts.
- Bundled transfer support for the default workflow.
- `QUICKSTART.md`, `NOTICE.md`, `checksums.txt`, and `release-manifest.json`.

## Known Limits

- Only Windows amd64 and Linux amd64 are supported in this release.
- macOS, Windows ARM, Linux ARM, `.deb`, `.rpm`, AppImage, Snap, Flatpak, MSI, NSIS, winget, choco, and scoop packages are not included.
- This release does not include automatic updates.
- This release does not include accounts, login, device binding, sync, resume, daemon mode, or GUI history.
- FilePilot does not provide an official hosted FilePilot rendezvous or relay service in v0.1.0.
- The GUI does not modify PATH or perform CLI registration.
- Sender and receiver must copy the full FilePilot Session ID exactly.
- The sender window or terminal must stay open until receive completes.
- VPN, proxy, firewall, company networks, campus networks, or restricted networks may prevent pairing or transfer.

## Verification

Each release package includes:

- `checksums.txt`
- `release-manifest.json`
- `NOTICE.md`

Before publication, the FilePilot release acceptance pass must verify package contents, manifest fields, checksums, bundled transfer support discovery, CLI help output, GUI startup, optional CLI registration, real cross-machine transfers, and human-reviewed backend provenance and notices.
