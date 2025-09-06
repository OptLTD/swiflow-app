#!/bin/bash

# Default mode is mainland for China mirrors
mode="${1:-mainland}"

# Configure UV mirror settings based on mode
if [ "$mode" = "standard" ]; then
    uv_conf=""
else
    uv_conf='[global]
index-url = https://pypi.tuna.tsinghua.edu.cn/simple'
fi

echo "Starting installation of UV and Python..."

# Function to print colored output - use tput for better compatibility
if command -v tput >/dev/null 2>&1; then
    # Use tput for better terminal compatibility
    print_green() { echo "$(tput setaf 2)$1$(tput sgr0)"; }
    print_blue() { echo "$(tput setaf 5)$1$(tput sgr0)"; }  # Changed to magenta for better visibility
    print_yellow() { echo "$(tput setaf 3)$1$(tput sgr0)"; }
    print_red() { echo "$(tput setaf 1)$1$(tput sgr0)"; }
    print_cyan() { echo "$(tput setaf 6)$1$(tput sgr0)"; }
else
    # Fallback to ANSI escape codes
    print_green() { echo "\033[32m$1\033[0m"; }
    print_blue() { echo "\033[35m$1\033[0m"; }  # Changed to magenta for better visibility
    print_yellow() { echo "\033[33m$1\033[0m"; }
    print_red() { echo "\033[31m$1\033[0m"; }
    print_cyan() { echo "\033[36m$1\033[0m"; }
fi

# Handle potential conflicts with system Python installations
print_blue "Checking for potential Python conflicts..."

# Check for Homebrew Python that might interfere
if command -v brew >/dev/null 2>&1; then
    if brew list python@3.12 >/dev/null 2>&1 || brew list python@3.11 >/dev/null 2>&1 || brew list python@3.10 >/dev/null 2>&1; then
        print_yellow "Warning: Homebrew Python detected. UV will manage Python versions independently."
    fi
fi

# Check for pyenv that might interfere
if command -v pyenv >/dev/null 2>&1; then
    print_yellow "Warning: pyenv detected. Consider using 'pyenv global system' to avoid conflicts."
fi

# Check for system Python in /usr/bin
if [ -f "/usr/bin/python3" ]; then
    print_blue "System Python3 found at /usr/bin/python3"
fi

# Check if UV is already installed
uv_exists=false
if command -v uv >/dev/null 2>&1; then
    uv_version=$(uv --version 2>/dev/null)
    if [ $? -eq 0 ]; then
        uv_exists=true
        print_yellow "UV is already installed: $uv_version"
    fi
fi

# Step 1: Install UV using accelerated mirror (if not exists)
if [ "$uv_exists" = false ]; then
    print_blue "Installing UV package manager..."
    
    # Set UV installer mirror for faster download
    export UV_INSTALLER_GITHUB_BASE_URL="https://gh-proxy.com/https://github.com"
    
    # Download and execute UV installer
    if curl -LsSf https://proxy.swiflow.cc/https://astral.sh/uv/install.sh | sh; then
        print_green "UV installation completed!"
    else
        print_red "Failed to install UV"
        exit 1
    fi
else
    print_yellow "Skipping UV installation (already exists)"
fi

# Refresh PATH to include UV
# UV installs to ~/.local/bin on Unix systems
if [ -d "$HOME/.local/bin" ]; then
    export PATH="$HOME/.local/bin:$PATH"
fi

# Also check for UV in ~/.cargo/bin (alternative installation location)
if [ -d "$HOME/.cargo/bin" ]; then
    export PATH="$HOME/.cargo/bin:$PATH"
fi

# Verify UV is now accessible
if ! command -v uv >/dev/null 2>&1; then
    print_red "UV installation failed or not in PATH"
    print_yellow "Please restart your terminal or run: source ~/.bashrc (or ~/.zshrc)"
    exit 1
fi

# Check if Python is already installed via UV
python_exists=false
if uv python list 2>/dev/null | grep -q "\*"; then
    python_exists=true
    print_yellow "Python is already installed via UV"
fi

# Step 2: Install Python using UV with accelerated mirror (if not exists)
if [ "$python_exists" = false ]; then
    print_blue "Installing Python using UV..."
    
    # Set Python download mirror for faster download
    export UV_PYTHON_INSTALL_MIRROR="https://gh-proxy.com/https://github.com/astral-sh/python-build-standalone/releases/download"
    
    # Install Python using UV without --default to avoid warnings
    if uv python install; then
        # Set the installed Python as the global default
        print_blue "Setting Python as global default..."
        uv python pin
        print_green "Python installation completed!"
    else
        print_red "Failed to install Python"
        exit 1
    fi
else
    print_yellow "Skipping Python installation (already exists)"
fi

# Configure UV mirror (mainland mode only)
if [ -n "$uv_conf" ]; then
    print_blue "Configuring UV to use China mirrors..."
    
    uv_config_dir="$HOME/.uv"
    if [ ! -d "$uv_config_dir" ]; then
        mkdir -p "$uv_config_dir"
    fi
    
    if echo "$uv_conf" > "$uv_config_dir/config.toml"; then
        print_green "UV configured to use TUNA mirror"
    else
        print_red "Failed to configure UV mirror"
    fi
fi

# Ensure Python is accessible via 'python' command
print_blue "Configuring Python command access..."

# Add UV's Python shims to PATH for current script execution
uv_python_path="$HOME/.local/bin"
if [ -d "$uv_python_path" ]; then
    export PATH="$uv_python_path:$PATH"
    print_green "Added UV Python path to current script execution"
fi

# Add to shell profile for permanent PATH modification
shell_profile=""
# Detect current shell more reliably
if [ -n "$ZSH_VERSION" ] || [[ "$0" == *"zsh"* ]] || [[ "$SHELL" == *"zsh"* ]]; then
    shell_profile="$HOME/.zshrc"
elif [ -n "$BASH_VERSION" ] || [[ "$0" == *"bash"* ]] || [[ "$SHELL" == *"bash"* ]]; then
    shell_profile="$HOME/.bashrc"
else
    # Fallback to .profile for other shells
    shell_profile="$HOME/.profile"
fi

if [ -n "$shell_profile" ]; then
    # Create shell profile if it doesn't exist
    if [ ! -f "$shell_profile" ]; then
        touch "$shell_profile"
        print_yellow "Created $shell_profile"
    fi
    
    if ! grep -q "$uv_python_path" "$shell_profile"; then
        echo "export PATH=\"$uv_python_path:\$PATH\"" >> "$shell_profile"
        print_green "Added UV Python path to $shell_profile"
    else
        print_yellow "UV Python path already exists in $shell_profile"
    fi
fi

# Display version information
if uv_version=$(uv --version 2>/dev/null); then
    print_green "UV successfully installed! Version: $uv_version"
    
    # Try to display Python version
    if python_version=$(python --version 2>/dev/null); then
        print_green "Python successfully configured! Version: $python_version"
    elif python_version=$(uv run python --version 2>/dev/null); then
        print_yellow "Python installed but requires 'uv run' prefix. Version: $python_version"
        print_yellow "Consider restarting your terminal for direct 'python' access"
    else
        print_yellow "Python installed but may require PATH refresh. Try: uv run python --version"
    fi
else
    print_yellow "Warning: Could not get UV version"
fi

print_green "Installation complete!"
print_green "You can now use Python and the UV package manager."
print_cyan "Usage examples:"
print_cyan "  python --version         # Check Python version"
print_cyan "  uv run python script.py  # Run Python scripts"
print_cyan "  uv python list          # List available Python versions"
print_cyan "  uv pip install package  # Install Python packages"
print_cyan "  uv venv                  # Create virtual environment"
echo ""
print_yellow "IMPORTANT: To use 'uv' and 'python' commands in your terminal:"
print_yellow "  1. Restart your terminal, OR"
print_yellow "  2. Run: source $shell_profile"
print_yellow "  3. Then test with: uv --version"