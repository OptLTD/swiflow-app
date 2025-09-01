# Windows Node.js LTS Installation Script (mainland/standard merged version)
# Usage:
#   .\win-js-npx.ps1                # Default: mainland (China mirror priority)
#   .\win-js-npx.ps1 -mode standard # Official source priority

param(
    [string]$mode = "mainland"
)

$NodeVersion = "20.12.2"
$Arch = if ([System.Environment]::Is64BitOperatingSystem) {
    if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "x64" }
} else {
    "x86"
}
$NodeDistros = @{
    "x64"   = "node-v$NodeVersion-win-x64.zip"
    "arm64" = "node-v$NodeVersion-win-arm64.zip"
    "x86"   = "node-v$NodeVersion-win-x86.zip"
}
$NodeFile = $NodeDistros[$Arch]
$TempDir = New-Item -ItemType Directory -Path ([System.IO.Path]::GetTempPath() + [System.Guid]::NewGuid().ToString())

$urls_mainland = @(
    "https://mirrors.huaweicloud.com/nodejs/release/v$NodeVersion/$NodeFile",
    "https://npmmirror.com/mirrors/node/v$NodeVersion/$NodeFile",
    "https://registry.npmmirror.com/-/binary/node/v$NodeVersion/$NodeFile",
    "https://nodejs.org/dist/v$NodeVersion/$NodeFile"
)
$urls_standard = @(
    "https://nodejs.org/dist/v$NodeVersion/$NodeFile",
    "https://mirrors.huaweicloud.com/nodejs/release/v$NodeVersion/$NodeFile",
    "https://npmmirror.com/mirrors/node/v$NodeVersion/$NodeFile",
    "https://registry.npmmirror.com/-/binary/node/v$NodeVersion/$NodeFile"
)

if ($mode -eq "standard") {
    $NodeUrls = $urls_standard
    $npmMirror = $false
} else {
    $NodeUrls = $urls_mainland
    $npmMirror = $true
}

function Write-Color($Text, $Color) {
    Write-Host $Text -ForegroundColor $Color
}

function Check-Node {
    if (Get-Command node -ErrorAction SilentlyContinue) {
        $ver = node --version
        Write-Color "Node.js detected: $ver" Yellow
        return $true
    } else {
        Write-Color "Node.js not detected, will install..." Yellow
        return $false
    }
}

function Download-Node {
    Write-Color "Attempting to download Node.js $NodeVersion ($Arch)..." Cyan
    foreach ($url in $NodeUrls) {
        Write-Color "Trying mirror: $url" Cyan
        try {
            Invoke-WebRequest -Uri $url -OutFile "$TempDir\$NodeFile" -UseBasicParsing -ErrorAction Stop
            Write-Color "Download succeeded!" Green
            return $true
        } catch {
            Write-Color "Download failed for this mirror." Red
        }
    }
    Write-Color "All mirror downloads failed, please download Node.js manually: https://nodejs.org/en/download" Red
    exit 1
}

function Install-Node {
    Write-Color "Extracting Node.js..." Cyan
    $dest = "C:\\nodejs"
    if (!(Test-Path $dest)) { New-Item -ItemType Directory -Path $dest | Out-Null }
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    [System.IO.Compression.ZipFile]::ExtractToDirectory("$TempDir\$NodeFile", $dest)
    # Configure PATH
    $envPath = [System.Environment]::GetEnvironmentVariable("Path", "User")
    if ($envPath -notlike "*C:\\nodejs\\*" ) {
        [System.Environment]::SetEnvironmentVariable("Path", "C:\\nodejs;" + $envPath, "User")
    }
    $env:Path = "C:\\nodejs;" + $env:Path
    Write-Color "Node.js $NodeVersion installation complete!" Green
}

function Configure-Npm {
    Write-Color "Configuring npm registry to Taobao mirror..." Cyan
    npm config set registry https://registry.npmmirror.com
    npm config set disturl https://npmmirror.com/mirrors/node
    Write-Color "npm configured to use Taobao mirror" Green
}

# Main flow
if (-not (Check-Node)) {
    if (Download-Node) {
        Install-Node
    }
}

if (Get-Command npm -ErrorAction SilentlyContinue) {
    if ($npmMirror) { Configure-Npm }
    Write-Color "Node.js and npm are ready!" Cyan
    node -v
    npm -v
} else {
    Write-Color "npm not detected, Node.js installation may have failed." Red
}

# Clean up
Remove-Item -Recurse -Force $TempDir 