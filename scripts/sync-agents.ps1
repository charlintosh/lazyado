# Sync .agents/skills/ to all supported folders via symlinks
# Source: .agents/skills/
# Targets: folders listed in .agents.support.json

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent $ScriptDir
$ConfigFile = Join-Path $ProjectRoot ".agents.support.json"
$SourceDir = Join-Path $ProjectRoot ".agents\skills"

Write-Host "Syncing .agents/skills/ to supported folders..." -ForegroundColor Cyan
Write-Host ""

# Check if config file exists
if (-not (Test-Path $ConfigFile)) {
    Write-Host "Error: $ConfigFile not found" -ForegroundColor Red
    exit 1
}

# Check if source directory exists
if (-not (Test-Path $SourceDir)) {
    Write-Host "Error: $SourceDir not found" -ForegroundColor Red
    exit 1
}

# Parse folders from JSON
$Config = Get-Content $ConfigFile | ConvertFrom-Json
$Folders = $Config.folders

foreach ($folder in $Folders) {
    $TargetParent = Join-Path $ProjectRoot $folder
    $TargetDir = Join-Path $TargetParent "skills"

    Write-Host "  $folder/skills/ " -NoNewline

    # Create parent folder if it doesn't exist
    if (-not (Test-Path $TargetParent)) {
        New-Item -ItemType Directory -Path $TargetParent -Force | Out-Null
        Write-Host "(created parent) " -NoNewline
    }

    # Remove existing directory or symlink
    if (Test-Path $TargetDir) {
        $item = Get-Item $TargetDir -Force
        if ($item.LinkType -eq "SymbolicLink") {
            Remove-Item $TargetDir -Force
            Write-Host "(removed old symlink) " -NoNewline
        } else {
            Remove-Item $TargetDir -Recurse -Force
            Write-Host "(removed old directory) " -NoNewline
        }
    }

    # Create symlink (requires admin on Windows or Developer Mode enabled)
    try {
        New-Item -ItemType SymbolicLink -Path $TargetDir -Target $SourceDir -Force | Out-Null
        Write-Host "-> symlinked" -ForegroundColor Green
    } catch {
        Write-Host "-> FAILED (run as admin or enable Developer Mode)" -ForegroundColor Red
        Write-Host "   Error: $_" -ForegroundColor Yellow
    }
}

Write-Host ""
Write-Host "Done! All skills folders are now symlinked to .agents/skills/" -ForegroundColor Green
