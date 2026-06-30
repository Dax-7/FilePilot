# FilePilot Quick Start

Run FilePilot from this extracted `FilePilot/` directory.

## Desktop GUI

Windows:

```powershell
.\filepilot-gui.exe
```

Linux:

```bash
./filepilot-gui
```

Send: choose a file or directory, start the send, share the full FilePilot Session ID, and keep the sender window open until receive completes.

Receive: paste the full FilePilot Session ID, choose a writable receive folder, and wait for FilePilot to finish.

## CLI

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

`fp` is the short executable name and runs the same CLI.

## Optional CLI Registration

Windows:

```powershell
.\install-cli.ps1
```

Linux:

```bash
./install-cli.sh
```

Open a new terminal after registration, then run `filepilot doctor` or `fp doctor` from any directory.

To unregister, run `.\uninstall-cli.ps1` on Windows or `./uninstall-cli.sh` on Linux from this same extracted directory.

## Common Checks

- Keep the sender terminal or GUI window open until receive completes.
- Copy the full FilePilot Session ID exactly.
- Choose a writable receive folder.
- VPN, proxy, firewall, company networks, campus networks, or restricted networks may prevent pairing or transfer.
