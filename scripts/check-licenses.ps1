param()

$ErrorActionPreference = 'Stop'
$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot '..')

$requiredFiles = @(
    'LICENSE',
    'NOTICE',
    'THIRD_PARTY_NOTICES.md',
    'licenses/croc-MIT-LICENSE.txt'
)

foreach ($relative in $requiredFiles) {
    $path = Join-Path $RepoRoot ($relative -replace '/', [System.IO.Path]::DirectorySeparatorChar)
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "Required license file missing: $relative"
    }
    Write-Host "[PASS] Found $relative"
}

Push-Location $RepoRoot
try {
    if (Get-Command go -ErrorAction SilentlyContinue) {
        Write-Host "[INFO] Go modules in this build:"
        try {
            $goOutput = & go list -m all 2>&1
            $goExitCode = $LASTEXITCODE
        }
        catch {
            $goOutput = $_
            $goExitCode = 1
        }
        if ($goExitCode -ne 0) {
            Write-Host "[WARN] go list -m all failed; dependency license review remains manual for this run."
        }
        $goOutput | ForEach-Object { Write-Host $_ }
    }
    else {
        Write-Host "[SKIP] go not found; skipping Go module listing."
    }

    $packageLock = Join-Path $RepoRoot 'cmd/filepilot-gui/frontend/package-lock.json'
    if (Test-Path -LiteralPath $packageLock -PathType Leaf) {
        Write-Host "[INFO] npm packages recorded in package-lock.json:"
        Select-String -LiteralPath $packageLock -Pattern '"node_modules/[^"]+"' |
            ForEach-Object {
                if ($_.Line -match '"(node_modules/[^"]+)"') {
                    Write-Host $Matches[1]
                }
            }
    }
    else {
        Write-Host "[SKIP] package-lock.json not found; skipping npm package listing."
    }
}
finally {
    Pop-Location
}
