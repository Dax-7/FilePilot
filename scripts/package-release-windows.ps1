param(
    [Parameter(Mandatory = $true)]
    [string]$Version,

    [Parameter(Mandatory = $true)]
    [string]$CrocPath,

    [Parameter(Mandatory = $true)]
    [string]$BackendSource,

    [string]$BackendVersion = "",

    [string]$BackendLicense = "",

    [string]$BackendLicenseUrl = "",

    [ValidateSet("pending", "passed")]
    [string]$AcceptanceStatus = "pending",

    [string]$OutputDir = "",

    [string]$QuickstartPath = "",

    [string]$NoticePath = "",

    [switch]$SkipBuild
)

$ErrorActionPreference = 'Stop'

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = Resolve-Path (Join-Path $ScriptDir '..')
if ($OutputDir -eq "") {
    $OutputDir = Join-Path $RepoRoot 'release'
}

$Platform = 'windows-amd64'
$ArchiveName = "FilePilot-$Version-$Platform.zip"
$ReleaseRoot = [System.IO.Path]::GetFullPath($OutputDir)
$StagingParent = Join-Path $ReleaseRoot 'staging\windows-amd64'
$PackageDir = Join-Path $StagingParent 'FilePilot'
$BuildDir = Join-Path $ReleaseRoot 'build\windows-amd64'
$ArchivePath = Join-Path $ReleaseRoot $ArchiveName

function Remove-TreeUnder {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][string]$Root
    )
    $fullPath = [System.IO.Path]::GetFullPath($Path)
    $fullRoot = [System.IO.Path]::GetFullPath($Root).TrimEnd('\')
    if ($fullPath -ne $fullRoot -and -not $fullPath.StartsWith($fullRoot + '\', [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "Refusing to remove path outside release root: $fullPath"
    }
    if (Test-Path -LiteralPath $fullPath) {
        Remove-Item -LiteralPath $fullPath -Recurse -Force
    }
}

function Copy-RequiredFile {
    param(
        [Parameter(Mandatory = $true)][string]$Source,
        [Parameter(Mandatory = $true)][string]$Destination
    )
    if (-not (Test-Path -LiteralPath $Source -PathType Leaf)) {
        throw "Required file not found: $Source"
    }
    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $Destination) | Out-Null
    Copy-Item -LiteralPath $Source -Destination $Destination -Force
}

function Copy-OptionalFile {
    param(
        [Parameter(Mandatory = $true)][string]$Source,
        [Parameter(Mandatory = $true)][string]$Destination
    )
    if (Test-Path -LiteralPath $Source -PathType Leaf) {
        New-Item -ItemType Directory -Force -Path (Split-Path -Parent $Destination) | Out-Null
        Copy-Item -LiteralPath $Source -Destination $Destination -Force
        return $true
    }
    return $false
}

function Copy-RequiredDirectory {
    param(
        [Parameter(Mandatory = $true)][string]$Source,
        [Parameter(Mandatory = $true)][string]$Destination
    )
    if (-not (Test-Path -LiteralPath $Source -PathType Container)) {
        throw "Required directory not found: $Source"
    }
    New-Item -ItemType Directory -Force -Path $Destination | Out-Null
    Copy-Item -Path (Join-Path $Source '*') -Destination $Destination -Recurse -Force
}

function Get-RelativePath {
    param(
        [Parameter(Mandatory = $true)][string]$Root,
        [Parameter(Mandatory = $true)][string]$Path
    )
    $fullRoot = (Resolve-Path -LiteralPath $Root).Path.TrimEnd('\')
    $fullPath = (Resolve-Path -LiteralPath $Path).Path
    return $fullPath.Substring($fullRoot.Length + 1).Replace('\', '/')
}

function Detect-BackendVersion {
    param([Parameter(Mandatory = $true)][string]$Path)
    foreach ($arg in @('--version', 'version')) {
        try {
            $output = & $Path $arg 2>&1
            if ($LASTEXITCODE -eq 0 -and "$output".Trim() -ne '') {
                return "$output".Trim()
            }
        }
        catch {
        }
    }
    return 'unknown'
}

New-Item -ItemType Directory -Force -Path $ReleaseRoot | Out-Null
Remove-TreeUnder -Path $StagingParent -Root $ReleaseRoot
Remove-TreeUnder -Path $BuildDir -Root $ReleaseRoot
if (Test-Path -LiteralPath $ArchivePath) {
    Remove-Item -LiteralPath $ArchivePath -Force
}
New-Item -ItemType Directory -Force -Path $PackageDir | Out-Null
New-Item -ItemType Directory -Force -Path $BuildDir | Out-Null

$resolvedCroc = Resolve-Path -LiteralPath $CrocPath

if ($SkipBuild) {
    $cliPath = Join-Path $RepoRoot 'bin\filepilot.exe'
    $guiPathCandidates = @(
        (Join-Path $RepoRoot 'cmd\filepilot-gui\build\bin\filepilot-gui.exe'),
        (Join-Path $RepoRoot 'bin\filepilot-gui.exe')
    )
}
else {
    $cliPath = Join-Path $BuildDir 'filepilot.exe'
    Push-Location $RepoRoot
    try {
        go build -o $cliPath .\cmd\filepilot
        & (Join-Path $ScriptDir 'build-gui-windows.ps1')
    }
    finally {
        Pop-Location
    }
    $guiPathCandidates = @(
        (Join-Path $RepoRoot 'cmd\filepilot-gui\build\bin\filepilot-gui.exe'),
        (Join-Path $RepoRoot 'bin\filepilot-gui.exe')
    )
}

$guiPath = $null
foreach ($candidate in $guiPathCandidates) {
    if (Test-Path -LiteralPath $candidate -PathType Leaf) {
        $guiPath = $candidate
        break
    }
}
if ($null -eq $guiPath) {
    throw "Built GUI executable not found. Expected one of: $($guiPathCandidates -join ', ')"
}

Copy-RequiredFile -Source $cliPath -Destination (Join-Path $PackageDir 'filepilot.exe')
Copy-RequiredFile -Source $cliPath -Destination (Join-Path $PackageDir 'fp.exe')
Copy-RequiredFile -Source $guiPath -Destination (Join-Path $PackageDir 'filepilot-gui.exe')
Copy-RequiredFile -Source $resolvedCroc.Path -Destination (Join-Path $PackageDir 'backend\windows-amd64\croc.exe')
Copy-RequiredFile -Source (Join-Path $RepoRoot 'LICENSE') -Destination (Join-Path $PackageDir 'LICENSE')
Copy-RequiredFile -Source (Join-Path $RepoRoot 'THIRD_PARTY_NOTICES.md') -Destination (Join-Path $PackageDir 'THIRD_PARTY_NOTICES.md')
Copy-RequiredDirectory -Source (Join-Path $RepoRoot 'licenses') -Destination (Join-Path $PackageDir 'licenses')

Copy-OptionalFile -Source (Join-Path $RepoRoot 'scripts\install-cli.ps1') -Destination (Join-Path $PackageDir 'install-cli.ps1') | Out-Null
Copy-OptionalFile -Source (Join-Path $RepoRoot 'scripts\uninstall-cli.ps1') -Destination (Join-Path $PackageDir 'uninstall-cli.ps1') | Out-Null

if ($QuickstartPath -ne '') {
    Copy-RequiredFile -Source $QuickstartPath -Destination (Join-Path $PackageDir 'QUICKSTART.md')
}
elseif (-not (Copy-OptionalFile -Source (Join-Path $RepoRoot 'docs\release-quickstart.md') -Destination (Join-Path $PackageDir 'QUICKSTART.md'))) {
    @(
        '# FilePilot Quick Start',
        '',
        'This package was generated before the final release quickstart was added.',
        '',
        'GUI: run filepilot-gui.exe.',
        'CLI: run .\filepilot.exe send <path> or .\filepilot.exe receive <session-id> from this directory.',
        '',
        'Keep the sender window or terminal open until the receiver finishes.',
        '',
        'Release status: pending user-guide task.'
    ) | Set-Content -LiteralPath (Join-Path $PackageDir 'QUICKSTART.md') -Encoding ASCII
}

if ($NoticePath -ne '') {
    Copy-RequiredFile -Source $NoticePath -Destination (Join-Path $PackageDir 'NOTICE.md')
}
elseif (Copy-OptionalFile -Source (Join-Path $RepoRoot 'NOTICE') -Destination (Join-Path $PackageDir 'NOTICE.md')) {
}
elseif (-not (Copy-OptionalFile -Source (Join-Path $RepoRoot 'docs\release-notice-template.md') -Destination (Join-Path $PackageDir 'NOTICE.md'))) {
    @(
        '# FilePilot Notices',
        '',
        'Backend provenance, license notices, and final publication approval are pending human review.',
        'Do not publish this package until NOTICE.md has been reviewed for the selected backend binary.'
    ) | Set-Content -LiteralPath (Join-Path $PackageDir 'NOTICE.md') -Encoding ASCII
}

if ($BackendVersion -eq '') {
    $BackendVersion = Detect-BackendVersion -Path $resolvedCroc.Path
}
if ($BackendLicense -eq '') {
    $BackendLicense = 'pending-human-review'
}
if ($BackendLicenseUrl -eq '') {
    $BackendLicenseUrl = 'pending-human-review'
}

if ($AcceptanceStatus -eq 'passed') {
    if ($BackendVersion -eq '' -or $BackendVersion -eq 'unknown') {
        throw 'AcceptanceStatus passed requires an explicit reviewed -BackendVersion.'
    }
    if ($BackendLicense -eq '' -or $BackendLicense -eq 'pending-human-review') {
        throw 'AcceptanceStatus passed requires an explicit reviewed -BackendLicense.'
    }
    if ($BackendLicenseUrl -eq '' -or $BackendLicenseUrl -eq 'pending-human-review') {
        throw 'AcceptanceStatus passed requires an explicit reviewed -BackendLicenseUrl.'
    }
    if ($NoticePath -eq '') {
        throw 'AcceptanceStatus passed requires an explicit reviewed -NoticePath.'
    }
}

$backendPackagePath = 'backend/windows-amd64/croc.exe'
$backendHash = (Get-FileHash -LiteralPath (Join-Path $PackageDir $backendPackagePath) -Algorithm SHA256).Hash.ToLowerInvariant()
$gitCommit = 'unknown'
try {
    Push-Location $RepoRoot
    $gitCommit = (git rev-parse HEAD).Trim()
}
catch {
    $gitCommit = 'unknown'
}
finally {
    Pop-Location
}

$fileRecords = @()
$contentFiles = Get-ChildItem -LiteralPath $PackageDir -Recurse -File |
    Where-Object { $_.Name -ne 'checksums.txt' -and $_.Name -ne 'release-manifest.json' } |
    Sort-Object FullName
foreach ($file in $contentFiles) {
    $relative = Get-RelativePath -Root $PackageDir -Path $file.FullName
    $fileRecords += [ordered]@{
        path = $relative
        size_bytes = $file.Length
        sha256 = (Get-FileHash -LiteralPath $file.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
    }
}

$manifest = [ordered]@{
    schema_version = 1
    filepilot_version = $Version
    target_platform = $Platform
    package_name = $ArchiveName
    package_format = 'zip'
    build_time_utc = (Get-Date).ToUniversalTime().ToString('o')
    git_commit = $gitCommit
    release_acceptance_status = $AcceptanceStatus
    backend = [ordered]@{
        name = 'croc'
        version = $BackendVersion
        source = $BackendSource
        license = $BackendLicense
        license_url = $BackendLicenseUrl
        package_path = $backendPackagePath
        sha256 = $backendHash
    }
    files = $fileRecords
    generated_files = @('checksums.txt', 'release-manifest.json')
}
$manifest | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath (Join-Path $PackageDir 'release-manifest.json') -Encoding UTF8

$checksumLines = @()
$checksumFiles = Get-ChildItem -LiteralPath $PackageDir -Recurse -File |
    Where-Object { $_.Name -ne 'checksums.txt' } |
    Sort-Object FullName
foreach ($file in $checksumFiles) {
    $relative = Get-RelativePath -Root $PackageDir -Path $file.FullName
    $hash = (Get-FileHash -LiteralPath $file.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
    $checksumLines += "$hash  $relative"
}
$checksumLines | Set-Content -LiteralPath (Join-Path $PackageDir 'checksums.txt') -Encoding ASCII

Compress-Archive -LiteralPath $PackageDir -DestinationPath $ArchivePath -Force

Write-Host "Created release package: $ArchivePath"
Write-Host "Staged package directory: $PackageDir"
