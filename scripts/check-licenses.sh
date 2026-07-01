#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)

for file in LICENSE NOTICE THIRD_PARTY_NOTICES.md licenses/croc-MIT-LICENSE.txt; do
  if [ ! -f "$REPO_ROOT/$file" ]; then
    printf '%s\n' "Required license file missing: $file" >&2
    exit 1
  fi
  printf '[PASS] Found %s\n' "$file"
done

if command -v go >/dev/null 2>&1; then
  printf '%s\n' "[INFO] Go modules in this build:"
  if ! (cd "$REPO_ROOT" && go list -m all); then
    printf '%s\n' "[WARN] go list -m all failed; dependency license review remains manual for this run."
  fi
else
  printf '%s\n' "[SKIP] go not found; skipping Go module listing."
fi

PACKAGE_LOCK="$REPO_ROOT/cmd/filepilot-gui/frontend/package-lock.json"
if [ -f "$PACKAGE_LOCK" ]; then
  printf '%s\n' "[INFO] npm packages recorded in package-lock.json:"
  sed -n 's/^[[:space:]]*"\([^"]*node_modules[^"]*\)"[[:space:]]*:.*/\1/p' "$PACKAGE_LOCK"
else
  printf '%s\n' "[SKIP] package-lock.json not found; skipping npm package listing."
fi
