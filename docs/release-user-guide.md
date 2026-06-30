# FilePilot Release User Guide

This guide is for the first portable FilePilot release packages for Windows amd64 and Linux amd64.

FilePilot lets two machines send and receive files or directories with a FilePilot Session ID. You do not need to install a separate transfer tool, know the other machine's IP address, use SSH, create an account, or run lower-level transfer commands for the default release workflow.

## Get Started

1. Download the package for your platform:
   - Windows: `FilePilot-v0.1.0-windows-amd64.zip`
   - Linux: `FilePilot-v0.1.0-linux-amd64.tar.gz`
2. Extract the archive.
3. Open the extracted `FilePilot/` directory.
4. Run either the Desktop GUI or the CLI from that directory.

The release package includes the FilePilot Desktop GUI, `filepilot`, `fp`, optional CLI registration scripts, bundled transfer support, `QUICKSTART.md`, `NOTICE.md`, `checksums.txt`, and `release-manifest.json`.

## Desktop GUI

Use the GUI for normal desktop send and receive workflows.

Windows:

```powershell
.\filepilot-gui.exe
```

Linux:

```bash
./filepilot-gui
```

To send, choose the file or directory, start the send, and share the full FilePilot Session ID with the receiver through a trusted channel. Keep the sender window open until the receiver finishes.

To receive, paste the full FilePilot Session ID, choose a writable receive folder, and wait for FilePilot to complete.

The GUI does not modify PATH or register CLI commands.

## CLI From The Package Directory

Use the CLI for terminals, servers, scripts, and Agent workflows.

Windows PowerShell:

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

`filepilot` is the canonical command name. `fp` is the short executable name and runs the same CLI:

```bash
./fp send ./results
./fp receive FP-river-copper-lamp-7K2Q9M4XP8
```

## Optional CLI Registration

CLI registration is optional. It lets you run `filepilot` and `fp` from any terminal working directory.

Windows registration is current-user only. It does not require administrator privileges and does not edit the system PATH:

```powershell
.\install-cli.ps1
```

Open a new terminal after registration, then run:

```powershell
filepilot doctor
fp doctor
```

To unregister the current extracted FilePilot directory:

```powershell
.\uninstall-cli.ps1
```

Linux registration creates user-level symlinks in `~/.local/bin`. It does not use `sudo` and does not write `/usr/local/bin`:

```bash
./install-cli.sh
```

If `~/.local/bin` is not on PATH, the script asks before creating links there. Open a new terminal after registration, then run:

```bash
filepilot doctor
fp doctor
```

To unregister symlinks that point to the current extracted FilePilot directory:

```bash
./uninstall-cli.sh
```

## Send And Receive

On the sender:

```bash
filepilot send ./results
```

FilePilot prints a receiver command containing a FilePilot Session ID. Share the full Session ID with the receiver exactly as shown. Keep the sender terminal or GUI window open until receive completes.

On the receiver:

```bash
filepilot receive FP-river-copper-lamp-7K2Q9M4XP8
```

If you run `filepilot receive` without a Session ID in human mode, FilePilot prompts for one. JSON mode is non-interactive and requires the Session ID as an argument.

When receiving, choose or configure a folder that your user account can write to. FilePilot-created directory packages are unpacked automatically. User-supplied archives such as `.zip` or `.tar.gz` files are treated as normal files.

## Diagnostics

Run diagnostics before a transfer if setup looks suspicious:

```bash
filepilot doctor
```

In a normal release package, `doctor` should report the transfer support source as `bundled`. If it reports a fatal transfer-support error from an extracted release package, the package may be incomplete or corrupted.

## Common Problems

Keep the sender open until receive completes. Closing the sender terminal or GUI window cancels the active transfer.

Copy the full FilePilot Session ID exactly. Missing characters, extra spaces, or an expired active send attempt can prevent the receiver from joining.

Choose a writable receive folder. If the target folder is read-only or blocked by permissions, choose another folder and try again.

VPN, proxy, firewall, company networks, campus networks, or restricted networks may prevent pairing or transfer. Try again from a less restricted network when possible.

If transfer setup fails, run `filepilot doctor` from the same package directory or from a newly opened registered terminal.

## Boundaries

The first release is a portable package release, not an installer ecosystem. It does not include automatic updates, package-manager installation, accounts, login, sync, resume, daemon mode, or an official FilePilot-hosted rendezvous or relay service.
