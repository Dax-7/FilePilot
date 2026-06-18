#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
GUI_DIR="$REPO_ROOT/cmd/filepilot-gui"

cd "$GUI_DIR"

(cd frontend && npm install)

if [ -r /etc/os-release ]; then
  . /etc/os-release
  if [ "${ID:-}" = "ubuntu" ] && [ "${VERSION_ID:-}" = "24.04" ]; then
    printf '%s\n' "Ubuntu 24.04 detected. If webkit2gtk-4.0 is unavailable, run: wails build -tags webkit2_41"
  fi
fi

if command -v pkg-config >/dev/null 2>&1; then
  if ! pkg-config --exists webkit2gtk-4.0; then
    printf '%s\n' "webkit2gtk-4.0 was not found. If the build fails, run: wails build -tags webkit2_41"
  fi
else
  printf '%s\n' "pkg-config was not found. Install Wails Linux dependencies, then retry the build."
fi

wails build
