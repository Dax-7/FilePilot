#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
GUI_DIR="$REPO_ROOT/cmd/filepilot-gui"

cd "$GUI_DIR"

if ! command -v node >/dev/null 2>&1; then
  printf '%s\n' "Node.js was not found. Install Node.js 18 or newer, then retry the build." >&2
  exit 1
fi

if ! command -v npm >/dev/null 2>&1; then
  printf '%s\n' "npm was not found. Install npm 8 or newer with Node.js 18 or newer, then retry the build." >&2
  exit 1
fi

NODE_MAJOR=$(node -p "Number(process.versions.node.split('.')[0])")
if [ "$NODE_MAJOR" -lt 18 ]; then
  printf '%s\n' "Node.js 18 or newer is required for the GUI build." >&2
  printf '%s\n' "Current Node.js version: $(node -v)" >&2
  printf '%s\n' "Node.js path: $(command -v node)" >&2
  printf '%s\n' "npm path: $(command -v npm)" >&2
  printf '%s\n' "Install Node.js 20 LTS, remove frontend/node_modules, then retry: sh ./scripts/build-gui-linux.sh" >&2
  exit 1
fi

(cd frontend && npm install)

WAILS_TAGS=""

if command -v pkg-config >/dev/null 2>&1; then
  if pkg-config --exists webkit2gtk-4.0; then
    WAILS_TAGS=""
  elif pkg-config --exists webkit2gtk-4.1; then
    WAILS_TAGS="webkit2_41"
    printf '%s\n' "webkit2gtk-4.1 detected. Building with Wails tag: webkit2_41"
  else
    printf '%s\n' "webkit2gtk-4.0 or webkit2gtk-4.1 was not found. Install Wails Linux dependencies, then retry the build."
  fi
else
  printf '%s\n' "pkg-config was not found. Install Wails Linux dependencies, then retry the build."
fi

if [ -n "$WAILS_TAGS" ]; then
  wails build -tags "$WAILS_TAGS"
else
  wails build
fi
