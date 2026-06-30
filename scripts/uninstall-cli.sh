#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
if [ -z "${HOME:-}" ]; then
  printf '%s\n' "HOME is not set; cannot locate ~/.local/bin." >&2
  exit 1
fi
BIN_DIR="$HOME/.local/bin"

remove_link() {
  name=$1
  target="$SCRIPT_DIR/$name"
  link="$BIN_DIR/$name"

  if [ ! -L "$link" ]; then
    if [ -e "$link" ]; then
      printf '%s\n' "Leaving non-symlink in place: $link"
    else
      printf '%s\n' "$name is not registered at $link"
    fi
    return 0
  fi

  current_target=$(readlink "$link")
  if [ "$current_target" != "$target" ]; then
    printf '%s\n' "Leaving symlink that points somewhere else: $link -> $current_target"
    return 0
  fi

  rm "$link"
  printf '%s\n' "Removed $link"
}

remove_link filepilot
remove_link fp

printf '%s\n' "Open a new terminal for PATH changes to take effect."
