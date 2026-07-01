# FilePilot Notices

This is a draft notice file for FilePilot v0.1.0 release packages. Do not use it for a published release until the pending fields below are reviewed and resolved.

## FilePilot

FilePilot release package:

- Version: v0.1.0
- Source: `https://github.com/Dax-7/FilePilot`
- License: pending project license decision

## Bundled Transfer Support

This FilePilot package includes a human-reviewed backend-compatible executable for the default transfer workflow.

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
- Backend version: pending Linux verification
- Target platform: linux-amd64
- Packaged path: `backend/linux-amd64/croc`
- Source URL: `https://github.com/schollz/croc/releases/tag/v10.4.3`
- License: MIT
- License URL: `https://github.com/schollz/croc/blob/main/LICENSE`
- Backend SHA-256: `dca4d381a27fdf1742c449884566ed0415ccaa0c90f745215a30eba185033426`

## Required Third-Party Notices

The bundled croc backend is licensed under MIT. Before publication, include the exact required croc license notice from the reviewed release source or package license file.

## Review

- Reviewer: Dax
- Review date: 2026-07-01
- Review status: approved

Do not publish a release package while this file still contains placeholder values or pending review status.
