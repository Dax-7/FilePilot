# Third-Party Notices

This file lists third-party components that may be bundled in FilePilot release
artifacts. These components are not FilePilot-owned code and remain under their
respective licenses.

## croc

- Component: croc
- Purpose: Optional bundled transfer backend for the default FilePilot transfer workflow.
- License: MIT
- Upstream: https://github.com/schollz/croc
- Bundled files:
  - `croc-windows-amd64.exe`
  - `croc-linux-amd64`
- Package paths:
  - `backend/windows-amd64/croc.exe`
  - `backend/linux-amd64/croc`
- License notice: `licenses/croc-MIT-LICENSE.txt`

The croc backend is distributed as a separate third-party component. FilePilot
invokes it through its transfer backend abstraction; bundling croc does not make
croc part of FilePilot's own codebase.
