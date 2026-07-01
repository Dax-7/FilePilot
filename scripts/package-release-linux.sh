#!/usr/bin/env sh
set -eu

usage() {
  cat <<'USAGE'
Usage:
  sh ./scripts/package-release-linux.sh --version v0.1.0 --croc-path /path/to/croc --backend-source <source>

Options:
  --version <version>            FilePilot release version, such as v0.1.0.
  --croc-path <path>             Local reviewed linux-amd64 croc-compatible backend binary.
  --backend-source <source>      Human-reviewed backend source description or URL.
  --backend-version <version>    Backend version. Defaults to detected version or unknown.
  --backend-license <license>    Reviewed backend license name.
  --backend-license-url <url>    Reviewed backend license URL.
  --acceptance-status <status>   pending or passed. Defaults to pending.
  --output-dir <dir>             Release output directory. Defaults to ./release.
  --quickstart-path <path>       Optional package QUICKSTART.md source.
  --notice-path <path>           Optional package NOTICE.md source.
  --skip-build                   Use existing bin/filepilot and GUI build output.
USAGE
}

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)

VERSION=""
CROC_PATH=""
BACKEND_SOURCE=""
BACKEND_VERSION=""
BACKEND_LICENSE=""
BACKEND_LICENSE_URL=""
ACCEPTANCE_STATUS="pending"
OUTPUT_DIR="$REPO_ROOT/release"
QUICKSTART_PATH=""
NOTICE_PATH=""
SKIP_BUILD=0

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      VERSION=${2:-}
      shift 2
      ;;
    --croc-path)
      CROC_PATH=${2:-}
      shift 2
      ;;
    --backend-source)
      BACKEND_SOURCE=${2:-}
      shift 2
      ;;
    --backend-version)
      BACKEND_VERSION=${2:-}
      shift 2
      ;;
    --backend-license)
      BACKEND_LICENSE=${2:-}
      shift 2
      ;;
    --backend-license-url)
      BACKEND_LICENSE_URL=${2:-}
      shift 2
      ;;
    --acceptance-status)
      ACCEPTANCE_STATUS=${2:-}
      shift 2
      ;;
    --output-dir)
      OUTPUT_DIR=${2:-}
      shift 2
      ;;
    --quickstart-path)
      QUICKSTART_PATH=${2:-}
      shift 2
      ;;
    --notice-path)
      NOTICE_PATH=${2:-}
      shift 2
      ;;
    --skip-build)
      SKIP_BUILD=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      printf '%s\n' "Unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [ -z "$VERSION" ] || [ -z "$CROC_PATH" ] || [ -z "$BACKEND_SOURCE" ]; then
  usage >&2
  exit 2
fi

if [ "$ACCEPTANCE_STATUS" != "pending" ] && [ "$ACCEPTANCE_STATUS" != "passed" ]; then
  printf '%s\n' "--acceptance-status must be pending or passed." >&2
  exit 2
fi

if [ ! -f "$CROC_PATH" ]; then
  printf '%s\n' "Backend binary not found: $CROC_PATH" >&2
  exit 1
fi

command -v sha256sum >/dev/null 2>&1 || { printf '%s\n' "sha256sum is required." >&2; exit 1; }
command -v tar >/dev/null 2>&1 || { printf '%s\n' "tar is required." >&2; exit 1; }

PLATFORM="linux-amd64"
ARCHIVE_NAME="FilePilot-$VERSION-$PLATFORM.tar.gz"
RELEASE_ROOT=$(mkdir -p "$OUTPUT_DIR" && cd "$OUTPUT_DIR" && pwd)
STAGING_PARENT="$RELEASE_ROOT/staging/linux-amd64"
PACKAGE_DIR="$STAGING_PARENT/FilePilot"
BUILD_DIR="$RELEASE_ROOT/build/linux-amd64"
ARCHIVE_PATH="$RELEASE_ROOT/$ARCHIVE_NAME"

remove_under_release() {
  path=$1
  case "$path" in
    "$RELEASE_ROOT"/*)
      rm -rf "$path"
      ;;
    *)
      printf '%s\n' "Refusing to remove path outside release root: $path" >&2
      exit 1
      ;;
  esac
}

copy_required_file() {
  source=$1
  destination=$2
  if [ ! -f "$source" ]; then
    printf '%s\n' "Required file not found: $source" >&2
    exit 1
  fi
  mkdir -p "$(dirname "$destination")"
  cp "$source" "$destination"
}

copy_optional_file() {
  source=$1
  destination=$2
  if [ -f "$source" ]; then
    mkdir -p "$(dirname "$destination")"
    cp "$source" "$destination"
    return 0
  fi
  return 1
}

copy_required_directory() {
  source=$1
  destination=$2
  if [ ! -d "$source" ]; then
    printf '%s\n' "Required directory not found: $source" >&2
    exit 1
  fi
  mkdir -p "$destination"
  cp -R "$source"/. "$destination"/
}

json_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

detect_backend_version() {
  backend=$1
  if [ -x "$backend" ]; then
    if output=$("$backend" --version 2>&1) && [ -n "$output" ]; then
      printf '%s' "$output"
      return
    fi
    if output=$("$backend" version 2>&1) && [ -n "$output" ]; then
      printf '%s' "$output"
      return
    fi
  fi
  printf '%s' "unknown"
}

mkdir -p "$RELEASE_ROOT"
remove_under_release "$STAGING_PARENT"
remove_under_release "$BUILD_DIR"
rm -f "$ARCHIVE_PATH"
mkdir -p "$PACKAGE_DIR" "$BUILD_DIR"

if [ "$SKIP_BUILD" -eq 1 ]; then
  CLI_PATH="$REPO_ROOT/bin/filepilot"
else
  CLI_PATH="$BUILD_DIR/filepilot"
  (cd "$REPO_ROOT" && go build -o "$CLI_PATH" ./cmd/filepilot)
  sh "$SCRIPT_DIR/build-gui-linux.sh"
fi

GUI_PATH=""
for candidate in "$REPO_ROOT/cmd/filepilot-gui/build/bin/filepilot-gui" "$REPO_ROOT/bin/filepilot-gui"; do
  if [ -f "$candidate" ]; then
    GUI_PATH=$candidate
    break
  fi
done
if [ -z "$GUI_PATH" ]; then
  printf '%s\n' "Built GUI executable not found under cmd/filepilot-gui/build/bin or bin." >&2
  exit 1
fi

copy_required_file "$CLI_PATH" "$PACKAGE_DIR/filepilot"
copy_required_file "$CLI_PATH" "$PACKAGE_DIR/fp"
copy_required_file "$GUI_PATH" "$PACKAGE_DIR/filepilot-gui"
copy_required_file "$CROC_PATH" "$PACKAGE_DIR/backend/linux-amd64/croc"
copy_required_file "$REPO_ROOT/LICENSE" "$PACKAGE_DIR/LICENSE"
copy_required_file "$REPO_ROOT/THIRD_PARTY_NOTICES.md" "$PACKAGE_DIR/THIRD_PARTY_NOTICES.md"
copy_required_directory "$REPO_ROOT/licenses" "$PACKAGE_DIR/licenses"
chmod 0755 "$PACKAGE_DIR/filepilot" "$PACKAGE_DIR/fp" "$PACKAGE_DIR/filepilot-gui" "$PACKAGE_DIR/backend/linux-amd64/croc"

copy_optional_file "$REPO_ROOT/scripts/install-cli.sh" "$PACKAGE_DIR/install-cli.sh" || true
copy_optional_file "$REPO_ROOT/scripts/uninstall-cli.sh" "$PACKAGE_DIR/uninstall-cli.sh" || true
[ -f "$PACKAGE_DIR/install-cli.sh" ] && chmod 0755 "$PACKAGE_DIR/install-cli.sh"
[ -f "$PACKAGE_DIR/uninstall-cli.sh" ] && chmod 0755 "$PACKAGE_DIR/uninstall-cli.sh"

if [ -n "$QUICKSTART_PATH" ]; then
  copy_required_file "$QUICKSTART_PATH" "$PACKAGE_DIR/QUICKSTART.md"
elif ! copy_optional_file "$REPO_ROOT/docs/release-quickstart.md" "$PACKAGE_DIR/QUICKSTART.md"; then
  cat > "$PACKAGE_DIR/QUICKSTART.md" <<'QUICKSTART'
# FilePilot Quick Start

This package was generated before the final release quickstart was added.

GUI: run ./filepilot-gui.
CLI: run ./filepilot send <path> or ./filepilot receive <session-id> from this directory.

Keep the sender window or terminal open until the receiver finishes.

Release status: pending user-guide task.
QUICKSTART
fi

if [ -n "$NOTICE_PATH" ]; then
  copy_required_file "$NOTICE_PATH" "$PACKAGE_DIR/NOTICE.md"
elif copy_optional_file "$REPO_ROOT/NOTICE" "$PACKAGE_DIR/NOTICE.md"; then
  :
elif ! copy_optional_file "$REPO_ROOT/docs/release-notice-template.md" "$PACKAGE_DIR/NOTICE.md"; then
  cat > "$PACKAGE_DIR/NOTICE.md" <<'NOTICE'
# FilePilot Notices

Backend provenance, license notices, and final publication approval are pending human review.
Do not publish this package until NOTICE.md has been reviewed for the selected backend binary.
NOTICE
fi

if [ -z "$BACKEND_VERSION" ]; then
  BACKEND_VERSION=$(detect_backend_version "$CROC_PATH")
fi
if [ -z "$BACKEND_LICENSE" ]; then
  BACKEND_LICENSE="pending-human-review"
fi
if [ -z "$BACKEND_LICENSE_URL" ]; then
  BACKEND_LICENSE_URL="pending-human-review"
fi

if [ "$ACCEPTANCE_STATUS" = "passed" ]; then
  if [ -z "$BACKEND_VERSION" ] || [ "$BACKEND_VERSION" = "unknown" ]; then
    printf '%s\n' "--acceptance-status passed requires an explicit reviewed --backend-version." >&2
    exit 2
  fi
  if [ -z "$BACKEND_LICENSE" ] || [ "$BACKEND_LICENSE" = "pending-human-review" ]; then
    printf '%s\n' "--acceptance-status passed requires an explicit reviewed --backend-license." >&2
    exit 2
  fi
  if [ -z "$BACKEND_LICENSE_URL" ] || [ "$BACKEND_LICENSE_URL" = "pending-human-review" ]; then
    printf '%s\n' "--acceptance-status passed requires an explicit reviewed --backend-license-url." >&2
    exit 2
  fi
  if [ -z "$NOTICE_PATH" ]; then
    printf '%s\n' "--acceptance-status passed requires an explicit reviewed --notice-path." >&2
    exit 2
  fi
fi

BACKEND_PACKAGE_PATH="backend/linux-amd64/croc"
BACKEND_HASH=$(sha256sum "$PACKAGE_DIR/$BACKEND_PACKAGE_PATH" | awk '{print $1}')
GIT_COMMIT=$(cd "$REPO_ROOT" && git rev-parse HEAD 2>/dev/null || printf '%s' "unknown")
BUILD_TIME_UTC=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

FILES_JSON="$BUILD_DIR/files.json"
printf '[\n' > "$FILES_JSON"
first=1
(cd "$PACKAGE_DIR" && find . -type f ! -name checksums.txt ! -name release-manifest.json | sed 's#^\./##' | LC_ALL=C sort) |
while IFS= read -r rel; do
  hash=$(sha256sum "$PACKAGE_DIR/$rel" | awk '{print $1}')
  size=$(wc -c < "$PACKAGE_DIR/$rel" | tr -d ' ')
  escaped_rel=$(json_escape "$rel")
  if [ "$first" -eq 0 ]; then
    printf ',\n' >> "$FILES_JSON"
  fi
  first=0
  printf '    {"path":"%s","size_bytes":%s,"sha256":"%s"}' "$escaped_rel" "$size" "$hash" >> "$FILES_JSON"
done
printf '\n  ]\n' >> "$FILES_JSON"

cat > "$PACKAGE_DIR/release-manifest.json" <<MANIFEST
{
  "schema_version": 1,
  "filepilot_version": "$(json_escape "$VERSION")",
  "target_platform": "$PLATFORM",
  "package_name": "$(json_escape "$ARCHIVE_NAME")",
  "package_format": "tar.gz",
  "build_time_utc": "$(json_escape "$BUILD_TIME_UTC")",
  "git_commit": "$(json_escape "$GIT_COMMIT")",
  "release_acceptance_status": "$(json_escape "$ACCEPTANCE_STATUS")",
  "backend": {
    "name": "croc",
    "version": "$(json_escape "$BACKEND_VERSION")",
    "source": "$(json_escape "$BACKEND_SOURCE")",
    "license": "$(json_escape "$BACKEND_LICENSE")",
    "license_url": "$(json_escape "$BACKEND_LICENSE_URL")",
    "package_path": "$BACKEND_PACKAGE_PATH",
    "sha256": "$BACKEND_HASH"
  },
  "files": $(cat "$FILES_JSON"),
  "generated_files": ["checksums.txt", "release-manifest.json"]
}
MANIFEST

(cd "$PACKAGE_DIR" && find . -type f ! -name checksums.txt | sed 's#^\./##' | LC_ALL=C sort) |
while IFS= read -r rel; do
  hash=$(sha256sum "$PACKAGE_DIR/$rel" | awk '{print $1}')
  printf '%s  %s\n' "$hash" "$rel"
done > "$PACKAGE_DIR/checksums.txt"

(cd "$STAGING_PARENT" && tar -czf "$ARCHIVE_PATH" FilePilot)

printf '%s\n' "Created release package: $ARCHIVE_PATH"
printf '%s\n' "Staged package directory: $PACKAGE_DIR"
