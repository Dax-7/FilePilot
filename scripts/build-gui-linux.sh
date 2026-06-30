#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
GUI_DIR="$REPO_ROOT/cmd/filepilot-gui"

cd "$GUI_DIR"

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
