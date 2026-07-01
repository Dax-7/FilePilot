# FilePilot Notices

This notice file is prepared for FilePilot v0.1.0 release packages.

## FilePilot

FilePilot release package:

- Version: v0.1.0
- Source: `https://github.com/Dax-7/FilePilot`
- License: MIT
- License file: `LICENSE`

## Bundled Transfer Support

This FilePilot package may include a human-reviewed backend-compatible executable for the default transfer workflow. Bundled backend binaries are third-party components and are not FilePilot-owned code.

Windows package:

- Backend name: croc
- Backend version: v10.4.3
- Target platform: windows-amd64
- Packaged path: `backend/windows-amd64/croc.exe`
- Source URL: `https://github.com/schollz/croc/releases/tag/v10.4.3`
- License: MIT
- License URL: `https://github.com/schollz/croc/blob/main/LICENSE`
- Backend SHA-256: `ec650bdc2bffacecf400239022c9fc378b051793f311a04272cb830471e29f3f`

Linux package:

- Backend name: croc
- Backend version: v10.4.3
- Target platform: linux-amd64
- Packaged path: `backend/linux-amd64/croc`
- Source URL: `https://github.com/schollz/croc/releases/tag/v10.4.3`
- License: MIT
- License URL: `https://github.com/schollz/croc/blob/main/LICENSE`
- Backend SHA-256: `dca4d381a27fdf1742c449884566ed0415ccaa0c90f745215a30eba185033426`

## Required Third-Party Notices

The bundled croc backend is licensed under MIT. The package includes croc notice metadata in `THIRD_PARTY_NOTICES.md` and the croc MIT license text in `licenses/croc-MIT-LICENSE.txt`.

## Review

- Reviewer: Dax
- Review date: 2026-07-01
- Review status: approved
