#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
if [ -z "${HOME:-}" ]; then
  printf '%s\n' "HOME is not set; cannot locate ~/.local/bin." >&2
  exit 1
fi
BIN_DIR="$HOME/.local/bin"

if [ ! -f "$SCRIPT_DIR/filepilot" ] || [ ! -f "$SCRIPT_DIR/fp" ]; then
  printf '%s\n' "Required CLI executables were not found next to this script: filepilot and fp" >&2
  exit 1
fi

path_has_dir() {
  target=$1
  target_trimmed=${target%/}
  target_real=$target_trimmed
  if [ -d "$target_trimmed" ]; then
    target_real=$(CDPATH= cd -- "$target_trimmed" && pwd -P)
  fi

  old_ifs=$IFS
  IFS=:
  path_value=${PATH:-}
  for entry in $path_value; do
    [ -n "$entry" ] || continue
    entry_trimmed=${entry%/}
    if [ "$entry_trimmed" = "$target_trimmed" ]; then
      IFS=$old_ifs
      return 0
    fi
    if [ -d "$entry_trimmed" ] && [ -d "$target_trimmed" ]; then
      entry_real=$(CDPATH= cd -- "$entry_trimmed" && pwd -P)
      if [ "$entry_real" = "$target_real" ]; then
        IFS=$old_ifs
        return 0
      fi
    fi
  done
  IFS=$old_ifs
  return 1
}

install_link() {
  name=$1
  target="$SCRIPT_DIR/$name"
  link="$BIN_DIR/$name"

  if [ -L "$link" ]; then
    current_target=$(readlink "$link")
    if [ "$current_target" = "$target" ]; then
      printf '%s\n' "$name is already registered at $link"
      return 0
    fi
    printf '%s\n' "Refusing to replace existing symlink: $link -> $current_target" >&2
    exit 1
  fi

  if [ -e "$link" ]; then
    printf '%s\n' "Refusing to replace existing file: $link" >&2
    exit 1
  fi

  ln -s "$target" "$link"
  printf '%s\n' "Registered $name at $link"
}

if ! path_has_dir "$BIN_DIR"; then
  printf '%s\n' "$BIN_DIR is not currently on PATH."
  printf '%s' "Create FilePilot CLI symlinks there anyway? [y/N] "
  if ! read answer; then
    answer=
  fi
  case "$answer" in
    y|Y|yes|YES)
      ;;
    *)
      printf '%s\n' "No changes made."
      printf '%s\n' "Add $BIN_DIR to PATH, then run this script again."
      exit 0
      ;;
  esac
fi

mkdir -p "$BIN_DIR"
install_link filepilot
install_link fp

printf '%s\n' "Open a new terminal before running filepilot or fp from another directory."
