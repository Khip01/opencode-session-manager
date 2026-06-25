# opencode-sm installer (PowerShell), for Windows
#
# Usage:
#   .\install.ps1                                    # install latest
#   .\install.ps1 -Version v0.1.0-alpha.1            # install specific version
#   .\install.ps1 -FromLocal .\opencode-sm.tar.gz    # install from local file
#   .\install.ps1 -Prefix $env:LOCALAPPDATA\Programs\opencode-sm  # custom location
#   .\install.ps1 -DryRun                            # show what would happen
#   Get-Help .\install.ps1 -Full

[CmdletBinding()]
param(
    [string]$Version = "",
    [string]$FromLocal = "",
    [string]$Prefix = "",
    [switch]$DryRun,
    [switch]$NoPathCheck,
    [switch]$Help
)

$ErrorActionPreference = "Stop"

$Repo = "Khip01/opencode-session-manager"
$Binary = "opencode-sm"
$GitHubApi = "https://api.github.com"

function Show-Help {
    @"
opencode-sm installer (PowerShell)

USAGE:
  .\install.ps1 [options]

OPTIONS:
  -Version VERSION       Install specific version (e.g. v0.1.0-alpha.1)
  -FromLocal PATH        Install from local tarball instead of downloading
  -Prefix DIR            Install to DIR (default: auto-detect)
  -DryRun                Show what would happen without installing
  -NoPathCheck           Skip PATH warning if install dir not in PATH
  -Help                  Show this help

EXAMPLES:
  .\install.ps1                                       # Install latest from GitHub
  .\install.ps1 -Version v0.1.0-alpha.1               # Install specific version
  .\install.ps1 -FromLocal .\opencode-sm.tar.gz        # Install from local file
  .\install.ps1 -Prefix \$env:LOCALAPPDATA\Programs\opencode-sm

"@
}

function Log($msg) { Write-Host $msg }
function Err($msg)  { Write-Host "error: $msg" -ForegroundColor Red }
function Die($msg) { Err $msg; exit 1 }

function Resolve-LatestVersion {
    try {
        # /releases (plural) returns ALL non-draft releases including
        # pre-releases, sorted by created_at desc. We pick the first.
        # Note: /releases/latest ignores pre-releases and returns 404
        # when only alpha or beta tags exist, which broke install
        # for users on a repository that has not yet cut a stable
        # release.
        $releases = Invoke-RestMethod -Uri "$GitHubApi/repos/$Repo/releases?per_page=20" -ErrorAction Stop
        if (-not $releases -or $releases.Count -eq 0) {
            throw "no releases found"
        }
        $latest = $releases[0].tag_name
        if (-not $latest) { throw "first release has no tag_name" }
        return $latest
    } catch {
        Die "could not determine latest version from GitHub: $_"
    }
}

function Build-Url {
    param([string]$v)
    $vStripped = $v -replace '^v',''
    $osName = if ($IsWindows -or $env:OS -eq "Windows_NT") { "Windows" } else { "Unknown" }
    return "https://github.com/$Repo/releases/download/$v/opencode-session-manager_${vStripped}_${osName}_amd64.tar.gz"
}

function Build-ChecksumUrl {
    param([string]$v)
    $vStripped = $v -replace '^v',''
    return "https://github.com/$Repo/releases/download/$v/opencode-session-manager_${vStripped}_checksums.txt"
}

function Test-IsWritableDir($dir) {
    if (-not (Test-Path -LiteralPath $dir -PathType Container)) { return $false }
    try {
        $test = [System.IO.Path]::Combine($dir, ".write-test-$PID")
        [System.IO.File]::Create($test).Close()
        Remove-Item -LiteralPath $test -Force -ErrorAction SilentlyContinue
        return $true
    } catch {
        return $false
    }
}

function Determine-InstallDir {
    if ($Prefix) {
        try {
            New-Item -ItemType Directory -Force -Path $Prefix -ErrorAction Stop | Out-Null
        } catch {
            Die "cannot create $Prefix (permission denied); try a different -Prefix"
        }
        if (-not (Test-IsWritableDir $Prefix)) {
            Die "$Prefix is not writable by current user; try a different -Prefix"
        }
        return $Prefix
    }

    $candidates = @(
        "$env:LOCALAPPDATA\Programs\opencode-sm",
        "$env:USERPROFILE\bin"
    )

    foreach ($dir in $candidates) {
        $pathDirs = $env:PATH -split ';' | ForEach-Object { $_.TrimEnd('\').ToLowerInvariant() }
        if ($pathDirs -contains $dir.TrimEnd('\').ToLowerInvariant()) {
            New-Item -ItemType Directory -Force -Path $dir -ErrorAction SilentlyContinue | Out-Null
            if (Test-IsWritableDir $dir) {
                return $dir
            }
        }
    }

    $fallback = "$env:LOCALAPPDATA\Programs\opencode-sm"
    New-Item -ItemType Directory -Force -Path $fallback | Out-Null
    if (-not (Test-IsWritableDir $fallback)) {
        Die "no writable install dir found on PATH; retry with -Prefix DIR for a writable location"
    }
    return $fallback
}

function Test-InPath($dir) {
    $pathDirs = $env:PATH -split ';' | ForEach-Object { $_.TrimEnd('\').ToLowerInvariant() }
    return $pathDirs -contains $dir.TrimEnd('\').ToLowerInvariant()
}

function Get-Sha256($path) {
    $h = [System.Security.Cryptography.SHA256]::Create()
    try {
        $stream = [System.IO.File]::OpenRead($path)
        try {
            $hashBytes = $h.ComputeHash($stream)
            return ([BitConverter]::ToString($hashBytes) -replace '-', '').ToLowerInvariant()
        } finally { $stream.Dispose() }
    } finally { $h.Dispose() }
}

function Test-Checksum {
    param([string]$archive, [string]$checksumFile, [string]$archiveName)

    if (-not (Test-Path $checksumFile)) {
        Log "checksums file not available, skipping verification"
        return
    }

    $expected = $null
    foreach ($line in Get-Content $checksumFile) {
        $parts = $line -split '\s+', 2
        if ($parts.Count -ne 2) { continue }
        $hash = $parts[0].Trim().ToLowerInvariant()
        $name = $parts[1].Trim()
        if ($name -eq $archiveName) {
            $expected = $hash
            break
        }
    }

    if (-not $expected) {
        Log "no checksum entry for $archiveName, skipping verification"
        return
    }

    $actual = Get-Sha256 $archive
    if ($expected -ne $actual) {
        Err "checksum mismatch for $archiveName"
        Err "  expected: $expected"
        Err "  actual:   $actual"
        throw "checksum verification failed"
    }
    Log "checksum verified (sha256)"
}

function Install-Binary {
    param([string]$archive, [string]$installDir, [string]$binaryName)

    $target = Join-Path $installDir $binaryName
    Log "installing to $target"

    if ($DryRun) {
        Log "[dry-run] would copy $archive -> $target"
        return
    }

    if (Test-Path $target) {
        Log "existing installation found, replacing"
    }

    Copy-Item -Path $archive -Destination $target -Force
    Log "installed: $target"
}

function Warn-Path {
    param([string]$installDir)

    if ($NoPathCheck) { return }
    if (Test-InPath $installDir) { return }

    Log ""
    Log "WARNING: $installDir is not in your PATH"
    Log ""
    Log "Add it to PATH (PowerShell, current session):"
    Log "  \$env:PATH = '$installDir;' + \$env:PATH"
    Log ""
    Log "Persist for future sessions:"
    Log "  [Environment]::SetEnvironmentVariable('Path', $env:PATH + ';$installDir', 'User')"
    Log ""
    Log "Or run opencode-sm with full path:"
    Log "  $installDir\$Binary --version"
}

if ($Help) { Show-Help; exit 0 }

Log "opencode-sm installer (Windows)"
Log ""

if (-not $Version -and -not $FromLocal) {
    $Version = Resolve-LatestVersion
}
if (-not $Version) { Die "version is required" }

$tmpdir = Join-Path $env:TEMP "opencode-sm-install-$([System.Guid]::NewGuid().ToString('N').Substring(0,8))"
New-Item -ItemType Directory -Force -Path $tmpdir | Out-Null

try {
    $archive = Join-Path $tmpdir "opencode-sm.tar.gz"
    $checksumFile = Join-Path $tmpdir "checksums.txt"

    if ($FromLocal) {
        if (-not (Test-Path $FromLocal)) {
            Die "local file not found: $FromLocal"
        }
        Log "using local file: $FromLocal"
        Copy-Item -Path $FromLocal -Destination $archive
    } else {
        $url = Build-Url $Version
        Log "downloading $url"
        try {
            Invoke-WebRequest -Uri $url -OutFile $archive -UseBasicParsing
        } catch {
            Die "download failed: $_"
        }

        $csUrl = Build-ChecksumUrl $Version
        try {
            Invoke-WebRequest -Uri $csUrl -OutFile $checksumFile -UseBasicParsing
        } catch {
            Log "checksums file not available, skipping verification"
        }

        $archiveName = Split-Path -Leaf $url
        try { Test-Checksum $archive $checksumFile $archiveName }
        catch { Die $_ }
    }

    Log "extracting archive"
    $extractDir = Join-Path $tmpdir "extracted"
    New-Item -ItemType Directory -Force -Path $extractDir | Out-Null
    if ($IsWindows -or $env:OS -eq "Windows_NT") {
        # tar.exe is built-in on Windows 10+ (1809+)
        tar -xzf $archive -C $extractDir
    } else {
        tar -xzf $archive -C $extractDir
    }

    $binaries = Get-ChildItem -Path $extractDir -Recurse -Filter $Binary -File
    $extracted = $binaries | Select-Object -First 1
    if (-not $extracted) { Die "binary not found in archive" }

    $installDir = Determine-InstallDir
    Install-Binary -archive $extracted.FullName -installDir $installDir -binaryName $Binary

    Log ""
    Log "opencode-sm $Version installed to $installDir\$Binary"
    Log ""
    if (Test-InPath $installDir) {
        Log "Run:"
        Log "  $Binary --version"
    } else {
        Warn-Path $installDir
    }
} finally {
    if (Test-Path $tmpdir) {
        Remove-Item -Recurse -Force $tmpdir -ErrorAction SilentlyContinue
    }
}
