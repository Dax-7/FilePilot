#!/usr/bin/env sh
set -eu

usage() {
  cat <<'USAGE'
Usage:
  sh ./scripts/check-release-package.sh --package-dir /path/to/FilePilot [--platform linux-amd64] [--skip-executable-checks]

Checks an extracted FilePilot release package for required files, manifest fields,
checksums, and same-platform doctor/help output.
USAGE
}

PACKAGE_DIR=""
PLATFORM=""
SKIP_EXECUTABLE_CHECKS=0
FAILURES=0

while [ "$#" -gt 0 ]; do
  case "$1" in
    --package-dir)
      PACKAGE_DIR=${2:-}
      shift 2
      ;;
    --platform)
      PLATFORM=${2:-}
      shift 2
      ;;
    --skip-executable-checks)
      SKIP_EXECUTABLE_CHECKS=1
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

if [ -z "$PACKAGE_DIR" ]; then
  usage >&2
  exit 2
fi

pass() {
  printf '[PASS] %s\n' "$1"
}

skip() {
  printf '[SKIP] %s\n' "$1"
}

fail() {
  FAILURES=$((FAILURES + 1))
  printf '[FAIL] %s\n' "$1"
}

json_string() {
  key=$1
  sed -n 's/.*"'"$key"'"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$MANIFEST" | head -n 1
}

json_number() {
  key=$1
  sed -n 's/.*"'"$key"'"[[:space:]]*:[[:space:]]*\([0-9][0-9]*\).*/\1/p' "$MANIFEST" | head -n 1
}

json_has_path() {
  path=$1
  grep -Eq '"path"[[:space:]]*:[[:space:]]*"'"$path"'"' "$MANIFEST"
}

check_required_file() {
  rel=$1
  if [ -f "$PACKAGE_ROOT/$rel" ]; then
    pass "Required file exists: $rel"
  else
    fail "Required file missing: $rel"
  fi
}

check_required_manifest_value() {
  name=$1
  value=$2
  if [ -n "$value" ]; then
    pass "Manifest field is present: $name"
  else
    fail "Manifest field is missing or empty: $name"
  fi
}

hash_file() {
  sha256sum "$1" | awk '{print $1}'
}

PACKAGE_ROOT=$(CDPATH= cd -- "$PACKAGE_DIR" && pwd -P)
MANIFEST="$PACKAGE_ROOT/release-manifest.json"
CHECKSUMS="$PACKAGE_ROOT/checksums.txt"

[ -f "$MANIFEST" ] || { printf '%s\n' "release-manifest.json not found under $PACKAGE_ROOT" >&2; exit 1; }
[ -f "$CHECKSUMS" ] || { printf '%s\n' "checksums.txt not found under $PACKAGE_ROOT" >&2; exit 1; }

command -v sha256sum >/dev/null 2>&1 || { printf '%s\n' "sha256sum is required." >&2; exit 1; }

if [ -z "$PLATFORM" ]; then
  PLATFORM=$(json_string target_platform)
fi

case "$PLATFORM" in
  windows-amd64)
    EXPECTED_FORMAT="zip"
    EXPECTED_BACKEND_PATH="backend/windows-amd64/croc.exe"
    CLI_EXECUTABLE="$PACKAGE_ROOT/filepilot.exe"
    SHORT_EXECUTABLE="$PACKAGE_ROOT/fp.exe"
    REQUIRED_FILES="filepilot-gui.exe filepilot.exe fp.exe install-cli.ps1 uninstall-cli.ps1 backend/windows-amd64/croc.exe QUICKSTART.md NOTICE.md checksums.txt release-manifest.json"
    ;;
  linux-amd64)
    EXPECTED_FORMAT="tar.gz"
    EXPECTED_BACKEND_PATH="backend/linux-amd64/croc"
    CLI_EXECUTABLE="$PACKAGE_ROOT/filepilot"
    SHORT_EXECUTABLE="$PACKAGE_ROOT/fp"
    REQUIRED_FILES="filepilot-gui filepilot fp install-cli.sh uninstall-cli.sh backend/linux-amd64/croc QUICKSTART.md NOTICE.md checksums.txt release-manifest.json"
    ;;
  *)
    printf '%s\n' "Unsupported target platform: $PLATFORM" >&2
    exit 1
    ;;
esac

for file in $REQUIRED_FILES; do
  check_required_file "$file"
done

SCHEMA_VERSION=$(json_number schema_version)
TARGET_PLATFORM=$(json_string target_platform)
PACKAGE_FORMAT=$(json_string package_format)
ACCEPTANCE_STATUS=$(json_string release_acceptance_status)
BACKEND_NAME=$(json_string name)
BACKEND_VERSION=$(json_string version)
BACKEND_SOURCE=$(json_string source)
BACKEND_LICENSE=$(json_string license)
BACKEND_LICENSE_URL=$(json_string license_url)
BACKEND_PATH=$(json_string package_path)
BACKEND_SHA=$(json_string sha256)

[ "$SCHEMA_VERSION" = "1" ] && pass "Manifest schema_version is 1" || fail "Manifest schema_version must be 1"
[ "$TARGET_PLATFORM" = "$PLATFORM" ] && pass "Manifest target_platform is $PLATFORM" || fail "Manifest target_platform is $TARGET_PLATFORM, expected $PLATFORM"
[ "$PACKAGE_FORMAT" = "$EXPECTED_FORMAT" ] && pass "Manifest package_format is $EXPECTED_FORMAT" || fail "Manifest package_format is $PACKAGE_FORMAT, expected $EXPECTED_FORMAT"
case "$ACCEPTANCE_STATUS" in
  pending|passed) pass "Manifest release_acceptance_status is $ACCEPTANCE_STATUS" ;;
  *) fail "Manifest release_acceptance_status must be pending or passed" ;;
esac

check_required_manifest_value filepilot_version "$(json_string filepilot_version)"
check_required_manifest_value package_name "$(json_string package_name)"
check_required_manifest_value build_time_utc "$(json_string build_time_utc)"
check_required_manifest_value git_commit "$(json_string git_commit)"
check_required_manifest_value backend.name "$BACKEND_NAME"
check_required_manifest_value backend.version "$BACKEND_VERSION"
check_required_manifest_value backend.source "$BACKEND_SOURCE"
check_required_manifest_value backend.license "$BACKEND_LICENSE"
check_required_manifest_value backend.license_url "$BACKEND_LICENSE_URL"
check_required_manifest_value backend.sha256 "$BACKEND_SHA"

[ "$BACKEND_NAME" = "croc" ] && pass "Manifest backend.name is croc" || fail "Manifest backend.name is $BACKEND_NAME, expected croc"
[ "$BACKEND_PATH" = "$EXPECTED_BACKEND_PATH" ] && pass "Manifest backend.package_path is $EXPECTED_BACKEND_PATH" || fail "Manifest backend.package_path is $BACKEND_PATH, expected $EXPECTED_BACKEND_PATH"
case "$BACKEND_SHA" in
  [0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f]) pass "Manifest backend.sha256 is a lowercase SHA-256 hash" ;;
  *) fail "Manifest backend.sha256 must be a lowercase SHA-256 hash" ;;
esac

ACTUAL_BACKEND_SHA=$(hash_file "$PACKAGE_ROOT/$EXPECTED_BACKEND_PATH")
[ "$ACTUAL_BACKEND_SHA" = "$BACKEND_SHA" ] && pass "Bundled backend hash matches manifest" || fail "Bundled backend hash does not match manifest"

if [ "$ACCEPTANCE_STATUS" = "passed" ]; then
  [ "$BACKEND_VERSION" != "unknown" ] && pass "Passed manifest backend.version is reviewed" || fail "Passed release manifests must not use backend.version unknown"
  [ "$BACKEND_LICENSE" != "pending-human-review" ] && pass "Passed manifest backend.license is reviewed" || fail "Passed release manifests must not use backend.license pending-human-review"
  [ "$BACKEND_LICENSE_URL" != "pending-human-review" ] && pass "Passed manifest backend.license_url is reviewed" || fail "Passed release manifests must not use backend.license_url pending-human-review"
fi

grep -Fq '"checksums.txt"' "$MANIFEST" && pass "Manifest generated_files includes checksums.txt" || fail "Manifest generated_files missing checksums.txt"
grep -Fq '"release-manifest.json"' "$MANIFEST" && pass "Manifest generated_files includes release-manifest.json" || fail "Manifest generated_files missing release-manifest.json"

for file in $REQUIRED_FILES; do
  case "$file" in
    checksums.txt|release-manifest.json)
      ;;
    *)
      if json_has_path "$file"; then
        pass "Manifest files includes required package file: $file"
      else
        fail "Manifest files missing required package file: $file"
      fi
      ;;
  esac
done

CHECKSUM_PATHS="$PACKAGE_ROOT/.filepilot-acceptance-checksum-paths.txt"
: > "$CHECKSUM_PATHS"
line_number=0
while IFS= read -r line || [ -n "$line" ]; do
  line_number=$((line_number + 1))
  line=$(printf '%s' "$line" | tr -d '\r')
  [ -n "$line" ] || continue
  expected_hash=${line%%  *}
  rel=${line#*  }
  if [ "$expected_hash" = "$line" ] || [ -z "$rel" ]; then
    fail "Invalid checksums.txt line $line_number: $line"
    continue
  fi
  if [ ! -f "$PACKAGE_ROOT/$rel" ]; then
    fail "checksums.txt entry missing from package: $rel"
    continue
  fi
  actual_hash=$(hash_file "$PACKAGE_ROOT/$rel")
  if [ "$actual_hash" = "$expected_hash" ]; then
    pass "Checksum matches: $rel"
  else
    fail "Checksum mismatch: $rel"
  fi
  printf '%s\n' "$rel" >> "$CHECKSUM_PATHS"
done < "$CHECKSUMS"

for file in $REQUIRED_FILES; do
  [ "$file" = "checksums.txt" ] && continue
  if grep -Fxq "$file" "$CHECKSUM_PATHS"; then
    pass "checksums.txt includes required package file: $file"
  else
    fail "checksums.txt missing required package file: $file"
  fi
done
rm -f "$CHECKSUM_PATHS"

CURRENT_PLATFORM=$(uname -s 2>/dev/null || printf unknown)
case "$CURRENT_PLATFORM" in
  Linux) CURRENT_PLATFORM="linux-amd64" ;;
  MINGW*|MSYS*|CYGWIN*) CURRENT_PLATFORM="windows-amd64" ;;
  *) CURRENT_PLATFORM="unknown" ;;
esac

if [ "$SKIP_EXECUTABLE_CHECKS" -eq 1 ]; then
  skip "Executable checks disabled by --skip-executable-checks"
elif [ "$CURRENT_PLATFORM" != "$PLATFORM" ]; then
  skip "Executable checks require $PLATFORM but current host is $CURRENT_PLATFORM"
else
  doctor_output=$("$CLI_EXECUTABLE" doctor 2>&1) || {
    fail "filepilot doctor exited with a non-zero status"
    doctor_output=""
  }
  printf '%s\n' "$doctor_output" | grep -Fq "Backend source: bundled" && pass "filepilot doctor reports bundled backend" || fail "filepilot doctor output did not report bundled backend"

  help_output=$("$CLI_EXECUTABLE" --help 2>&1) || {
    fail "filepilot --help exited with a non-zero status"
    help_output=""
  }
  printf '%s\n' "$help_output" | grep -Fq "Usage: filepilot" && pass "filepilot --help runs" || fail "filepilot --help output did not contain Usage: filepilot"

  short_help_output=$("$SHORT_EXECUTABLE" --help 2>&1) || {
    fail "fp --help exited with a non-zero status"
    short_help_output=""
  }
  printf '%s\n' "$short_help_output" | grep -Fq "Usage: filepilot" && pass "fp --help runs" || fail "fp --help output did not contain Usage: filepilot"
fi

if [ "$FAILURES" -gt 0 ]; then
  printf '%s\n' "Release package checks failed: $FAILURES"
  exit 1
fi

printf '%s\n' "Release package checks passed."
