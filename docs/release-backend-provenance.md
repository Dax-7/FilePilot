# FilePilot Backend Provenance Record

Use this record for each backend binary selected for a FilePilot release package. Do not publish a release artifact until every field is complete and a human reviewer has approved the record.

## Release

- FilePilot version: v0.1.0
- Release acceptance status: pending

## Windows Backend Binary

- Target platform: windows-amd64
- Package name: `FilePilot-v0.1.0-windows-amd64.zip`
- Backend name: croc
- Backend version: v10.4.3
- Packaged path: `backend/windows-amd64/croc.exe`
- Local reviewed binary path: `C:\Users\Dan\scoop\apps\croc\10.4.3\croc.exe`
- Local shim path, not for packaging: `C:\Users\Dan\scoop\shims\croc.exe`
- SHA-256: `ec650bdc2bffacecf400239022c9fc378b051793f311a04272cb830471e29f3f`
- Source URL: `https://github.com/schollz/croc/releases/tag/v10.4.3`
- Source release, tag, or commit: `v10.4.3`
- Download URL: `https://github.com/schollz/croc/releases/download/v10.4.3/croc_v10.4.3_Windows-64bit.zip`
- Download checksum or signature source: Scoop manifest recorded archive SHA-256 `7d13d4871ceed35d62057df33e4f621fd6dbcb2e8cb5be0e754c20297efa15f8`; Scoop autoupdate points to `https://github.com/schollz/croc/releases/download/v10.4.3/croc_v10.4.3_checksums.txt`
- Reviewer: Dax
- Review date: 2026-07-01
- Review status: approved

## Linux Backend Binary

- Target platform: linux-amd64
- Package name: `FilePilot-v0.1.0-linux-amd64.tar.gz`
- Backend name: croc
- Backend version: v10.4.3
- Packaged path: `backend/linux-amd64/croc`
- Local reviewed binary path: `/usr/local/bin/croc`
- SHA-256: `dca4d381a27fdf1742c449884566ed0415ccaa0c90f745215a30eba185033426`
- Source URL: `https://github.com/schollz/croc/releases/tag/v10.4.3`
- Source release, tag, or commit: `v10.4.3`
- Download URL: `https://github.com/schollz/croc/releases/download/v10.4.3/croc_v10.4.3_Linux-64bit.tar.gz`
- Download checksum or signature source: `https://github.com/schollz/croc/releases/download/v10.4.3/croc_v10.4.3_checksums.txt`
- Reviewer: Dax
- Review date: 2026-07-01
- Review status: approved

## License And Notices

- Backend license name: MIT
- Backend license URL: `https://github.com/schollz/croc/blob/main/LICENSE`
- Windows backend license file reviewed: local Scoop package contains `C:\Users\Dan\scoop\apps\croc\10.4.3\LICENSE`
- Linux backend license file reviewed: upstream croc MIT license text recorded in `licenses/croc-MIT-LICENSE.txt`
- Required notices reviewed: approved; release packages include `LICENSE`, `NOTICE.md`, `THIRD_PARTY_NOTICES.md`, and `licenses/croc-MIT-LICENSE.txt`
- Package `NOTICE.md` source path: `NOTICE`
- NOTICE reviewer: Dax
- NOTICE review date: 2026-07-01
- NOTICE review status: approved

## Commands

Windows:

```powershell
& 'C:\Users\Dan\scoop\apps\croc\10.4.3\croc.exe' --version
Get-FileHash -Algorithm SHA256 'C:\Users\Dan\scoop\apps\croc\10.4.3\croc.exe'
```

Linux:

```bash
/usr/local/bin/croc --version
sha256sum /usr/local/bin/croc
```

## Packaging Inputs

Windows:

```powershell
.\scripts\package-release-windows.ps1 `
  -Version v0.1.0 `
  -CrocPath 'C:\Users\Dan\scoop\apps\croc\10.4.3\croc.exe' `
  -BackendSource 'https://github.com/schollz/croc/releases/tag/v10.4.3' `
  -BackendVersion 'v10.4.3' `
  -BackendLicense 'MIT' `
  -BackendLicenseUrl 'https://github.com/schollz/croc/blob/main/LICENSE' `
  -NoticePath .\NOTICE
```

Linux:

```bash
sh ./scripts/package-release-linux.sh \
  --version v0.1.0 \
  --croc-path /usr/local/bin/croc \
  --backend-source https://github.com/schollz/croc/releases/tag/v10.4.3 \
  --backend-version v10.4.3 \
  --backend-license MIT \
  --backend-license-url https://github.com/schollz/croc/blob/main/LICENSE \
  --notice-path ./NOTICE
```

The package scripts do not download backend binaries or verify upstream license terms. They only record publisher-supplied provenance, copy the reviewed local backend binary, compute package checksums, and include the reviewed `NOTICE.md`.
