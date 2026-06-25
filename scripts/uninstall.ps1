# opencode-sm uninstaller (PowerShell), for Windows
#
# Usage:
#   .\uninstall.ps1                     # remove from standard locations
#   .\uninstall.ps1 -Prefix DIR         # remove from specific directory
#   .\uninstall.ps1 -DryRun             # show what would be removed
#   .\uninstall.ps1 -Purge              # also remove user config

[CmdletBinding()]
param(
    [string]$Prefix = "",
    [switch]$DryRun,
    [switch]$Purge,
    [switch]$Help
)

$ErrorActionPreference = "Stop"

$Binary = "opencode-sm"

function Show-Help {
    @"
opencode-sm uninstaller (PowerShell)

USAGE:
  .\uninstall.ps1 [options]

OPTIONS:
  -Prefix DIR            Remove from DIR instead of scanning standard locations
  -Purge                 Also remove user config (if any)
  -DryRun                Show what would be removed without removing
  -Help                  Show this help

BEHAVIOR:
  If the opencode-sm binary is on PATH, this script delegates to
  'opencode-sm uninstall' for a single source of truth. Otherwise it
  falls back to a local cleanup that scans standard install locations.

REMOVED FILES:
  - The opencode-sm binary from the install location
  - (with -Purge) %LOCALAPPDATA%\opencode-sm\ if present

NOTE:
  This script does NOT remove:
  - Backups created by opencode-sm itself
  - opencode.db or any OpenCode data

"@
}

function Log($msg) { Write-Host $msg }
function Err($msg)  { Write-Host "error: $msg" -ForegroundColor Red }

if ($Help) { Show-Help; exit 0 }

# If the opencode-sm binary is on PATH, delegate to its built-in
# uninstall subcommand. This avoids re-implementing the logic in
# PowerShell and keeps a single source of truth.
if (-not $DryRun -and (Get-Command "opencode-sm" -ErrorAction SilentlyContinue)) {
    $delegateArgs = @("uninstall")
    if ($Prefix) { $delegateArgs += @("--prefix", $Prefix) }
    if ($Purge)  { $delegateArgs += @("--purge") }
    Log "delegating to opencode-sm $($delegateArgs -join ' ')"
    & opencode-sm @delegateArgs
    exit $LASTEXITCODE
}

Log "opencode-sm uninstaller (Windows, script fallback)"
Log ""

if ($DryRun) {
    Log "(DRY RUN, nothing will be removed)"
    Log ""
}

function Remove-BinaryFile($path) {
    if (-not (Test-Path $path)) { return 0 }
    if ($DryRun) {
        Log "[dry-run] would remove: $path"
        return 1
    }
    try {
        Remove-Item -Path $path -Force -ErrorAction Stop
        Log "removed: $path"
        return 1
    } catch {
        Err "cannot remove $path: $_"
        return 0
    }
}

function Remove-DirIfEmpty($dir) {
    if (-not (Test-Path $dir)) { return }
    $items = Get-ChildItem -Path $dir -Force -ErrorAction SilentlyContinue
    if ($items.Count -eq 0) {
        if ($DryRun) {
            Log "[dry-run] would remove empty dir: $dir"
        } else {
            try { Remove-Item -Path $dir -Force -ErrorAction Stop } catch {}
            Log "removed empty dir: $dir"
        }
    }
}

function Purge-Config {
    $configDir = Join-Path $env:LOCALAPPDATA "opencode-sm"
    if (Test-Path $configDir) {
        if ($DryRun) {
            Log "[dry-run] would purge: $configDir"
        } else {
            Remove-Item -Path $configDir -Recurse -Force -ErrorAction Stop
            Log "purged: $configDir"
        }
    } else {
        Log "no user config at $configDir"
    }
}

$removed = 0

if ($Prefix) {
    $target = Join-Path $Prefix $Binary
    if (Test-Path $target) {
        $removed += Remove-BinaryFile $target
    } else {
        Err "no opencode-sm found at $target"
    }
} else {
    $candidates = @(
        (Join-Path $env:LOCALAPPDATA "Programs\opencode-sm\$Binary"),
        (Join-Path $env:LOCALAPPDATA "Programs\opencode-sm"),
        "$env:LOCALAPPDATA\Microsoft\WindowsApps\$Binary",
        "$env:USERPROFILE\bin\$Binary"
    )

    foreach ($path in $candidates) {
        if (Test-Path $path -PathType Leaf) {
            $removed += Remove-BinaryFile $path
        }
    }

    if ($removed -gt 0 -and -not $Prefix) {
        $parentDir = Join-Path $env:LOCALAPPDATA "Programs\opencode-sm"
        Remove-DirIfEmpty $parentDir
    }
}

if ($Purge) { Purge-Config }

if (-not $DryRun) {
        Log ""
        Log "uninstall complete (script fallback)"
        Log ""
        Log "Note: this script does not touch:"
        Log "  - Backups (*.opencode-sm-backup) in the same dir as opencode.db"
        Log "  - opencode.db or any OpenCode data"
    }
}
