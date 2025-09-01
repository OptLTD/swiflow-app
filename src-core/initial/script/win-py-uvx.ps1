param(
    [string]$mode = "mainland"
)

$pythonVersion = "3.12.2"
$pythonInstaller = "$env:TEMP\python-$pythonVersion-amd64.exe"
$skipPythonInstall = $false

$mirrors_mainland = @(
    @{ Name = "HUAWEICLOUD"; Url = "https://mirrors.huaweicloud.com/python/$pythonVersion/python-$pythonVersion-amd64.exe" },
    @{ Name = "NPMMIRROR"; Url = "https://registry.npmmirror.com/-/binary/python/$pythonVersion/python-$pythonVersion-amd64.exe" },
    @{ Name = "PYTHON.ORG"; Url = "https://www.python.org/ftp/python/$pythonVersion/python-$pythonVersion-amd64.exe" }
)
$mirrors_standard = @(
    @{ Name = "PYTHON.ORG"; Url = "https://www.python.org/ftp/python/$pythonVersion/python-$pythonVersion-amd64.exe" },
    @{ Name = "HUAWEICLOUD"; Url = "https://mirrors.huaweicloud.com/python/$pythonVersion/python-$pythonVersion-amd64.exe" },
    @{ Name = "NPMMIRROR"; Url = "https://registry.npmmirror.com/-/binary/python/$pythonVersion/python-$pythonVersion-amd64.exe" }
)

if ($mode -eq "standard") {
    $mirrors = $mirrors_standard
    $pipConf = $null
    $uvConf = $null
} else {
    $mirrors = $mirrors_mainland
    $pipConf = @"
[global]
index-url = https://pypi.tuna.tsinghua.edu.cn/simple
trusted-host = pypi.tuna.tsinghua.edu.cn
"@
    $uvConf = @"
[global]
index-url = https://pypi.tuna.tsinghua.edu.cn/simple
"@
}

Write-Host "Starting installation of Python $pythonVersion and UV package manager..." -ForegroundColor Green

# Always install Python regardless of existing installation
$downloadSuccess = $false
foreach ($mirror in $mirrors) {
    Write-Host "Attempting download from $($mirror.Name): $($mirror.Url)" -ForegroundColor Blue
    try {
        Invoke-WebRequest -Uri $mirror.Url -OutFile $pythonInstaller -UseBasicParsing
        Write-Host "Download completed successfully from $($mirror.Name)!" -ForegroundColor Green
        $downloadSuccess = $true
        break
    } catch {
        Write-Host "Download failed from $($mirror.Name): $_" -ForegroundColor Red
    }
}
if (-not $downloadSuccess) {
    Write-Host "All download attempts failed. Please download Python manually from https://www.python.org/downloads/" -ForegroundColor Red
    exit 1
}

Write-Host "Installing Python $pythonVersion..." -ForegroundColor Blue
try {
    Start-Process -FilePath $pythonInstaller -ArgumentList "/quiet", "InstallAllUsers=0", "PrependPath=1", "Include_test=0", "Include_pip=1" -Wait
    Remove-Item $pythonInstaller -Force
} catch {
    Write-Host "Failed to install Python: $_" -ForegroundColor Red
    exit 1
}
Write-Host "Python $pythonVersion installation complete!" -ForegroundColor Green
$env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")

# Configure pip mirror (mainland mode only)
if ($pipConf) {
    Write-Host "Configuring pip to use China mirrors..." -ForegroundColor Blue
    try {
        $pipConfigDir = "$env:APPDATA\pip"
        if (-not (Test-Path $pipConfigDir)) { New-Item -ItemType Directory -Path $pipConfigDir -Force | Out-Null }
        $pipConf | Out-File -FilePath "$pipConfigDir\pip.conf" -Encoding utf8 -Force
        Write-Host "Pip configured to use TUNA mirror" -ForegroundColor Green
    } catch {
        Write-Host "Failed to configure pip mirror: $_" -ForegroundColor Red
    }
}

# Install UV
Write-Host "Installing UV package manager..." -ForegroundColor Blue
try {
    if ($pipConf) {
        python -m pip install uv -i https://pypi.tuna.tsinghua.edu.cn/simple
    } else {
        python -m pip install uv
    }

    if ($uvConf) {
        $uvConfigDir = "$env:USERPROFILE\.uv"
        if (-not (Test-Path $uvConfigDir)) { New-Item -ItemType Directory -Path $uvConfigDir -Force | Out-Null }
        $uvConf | Out-File -FilePath "$uvConfigDir\config.toml" -Encoding utf8 -Force
        Write-Host "UV configured to use TUNA mirror" -ForegroundColor Green
    }

    $uvVersion = python -m uv --version
    Write-Host "UV successfully installed! Version: $uvVersion" -ForegroundColor Green
    Write-Host "You can make UV easier to use with the following alias:" -ForegroundColor Cyan
    Write-Host "Add this to your PowerShell profile:" -ForegroundColor Cyan
    Write-Host 'function pip { python -m uv $args }'
} catch {
    Write-Host "Failed to install UV: $_" -ForegroundColor Red
    exit 1
}

Write-Host "Installation complete!" -ForegroundColor Green
Write-Host "You can now use Python and the UV package manager." -ForegroundColor Green
Write-Host 'Usage: python -m uv install [package_name]' -ForegroundColor Cyan
