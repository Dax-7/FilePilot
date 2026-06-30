# FilePilot Backend Provenance Record

Use this record for each backend binary selected for a FilePilot release package. Do not publish a release artifact until every field is complete and a human reviewer has approved the record.

## Release

- FilePilot version:
- Target platform: windows-amd64 / linux-amd64
- Package name:
- Release acceptance status: pending / passed

## Backend Binary

- Backend name: croc
- Backend version:
- Packaged path: `backend/<goos>-<goarch>/croc[.exe]`
- Local reviewed binary path:
- SHA-256:
- Source URL:
- Source release, tag, or commit:
- Download URL:
- Download checksum or signature source:
- Reviewer:
- Review date:
- Review status: pending / approved / rejected

## License And Notices

- License name:
- License URL:
- License file reviewed: yes / no
- Required notices reviewed: yes / no
- Package `NOTICE.md` source path:
- NOTICE reviewer:
- NOTICE review date:
- NOTICE review status: pending / approved / rejected

## Commands

Windows:

```powershell
Get-FileHash -Algorithm SHA256 C:\release-inputs\croc-windows-amd64.exe
```

Linux:

```bash
sha256sum /release-inputs/croc-linux-amd64
```

## Packaging Inputs

Windows:

```powershell
.\scripts\package-release-windows.ps1 `
  -Version v0.1.0 `
  -CrocPath C:\release-inputs\croc-windows-amd64.exe `
  -BackendSource "https://example.invalid/reviewed-backend-release" `
  -BackendVersion "<reviewed backend version>" `
  -BackendLicense "<reviewed license name>" `
  -BackendLicenseUrl "https://example.invalid/reviewed-backend-license" `
  -NoticePath C:\release-inputs\FilePilot-NOTICE.md
```

Linux:

```bash
sh ./scripts/package-release-linux.sh \
  --version v0.1.0 \
  --croc-path /release-inputs/croc-linux-amd64 \
  --backend-source https://example.invalid/reviewed-backend-release \
  --backend-version "<reviewed backend version>" \
  --backend-license "<reviewed license name>" \
  --backend-license-url https://example.invalid/reviewed-backend-license \
  --notice-path /release-inputs/FilePilot-NOTICE.md
```

The package scripts do not download backend binaries or verify upstream license terms. They only record publisher-supplied provenance, copy the reviewed local backend binary, compute package checksums, and include the reviewed `NOTICE.md`.
