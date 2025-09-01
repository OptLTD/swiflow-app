#!/bin/bash

# macOS Python + UV 安装脚本（mainland/standard 合并版）
# 用法：
#   ./mac-py-uvx.sh                # 默认国内镜像优先
#   ./mac-py-uvx.sh standard       # 官方源优先

MODE="${1:-mainland}"
PYTHON_VERSION="3.12.2"
PYTHON_INSTALLER="python-${PYTHON_VERSION}-macos11.pkg"
TEMP_DIR=$(mktemp -d)

PYTHON_URLS_MAINLAND=(
    "https://mirrors.huaweicloud.com/python/${PYTHON_VERSION}/${PYTHON_INSTALLER}"
    "https://registry.npmmirror.com/-/binary/python/${PYTHON_VERSION}/${PYTHON_INSTALLER}"
    "https://www.python.org/ftp/python/${PYTHON_VERSION}/${PYTHON_INSTALLER}"
)
PYTHON_URLS_STANDARD=(
    "https://www.python.org/ftp/python/${PYTHON_VERSION}/${PYTHON_INSTALLER}"
    "https://mirrors.huaweicloud.com/python/${PYTHON_VERSION}/${PYTHON_INSTALLER}"
    "https://registry.npmmirror.com/-/binary/python/${PYTHON_VERSION}/${PYTHON_INSTALLER}"
)

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

show_progress() {
    local pid=$1
    local delay=0.5
    local spinstr='|/-\'
    while [ "$(ps a | awk '{print $1}' | grep $pid)" ]; do
        local temp=${spinstr#?}
        printf " [%c]  " "$spinstr"
        local spinstr=$temp${spinstr%"$temp"}
        sleep $delay
        printf "\b\b\b\b\b\b"
    done
    printf "    \b\b\b\b"
}

check_python() {
    if command -v python3 &>/dev/null; then
        CURRENT_PYTHON=$(python3 --version 2>&1)
        echo -e "${YELLOW}Python detected: ${CURRENT_PYTHON}${NC}"
        return 1
    else
        echo -e "${YELLOW}Python not detected, will install...${NC}"
    fi
    return 0
}

download_python() {
    echo -e "${BLUE}Attempting to download Python ${PYTHON_VERSION}...${NC}"
    if [[ "$MODE" == "standard" ]]; then
        URLS=("${PYTHON_URLS_STANDARD[@]}")
    else
        URLS=("${PYTHON_URLS_MAINLAND[@]}")
    fi
    for url in "${URLS[@]}"; do
        echo -e "${BLUE}Trying mirror: ${url}${NC}"
        if curl -fL --progress-bar "$url" -o "${TEMP_DIR}/${PYTHON_INSTALLER}"; then
            echo -e "${GREEN}Download completed successfully!${NC}"
            return 0
        else
            echo -e "${RED}Download failed from this mirror.${NC}"
        fi
    done
    echo -e "${RED}All download attempts failed. Please download Python manually from https://www.python.org/downloads/${NC}"
    echo -e "${YELLOW}After installing Python, you can run this script again to install UV.${NC}"
    read -p "Would you like to open the Python download page now? (y/n): " OPEN_PAGE
    if [[ "$OPEN_PAGE" == "y" ]]; then
        open "https://www.python.org/downloads/"
    fi
    exit 1
}

install_python() {
    echo -e "${BLUE}Installing Python ${PYTHON_VERSION}...${NC}"
    sudo installer -pkg "${TEMP_DIR}/${PYTHON_INSTALLER}" -target / &
    show_progress $!
    if ! command -v python3 &>/dev/null; then
        echo -e "${RED}Python installation failed.${NC}"
        exit 1
    fi
    echo -e "${GREEN}Python ${PYTHON_VERSION} installation complete!${NC}"
}

configure_pip() {
    echo -e "${BLUE}Configuring pip to use China mirrors...${NC}"
    PIP_CONFIG_DIR="$HOME/Library/Application Support/pip"
    mkdir -p "$PIP_CONFIG_DIR"
    cat > "$PIP_CONFIG_DIR/pip.conf" <<EOF
[global]
index-url = https://pypi.tuna.tsinghua.edu.cn/simple
trusted-host = pypi.tuna.tsinghua.edu.cn
EOF
    echo -e "${GREEN}Pip configured to use TUNA mirror${NC}"
}

install_uv() {
    echo -e "${BLUE}Installing UV package manager...${NC}"
    if [[ "$MODE" == "standard" ]]; then
        python3 -m pip install --user uv &
    else
        python3 -m pip install --user uv -i https://pypi.tuna.tsinghua.edu.cn/simple &
    fi
    show_progress $!
    if [[ "$MODE" != "standard" ]]; then
        UV_CONFIG_DIR="$HOME/.uv"
        mkdir -p "$UV_CONFIG_DIR"
        cat > "$UV_CONFIG_DIR/config.toml" <<EOF
[global]
index-url = "https://pypi.tuna.tsinghua.edu.cn/simple"
EOF
        echo -e "${GREEN}UV configured to use TUNA mirror${NC}"
    fi
    if ! python3 -m uv --version &>/dev/null; then
        echo -e "${YELLOW}UV installation might have succeeded but cannot verify version.${NC}"
    else
        UV_VERSION=$(python3 -m uv --version)
        echo -e "${GREEN}UV successfully installed! Version: ${UV_VERSION}${NC}"
    fi
    PYTHON_SCRIPTS_DIR=$(python3 -m site --user-base)/bin
    TERMINAL_SHELL=$(ps -o comm= -p $(ps -o ppid= -p $$))
    if [[ "$TERMINAL_SHELL" == *"zsh"* ]]; then
        SHELL_RC="$HOME/.zshrc"
    elif [[ "$TERMINAL_SHELL" == *"bash"* ]]; then
        SHELL_RC="$HOME/.bashrc"
    else
        SHELL_RC="$HOME/.zshrc"
    fi
    if ! grep -q "export PATH=\"$PYTHON_SCRIPTS_DIR:\$PATH\"" "$SHELL_RC"; then
        echo "export PATH=\"$PYTHON_SCRIPTS_DIR:\$PATH\"" >> "$SHELL_RC"
    fi
    export PATH="$PYTHON_SCRIPTS_DIR:$PATH"
}

echo 'install' > install.log
if check_python; then
    echo -e "${GREEN}Starting installation of Python${NC}"
    download_python && install_python
    if [[ "$MODE" != "standard" ]]; then
        configure_pip
    fi
    echo -e "${GREEN}Python Installation complete!${NC}"
fi

if ! command -v uv &>/dev/null; then
    echo -e "${GREEN}Starting installation of uv${NC}"
    install_uv
    echo -e "${CYAN}Usage: uv install [package_name]${NC}"
fi

rm -rf "$TEMP_DIR"
rm install.log 