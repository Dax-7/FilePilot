param(
    [Parameter(Mandatory = $true)]
    [string]$PackageDir,

    [ValidateSet("", "windows-amd64", "linux-amd64")]
    [string]$Platform = "",

    [switch]$SkipExecutableChecks
)

$ErrorActionPreference = 'Stop'
$Failures = 0

function Pass {
    param([Parameter(Mandatory = $true)][string]$Message)
    Write-Host "[PASS] $Message"
}

function Skip {
    param([Parameter(Mandatory = $true)][string]$Message)
    Write-Host "[SKIP] $Message"
}

function Fail {
    param([Parameter(Mandatory = $true)][string]$Message)
    $script:Failures++
    Write-Host "[FAIL] $Message"
}

function Package-Path {
    param([Parameter(Mandatory = $true)][string]$RelativePath)
    return Join-Path $PackageRoot ($RelativePath -replace '/', [System.IO.Path]::DirectorySeparatorChar)
}

function Test-RequiredFile {
    param([Parameter(Mandatory = $true)][string]$RelativePath)
    $path = Package-Path -RelativePath $RelativePath
    if (Test-Path -LiteralPath $path -PathType Leaf) {
        Pass "Required file exists: $RelativePath"
        return
    }
    Fail "Required file missing: $RelativePath"
}

function Test-RequiredManifestValue {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)]$Value
    )
    if ($null -eq $Value -or "$Value" -eq "") {
        Fail "Manifest field is missing or empty: $Name"
    }
    else {
        Pass "Manifest field is present: $Name"
    }
}

function Get-FileHashLower {
    param([Parameter(Mandatory = $true)][string]$Path)
    return (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant()
}

function Test-Command {
    param(
        [Parameter(Mandatory = $true)][string]$Executable,
        [Parameter(Mandatory = $true)][string[]]$Arguments,
        [Parameter(Mandatory = $true)][string]$ExpectedText,
        [Parameter(Mandatory = $true)][string]$Label
    )
    Push-Location $PackageRoot
    try {
        $output = & $Executable @Arguments 2>&1
        $exitCode = $LASTEXITCODE
    }
    finally {
        Pop-Location
    }

    $text = "$output"
    if ($exitCode -ne 0) {
        Fail "$Label exited with code $exitCode"
        return
    }
    if ($text -notmatch [regex]::Escape($ExpectedText)) {
        Fail "$Label output did not contain expected text: $ExpectedText"
        return
    }
    Pass "$Label"
}

$PackageRoot = (Resolve-Path -LiteralPath $PackageDir).Path
$manifestPath = Package-Path -RelativePath 'release-manifest.json'
$checksumsPath = Package-Path -RelativePath 'checksums.txt'

if (-not (Test-Path -LiteralPath $manifestPath -PathType Leaf)) {
    throw "release-manifest.json not found under $PackageRoot"
}
if (-not (Test-Path -LiteralPath $checksumsPath -PathType Leaf)) {
    throw "checksums.txt not found under $PackageRoot"
}

$manifest = Get-Content -Raw -LiteralPath $manifestPath | ConvertFrom-Json
if ($Platform -eq "") {
    $Platform = "$($manifest.target_platform)"
}

switch ($Platform) {
    'windows-amd64' {
        $expectedFormat = 'zip'
        $requiredFiles = @(
            'filepilot-gui.exe',
            'filepilot.exe',
            'fp.exe',
            'install-cli.ps1',
            'uninstall-cli.ps1',
            'backend/windows-amd64/croc.exe',
            'QUICKSTART.md',
            'NOTICE.md',
            'checksums.txt',
            'release-manifest.json'
        )
        $cliExecutable = Package-Path -RelativePath 'filepilot.exe'
        $shortExecutable = Package-Path -RelativePath 'fp.exe'
        $expectedBackendPath = 'backend/windows-amd64/croc.exe'
    }
    'linux-amd64' {
        $expectedFormat = 'tar.gz'
        $requiredFiles = @(
            'filepilot-gui',
            'filepilot',
            'fp',
            'install-cli.sh',
            'uninstall-cli.sh',
            'backend/linux-amd64/croc',
            'QUICKSTART.md',
            'NOTICE.md',
            'checksums.txt',
            'release-manifest.json'
        )
        $cliExecutable = Package-Path -RelativePath 'filepilot'
        $shortExecutable = Package-Path -RelativePath 'fp'
        $expectedBackendPath = 'backend/linux-amd64/croc'
    }
    default {
        throw "Unsupported target platform: $Platform"
    }
}

foreach ($file in $requiredFiles) {
    Test-RequiredFile -RelativePath $file
}

if ([int]$manifest.schema_version -eq 1) { Pass 'Manifest schema_version is 1' } else { Fail 'Manifest schema_version must be 1' }
if ("$($manifest.target_platform)" -eq $Platform) { Pass "Manifest target_platform is $Platform" } else { Fail "Manifest target_platform is $($manifest.target_platform), expected $Platform" }
if ("$($manifest.package_format)" -eq $expectedFormat) { Pass "Manifest package_format is $expectedFormat" } else { Fail "Manifest package_format is $($manifest.package_format), expected $expectedFormat" }
if (@('pending', 'passed') -contains "$($manifest.release_acceptance_status)") { Pass "Manifest release_acceptance_status is $($manifest.release_acceptance_status)" } else { Fail 'Manifest release_acceptance_status must be pending or passed' }

Test-RequiredManifestValue -Name 'filepilot_version' -Value $manifest.filepilot_version
Test-RequiredManifestValue -Name 'package_name' -Value $manifest.package_name
Test-RequiredManifestValue -Name 'build_time_utc' -Value $manifest.build_time_utc
Test-RequiredManifestValue -Name 'git_commit' -Value $manifest.git_commit
Test-RequiredManifestValue -Name 'backend.name' -Value $manifest.backend.name
Test-RequiredManifestValue -Name 'backend.version' -Value $manifest.backend.version
Test-RequiredManifestValue -Name 'backend.source' -Value $manifest.backend.source
Test-RequiredManifestValue -Name 'backend.license' -Value $manifest.backend.license
Test-RequiredManifestValue -Name 'backend.license_url' -Value $manifest.backend.license_url
Test-RequiredManifestValue -Name 'backend.sha256' -Value $manifest.backend.sha256

if ("$($manifest.backend.name)" -eq 'croc') { Pass 'Manifest backend.name is croc' } else { Fail "Manifest backend.name is $($manifest.backend.name), expected croc" }
if ("$($manifest.backend.package_path)" -eq $expectedBackendPath) { Pass "Manifest backend.package_path is $expectedBackendPath" } else { Fail "Manifest backend.package_path is $($manifest.backend.package_path), expected $expectedBackendPath" }
if ("$($manifest.backend.sha256)" -match '^[a-f0-9]{64}$') { Pass 'Manifest backend.sha256 is a lowercase SHA-256 hash' } else { Fail 'Manifest backend.sha256 must be a lowercase SHA-256 hash' }

$backendHash = Get-FileHashLower -Path (Package-Path -RelativePath $expectedBackendPath)
if ($backendHash -eq "$($manifest.backend.sha256)") { Pass 'Bundled backend hash matches manifest' } else { Fail 'Bundled backend hash does not match manifest' }

if ("$($manifest.release_acceptance_status)" -eq 'passed') {
    if ("$($manifest.backend.version)" -eq 'unknown') {
        Fail 'Passed release manifests must not use backend.version unknown'
    }
    if ("$($manifest.backend.license)" -eq 'pending-human-review') {
        Fail 'Passed release manifests must not use backend.license pending-human-review'
    }
    if ("$($manifest.backend.license_url)" -eq 'pending-human-review') {
        Fail 'Passed release manifests must not use backend.license_url pending-human-review'
    }
}

$generated = @($manifest.generated_files)
foreach ($generatedFile in @('checksums.txt', 'release-manifest.json')) {
    if ($generated -contains $generatedFile) {
        Pass "Manifest generated_files includes $generatedFile"
    }
    else {
        Fail "Manifest generated_files missing $generatedFile"
    }
}

$manifestFilesByPath = @{}
foreach ($record in @($manifest.files)) {
    $relative = "$($record.path)"
    $filePath = Package-Path -RelativePath $relative
    if (-not (Test-Path -LiteralPath $filePath -PathType Leaf)) {
        Fail "Manifest file entry missing from package: $relative"
        continue
    }
    $actualSize = (Get-Item -LiteralPath $filePath).Length
    $actualHash = Get-FileHashLower -Path $filePath
    if ([int64]$record.size_bytes -ne $actualSize) {
        Fail "Manifest size mismatch for ${relative}: $($record.size_bytes) != $actualSize"
    }
    elseif ("$($record.sha256)" -ne $actualHash) {
        Fail "Manifest hash mismatch for $relative"
    }
    else {
        Pass "Manifest file record matches: $relative"
    }
    $manifestFilesByPath[$relative] = $true
}

foreach ($required in $requiredFiles) {
    if ($required -in @('checksums.txt', 'release-manifest.json')) {
        continue
    }
    if ($manifestFilesByPath.ContainsKey($required)) {
        Pass "Manifest files includes required package file: $required"
    }
    else {
        Fail "Manifest files missing required package file: $required"
    }
}

$checksumFilesByPath = @{}
$lineNumber = 0
foreach ($line in Get-Content -LiteralPath $checksumsPath) {
    $lineNumber++
    if ($line.Trim() -eq '') {
        continue
    }
    if ($line -notmatch '^([a-fA-F0-9]{64})\s{2,}(.+)$') {
        Fail "Invalid checksums.txt line ${lineNumber}: $line"
        continue
    }
    $expectedHash = $Matches[1].ToLowerInvariant()
    $relative = $Matches[2]
    $filePath = Package-Path -RelativePath $relative
    if (-not (Test-Path -LiteralPath $filePath -PathType Leaf)) {
        Fail "checksums.txt entry missing from package: $relative"
        continue
    }
    $actualHash = Get-FileHashLower -Path $filePath
    if ($actualHash -eq $expectedHash) {
        Pass "Checksum matches: $relative"
    }
    else {
        Fail "Checksum mismatch: $relative"
    }
    $checksumFilesByPath[$relative] = $true
}

foreach ($required in $requiredFiles) {
    if ($required -eq 'checksums.txt') {
        continue
    }
    if ($checksumFilesByPath.ContainsKey($required)) {
        Pass "checksums.txt includes required package file: $required"
    }
    else {
        Fail "checksums.txt missing required package file: $required"
    }
}

$currentPlatform = if ($env:OS -eq 'Windows_NT') { 'windows-amd64' } else { 'linux-amd64' }
if ($SkipExecutableChecks) {
    Skip 'Executable checks disabled by -SkipExecutableChecks'
}
elseif ($currentPlatform -ne $Platform) {
    Skip "Executable checks require $Platform but current host is $currentPlatform"
}
else {
    Test-Command -Executable $cliExecutable -Arguments @('doctor') -ExpectedText 'Backend source: bundled' -Label 'filepilot doctor reports bundled backend'
    Test-Command -Executable $cliExecutable -Arguments @('--help') -ExpectedText 'Usage: filepilot' -Label 'filepilot --help runs'
    Test-Command -Executable $shortExecutable -Arguments @('--help') -ExpectedText 'Usage: filepilot' -Label 'fp --help runs'
}

if ($Failures -gt 0) {
    Write-Host "Release package checks failed: $Failures"
    exit 1
}

Write-Host 'Release package checks passed.'
