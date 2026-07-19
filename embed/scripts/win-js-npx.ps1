# Windows Node.js LTS Installation Script
# Prefer standard installers (winget → MSI), zip only as user-local fallback.
# Usage:
#   .\win-js-npx.ps1                # Default: mainland (npm mirror after install)
#   .\win-js-npx.ps1 -mode standard # Official npm registry

param(
    [string]$mode = "mainland"
)

$ErrorActionPreference = "Stop"

$NodeVersion = "20.12.2"
$Arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") {
    "arm64"
} elseif ([System.Environment]::Is64BitOperatingSystem) {
    "x64"
} else {
    "x86"
}

# Official MSI arch names: x64 / arm64 / x86
$MsiFile = "node-v$NodeVersion-$Arch.msi"
$ZipFile = "node-v$NodeVersion-win-$Arch.zip"
# User-local portable fallback (never C:\nodejs)
$PortableHome = Join-Path $env:LOCALAPPDATA "Programs\nodejs"

$TempDir = New-Item -ItemType Directory -Path ([System.IO.Path]::GetTempPath() + [System.Guid]::NewGuid().ToString())

$msi_mainland = @(
    "https://npmmirror.com/mirrors/node/v$NodeVersion/$MsiFile",
    "https://mirrors.huaweicloud.com/nodejs/release/v$NodeVersion/$MsiFile",
    "https://nodejs.org/dist/v$NodeVersion/$MsiFile"
)
$msi_standard = @(
    "https://nodejs.org/dist/v$NodeVersion/$MsiFile",
    "https://npmmirror.com/mirrors/node/v$NodeVersion/$MsiFile",
    "https://mirrors.huaweicloud.com/nodejs/release/v$NodeVersion/$MsiFile"
)
$zip_mainland = @(
    "https://mirrors.huaweicloud.com/nodejs/release/v$NodeVersion/$ZipFile",
    "https://npmmirror.com/mirrors/node/v$NodeVersion/$ZipFile",
    "https://nodejs.org/dist/v$NodeVersion/$ZipFile"
)
$zip_standard = @(
    "https://nodejs.org/dist/v$NodeVersion/$ZipFile",
    "https://mirrors.huaweicloud.com/nodejs/release/v$NodeVersion/$ZipFile",
    "https://npmmirror.com/mirrors/node/v$NodeVersion/$ZipFile"
)

if ($mode -eq "standard") {
    $MsiUrls = $msi_standard
    $ZipUrls = $zip_standard
    $npmMirror = $false
} else {
    $MsiUrls = $msi_mainland
    $ZipUrls = $zip_mainland
    $npmMirror = $true
}

function Write-Color($Text, $Color) {
    Write-Host $Text -ForegroundColor $Color
}

function Refresh-Path {
    $machine = [System.Environment]::GetEnvironmentVariable("Path", "Machine")
    $user = [System.Environment]::GetEnvironmentVariable("Path", "User")
    $env:Path = (@($PortableHome, $machine, $user) | Where-Object { $_ }) -join ";"
}

function Test-NodeReady {
    Refresh-Path
    return [bool](Get-Command node -ErrorAction SilentlyContinue) -and
           [bool](Get-Command npm -ErrorAction SilentlyContinue)
}

function Check-Node {
    if (Test-NodeReady) {
        Write-Color ("Node.js detected: " + (node --version)) Yellow
        return $true
    }
    Write-Color "Node.js not detected, will install..." Yellow
    return $false
}

function Install-ViaWinget {
    if (-not (Get-Command winget -ErrorAction SilentlyContinue)) {
        Write-Color "winget not available, skip." Yellow
        return $false
    }
    Write-Color "Installing Node.js LTS via winget (OpenJS.NodeJS.LTS)..." Cyan
    $args = @(
        "install", "-e", "--id", "OpenJS.NodeJS.LTS",
        "--accept-package-agreements", "--accept-source-agreements",
        "--disable-interactivity"
    )
    $p = Start-Process -FilePath "winget" -ArgumentList $args `
        -Wait -PassThru -WindowStyle Hidden
    Refresh-Path
    if (Test-NodeReady) {
        Write-Color "winget install succeeded." Green
        return $true
    }
    Write-Color ("winget finished with exit code " + $p.ExitCode + ", node still missing.") Yellow
    return $false
}

function Download-File([string[]]$Urls, [string]$OutFile) {
    foreach ($url in $Urls) {
        Write-Color "Trying: $url" Cyan
        try {
            Invoke-WebRequest -Uri $url -OutFile $OutFile -UseBasicParsing -ErrorAction Stop
            Write-Color "Download succeeded!" Green
            return $true
        } catch {
            Write-Color "Download failed for this mirror." Red
        }
    }
    return $false
}

function Install-ViaMsi {
    Write-Color "Installing Node.js via official MSI (silent)..." Cyan
    $msiPath = Join-Path $TempDir.FullName $MsiFile
    if (-not (Download-File $MsiUrls $msiPath)) {
        return $false
    }
    # Per-machine silent install to Program Files\nodejs (standard layout).
    $p = Start-Process -FilePath "msiexec.exe" -ArgumentList @(
        "/i", "`"$msiPath`"", "/qn", "/norestart"
    ) -Wait -PassThru -WindowStyle Hidden
    Refresh-Path
    if (Test-NodeReady) {
        Write-Color "MSI install succeeded." Green
        return $true
    }
    Write-Color ("MSI finished with exit code " + $p.ExitCode + " (may need admin).") Yellow
    return $false
}

function Install-ViaZipPortable {
    Write-Color "Falling back to user-local portable install: $PortableHome" Cyan
    $zipPath = Join-Path $TempDir.FullName $ZipFile
    if (-not (Download-File $ZipUrls $zipPath)) {
        Write-Color "All downloads failed. Install manually: https://nodejs.org/en/download" Red
        exit 1
    }

    $extractRoot = Join-Path $TempDir.FullName "extract"
    New-Item -ItemType Directory -Path $extractRoot -Force | Out-Null
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    try {
        [System.IO.Compression.ZipFile]::ExtractToDirectory($zipPath, $extractRoot)
    } catch {
        Write-Color "Extract failed: $_" Red
        exit 1
    }

    $inner = Get-ChildItem -Path $extractRoot -Directory | Select-Object -First 1
    if (-not $inner) {
        Write-Color "Extract failed: unexpected zip layout" Red
        exit 1
    }

    if (Test-Path $PortableHome) {
        Remove-Item -LiteralPath $PortableHome -Recurse -Force
    }
    New-Item -ItemType Directory -Path $PortableHome -Force | Out-Null
    Copy-Item -Path (Join-Path $inner.FullName "*") -Destination $PortableHome -Recurse -Force

    if (-not (Test-Path (Join-Path $PortableHome "node.exe"))) {
        Write-Color "Install failed: node.exe missing under $PortableHome" Red
        exit 1
    }

    $envPath = [System.Environment]::GetEnvironmentVariable("Path", "User")
    if ([string]::IsNullOrEmpty($envPath)) { $envPath = "" }
    $parts = @($envPath -split ";" | Where-Object {
        $_ -and ($_ -ne $PortableHome) -and ($_ -ne "C:\nodejs") -and ($_ -notlike "C:\nodejs\*")
    })
    $newPath = (@($PortableHome) + $parts) -join ";"
    [System.Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    Refresh-Path

    if (-not (Test-NodeReady)) {
        Write-Color "Portable install finished but node/npm not on PATH." Red
        exit 1
    }
    Write-Color "Portable Node.js ready at $PortableHome" Green
    return $true
}

function Configure-Npm {
    Write-Color "Configuring npm registry to npmmirror..." Cyan
    npm config set registry https://registry.npmmirror.com
    Write-Color "npm configured to use npmmirror" Green
}

try {
    if (-not (Check-Node)) {
        $ok = $false
        if (-not $ok) { $ok = Install-ViaWinget }
        if (-not $ok) { $ok = Install-ViaMsi }
        if (-not $ok) { $ok = Install-ViaZipPortable }
        if (-not $ok) {
            Write-Color "Node.js installation failed." Red
            exit 1
        }
    }

    Refresh-Path
    if (Test-NodeReady) {
        if ($npmMirror) { Configure-Npm }
        Write-Color "Node.js and npm are ready!" Cyan
        node -v
        npm -v
        if (Get-Command npx -ErrorAction SilentlyContinue) { npx -v }
    } else {
        Write-Color "npm not detected after install." Red
        exit 1
    }

    if (Test-Path "C:\nodejs") {
        Write-Color "Note: legacy C:\nodejs exists from an older install; you can delete it if unused." Yellow
    }
} finally {
    if (Test-Path $TempDir) {
        Remove-Item -Recurse -Force $TempDir -ErrorAction SilentlyContinue
    }
}
