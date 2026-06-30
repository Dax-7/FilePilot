param()

$ErrorActionPreference = 'Stop'

$PackageDir = [System.IO.Path]::GetFullPath($PSScriptRoot)

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
if ($null -eq $userPath -or $userPath.Trim() -eq '') {
    Write-Host "FilePilot CLI is not registered in the current user's PATH."
    exit 0
}

$removed = 0
$remaining = @()
foreach ($entry in ($userPath -split ';')) {
    $trimmed = $entry.Trim()
    if ($trimmed -eq '') {
        continue
    }

    if ([string]::Equals((Normalize-PathEntry -Entry $trimmed), $target, [System.StringComparison]::OrdinalIgnoreCase)) {
        $removed++
        continue
    }

    $remaining += $trimmed
}

if ($removed -eq 0) {
    Write-Host "FilePilot CLI registration for this directory was not found:"
    Write-Host "  $PackageDir"
    exit 0
}

[Environment]::SetEnvironmentVariable('Path', ($remaining -join ';'), 'User')

Write-Host "Removed FilePilot CLI registration for the current user:"
Write-Host "  $PackageDir"
Write-Host "Open a new terminal for PATH changes to take effect."
