param()

$ErrorActionPreference = 'Stop'

$PackageDir = [System.IO.Path]::GetFullPath($PSScriptRoot)
$RequiredCommands = @('filepilot.exe', 'fp.exe')

foreach ($command in $RequiredCommands) {
    $path = Join-Path $PackageDir $command
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "Required CLI executable not found next to this script: $command"
    }
}

function Normalize-PathEntry {
    param([Parameter(Mandatory = $true)][string]$Entry)

    $expanded = [Environment]::ExpandEnvironmentVariables($Entry.Trim().Trim('"'))
    if ($expanded -eq '') {
        return ''
    }

    try {
        return [System.IO.Path]::GetFullPath($expanded).TrimEnd('\', '/')
    }
    catch {
        return $expanded.TrimEnd('\', '/')
    }
}

$target = Normalize-PathEntry -Entry $PackageDir
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($null -eq $userPath) {
    $userPath = ''
}

$entries = @(
    $userPath -split ';' |
        ForEach-Object { $_.Trim() } |
        Where-Object { $_ -ne '' }
)

$alreadyRegistered = $false
foreach ($entry in $entries) {
    if ([string]::Equals((Normalize-PathEntry -Entry $entry), $target, [System.StringComparison]::OrdinalIgnoreCase)) {
        $alreadyRegistered = $true
        break
    }
}

if ($alreadyRegistered) {
    Write-Host "FilePilot CLI is already registered for the current user:"
    Write-Host "  $PackageDir"
    Write-Host "Open a new terminal before running filepilot or fp from another directory."
    exit 0
}

if ($entries.Count -eq 0) {
    $newUserPath = $PackageDir
}
else {
    $newUserPath = ($entries + $PackageDir) -join ';'
}

[Environment]::SetEnvironmentVariable('Path', $newUserPath, 'User')

Write-Host "Registered FilePilot CLI for the current user:"
Write-Host "  $PackageDir"
Write-Host "Open a new terminal before running filepilot or fp from another directory."
