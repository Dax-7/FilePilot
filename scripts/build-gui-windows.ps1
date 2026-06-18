$ErrorActionPreference = 'Stop'

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = Resolve-Path (Join-Path $ScriptDir '..')
$GuiDir = Join-Path $RepoRoot 'cmd/filepilot-gui'

Push-Location $GuiDir
try {
    Push-Location 'frontend'
    try {
        npm install
    }
    finally {
        Pop-Location
    }
    wails build
}
finally {
    Pop-Location
}
