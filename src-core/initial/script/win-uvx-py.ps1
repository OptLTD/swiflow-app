param(
    [string]$mode = "mainland"
)

# Configure UV mirror settings based on mode
if ($mode -eq "standard") {
    $uvConf = $null
} else {
    $uvConf = @"
[global]
index-url = https://pypi.tuna.tsinghua.edu.cn/simple
"@
}

Write-Host "Starting installation of UV and Python..." -ForegroundColor Green

# Disable Microsoft App Execution Aliases for Python
Write-Host "Checking and disabling Microsoft App Execution Aliases..." -ForegroundColor Blue

# Method 1: Disable via registry (check multiple possible paths)
try {
    # Find the actual DesktopAppInstaller package name dynamically
    $packagePaths = @()
    $baseRegistryPath = "HKCU:\Software\Microsoft\Windows\CurrentVersion\AppModel\SystemAppData"
    
    # Look for DesktopAppInstaller packages
    if (Test-Path $baseRegistryPath) {
        $packages = Get-ChildItem $baseRegistryPath | Where-Object { $_.Name -like "*DesktopAppInstaller*" }
        foreach ($package in $packages) {
            $aliasDataPath = Join-Path $package.PSPath "AliasData"
            if (Test-Path $aliasDataPath) {
                $packagePaths += $aliasDataPath
            }
        }
    }
    
    # Common alias names to disable
    $aliasNames = @("python.exe", "python3.exe", "pip.exe", "pip3.exe")
    
    foreach ($packagePath in $packagePaths) {
        foreach ($aliasName in $aliasNames) {
            $fullAliasPath = Join-Path $packagePath $aliasName
            if (Test-Path $fullAliasPath) {
                Set-ItemProperty -Path $fullAliasPath -Name "State" -Value 0 -Force
                Write-Host "$aliasName alias disabled in registry" -ForegroundColor Green
            }
        }
    }
    
    if ($packagePaths.Count -eq 0) {
        Write-Host "No DesktopAppInstaller registry paths found" -ForegroundColor Yellow
    }
} catch {
    Write-Host "Warning: Could not disable registry aliases: $_" -ForegroundColor Yellow
}

# Method 2: Handle WindowsApps python executables directly
try {
    $windowsAppsPath = "$env:LOCALAPPDATA\Microsoft\WindowsApps"
    $pythonExes = @("python.exe", "python3.exe", "pip.exe", "pip3.exe")
    
    foreach ($exeName in $pythonExes) {
        $exePath = Join-Path $windowsAppsPath $exeName
        if (Test-Path $exePath) {
            # Try to rename the executable to disable it
            $backupPath = "$exePath.disabled"
            try {
                if (-not (Test-Path $backupPath)) {
                    Rename-Item -Path $exePath -NewName "$exeName.disabled" -Force
                    Write-Host "$exeName renamed to $exeName.disabled" -ForegroundColor Green
                }
            } catch {
                Write-Host "Warning: Could not rename $exeName in WindowsApps: $_" -ForegroundColor Yellow
            }
        }
    }
} catch {
    Write-Host "Warning: Could not process WindowsApps executables: $_" -ForegroundColor Yellow
}

# Method 3: Remove from WindowsApps PATH if present
try {
    $currentPath = $env:PATH
    $windowsAppsPath = "$env:LOCALAPPDATA\Microsoft\WindowsApps"
    if ($currentPath -like "*$windowsAppsPath*") {
        # Temporarily remove WindowsApps from PATH for this session
        $newPath = ($currentPath -split ';' | Where-Object { $_ -ne $windowsAppsPath }) -join ';'
        $env:PATH = $newPath
        Write-Host "WindowsApps temporarily removed from PATH" -ForegroundColor Green
    }
} catch {
    Write-Host "Warning: Could not modify PATH: $_" -ForegroundColor Yellow
}

# Method 4: Force refresh environment variables
try {
    # Refresh environment variables to ensure changes take effect
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")
} catch {
    Write-Host "Warning: Could not refresh environment variables: $_" -ForegroundColor Yellow
}

# Check if UV is already installed
$uvExists = $false
try {
    $uvVersion = uv --version 2>$null
    if ($uvVersion) {
        $uvExists = $true
        Write-Host "UV is already installed: $uvVersion" -ForegroundColor Yellow
    }
} catch {
    # UV not found, will install
}

# Step 1: Install UV using accelerated mirror (if not exists)
if (-not $uvExists) {
    Write-Host "Installing UV package manager..." -ForegroundColor Blue
    try {
        # Set UV installer mirror for faster download
        $env:UV_INSTALLER_GITHUB_BASE_URL = "https://gh-proxy.com/https://github.com"
        
        # Download and execute UV installer
        powershell -ExecutionPolicy ByPass -c "irm https://proxy.swiflow.cc/https://astral.sh/uv/install.ps1 | iex"
        
        Write-Host "UV installation completed!" -ForegroundColor Green
    } catch {
        Write-Host "Failed to install UV: $_" -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "Skipping UV installation (already exists)" -ForegroundColor Yellow
}

# Refresh PATH to include UV
$env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")

# Check if Python is already installed via UV
$pythonExists = $false
try {
    $pythonList = uv python list 2>$null
    if ($pythonList -and ($pythonList -match "\*")) {
        $pythonExists = $true
        Write-Host "Python is already installed via UV" -ForegroundColor Yellow
    }
} catch {
    # Python not found via UV, will install
}

# Step 2: Install Python using UV with accelerated mirror (if not exists)
if (-not $pythonExists) {
    Write-Host "Installing Python using UV..." -ForegroundColor Blue
    try {
        # Set Python download mirror for faster download
        $env:UV_PYTHON_INSTALL_MIRROR = "https://gh-proxy.com/https://github.com/astral-sh/python-build-standalone/releases/download"
        
        # Install Python using UV
        uv python install -f --default
        
        # Set the installed Python as the global default
        Write-Host "Setting Python as global default..." -ForegroundColor Blue
        uv python pin
        
        Write-Host "Python installation completed!" -ForegroundColor Green
    } catch {
        Write-Host "Failed to install Python: $_" -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "Skipping Python installation (already exists)" -ForegroundColor Yellow
}

# Configure UV mirror (mainland mode only)
if ($uvConf) {
    Write-Host "Configuring UV to use China mirrors..." -ForegroundColor Blue
    try {
        $uvConfigDir = "$env:USERPROFILE\.uv"
        if (-not (Test-Path $uvConfigDir)) { New-Item -ItemType Directory -Path $uvConfigDir -Force | Out-Null }
        $uvConf | Out-File -FilePath "$uvConfigDir\config.toml" -Encoding utf8 -Force
        Write-Host "UV configured to use TUNA mirror" -ForegroundColor Green
    } catch {
        Write-Host "Failed to configure UV mirror: $_" -ForegroundColor Red
    }
}

# Ensure Python is accessible via 'python' command
Write-Host "Configuring Python command access..." -ForegroundColor Blue
try {
    # Add UV's Python shims to PATH for current session
    $uvPythonPath = "$env:USERPROFILE\.local\bin"
    if (Test-Path $uvPythonPath) {
        $env:Path = "$uvPythonPath;$env:Path"
        Write-Host "Added UV Python path to current session" -ForegroundColor Green
    }
    
    # Try to add to user PATH permanently
    $currentUserPath = [System.Environment]::GetEnvironmentVariable("Path", "User")
    if ($currentUserPath -notlike "*$uvPythonPath*") {
        [System.Environment]::SetEnvironmentVariable("Path", "$uvPythonPath;$currentUserPath", "User")
        Write-Host "Added UV Python path to user PATH" -ForegroundColor Green
    }
} catch {
    Write-Host "Warning: Could not configure Python command access: $_" -ForegroundColor Yellow
}

# Display version information
try {
    $uvVersion = uv --version
    Write-Host "UV successfully installed! Version: $uvVersion" -ForegroundColor Green
    
    # Try to display Python version
    try {
        $pythonVersion = python --version 2>$null
        if ($pythonVersion) {
            Write-Host "Python successfully configured! Version: $pythonVersion" -ForegroundColor Green
        } else {
            Write-Host "Python installed but may require PATH refresh. Try: uv run python --version" -ForegroundColor Yellow
        }
    } catch {
        Write-Host "Python installed but may require PATH refresh. Try: uv run python --version" -ForegroundColor Yellow
    }
} catch {
    Write-Host "Warning: Could not get UV version" -ForegroundColor Yellow
}

Write-Host "Installation complete!" -ForegroundColor Green
Write-Host "You can now use Python and the UV package manager." -ForegroundColor Green
Write-Host "Usage examples:" -ForegroundColor Cyan
Write-Host "  python --version         # Check Python version" -ForegroundColor Cyan
Write-Host "  uv run python script.py  # Run Python scripts" -ForegroundColor Cyan
Write-Host "  uv python list          # List available Python versions" -ForegroundColor Cyan
Write-Host "  uv pip install package  # Install Python packages" -ForegroundColor Cyan
Write-Host "  uv venv                  # Create virtual environment" -ForegroundColor Cyan
Write-Host "" -ForegroundColor White
Write-Host "Note: If 'python' command is not recognized, restart your terminal or run:" -ForegroundColor Yellow
Write-Host "  uv run python --version" -ForegroundColor Yellow
