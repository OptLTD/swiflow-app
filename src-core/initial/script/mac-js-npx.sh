#!/bin/bash

# macOS Node.js LTS 安装脚本（mainland/standard 合并版）
# 用法：
#   ./mac-js-npx.sh                # 默认国内镜像优先
#   ./mac-js-npx.sh standard       # 官方源优先

MODE="${1:-mainland}"
NODE_VERSION="20.12.2"
NODE_DISTRO="node-v${NODE_VERSION}-darwin-x64.tar.gz"
NODE_DISTRO_ARM="node-v${NODE_VERSION}-darwin-arm64.tar.gz"
TEMP_DIR=$(mktemp -d)

NODE_URLS_X64_MAINLAND=(
    "https://mirrors.huaweicloud.com/nodejs/release/v${NODE_VERSION}/${NODE_DISTRO}"
    "https://npmmirror.com/mirrors/node/v${NODE_VERSION}/${NODE_DISTRO}"
    "https://registry.npmmirror.com/-/binary/node/v${NODE_VERSION}/${NODE_DISTRO}"
    "https://nodejs.org/dist/v${NODE_VERSION}/${NODE_DISTRO}"
)
NODE_URLS_X64_STANDARD=(
    "https://nodejs.org/dist/v${NODE_VERSION}/${NODE_DISTRO}"
    "https://mirrors.huaweicloud.com/nodejs/release/v${NODE_VERSION}/${NODE_DISTRO}"
    "https://npmmirror.com/mirrors/node/v${NODE_VERSION}/${NODE_DISTRO}"
    "https://registry.npmmirror.com/-/binary/node/v${NODE_VERSION}/${NODE_DISTRO}"
)
NODE_URLS_ARM64_MAINLAND=(
    "https://mirrors.huaweicloud.com/nodejs/release/v${NODE_VERSION}/${NODE_DISTRO_ARM}"
    "https://npmmirror.com/mirrors/node/v${NODE_VERSION}/${NODE_DISTRO_ARM}"
    "https://registry.npmmirror.com/-/binary/node/v${NODE_VERSION}/${NODE_DISTRO_ARM}"
    "https://nodejs.org/dist/v${NODE_VERSION}/${NODE_DISTRO_ARM}"
)
NODE_URLS_ARM64_STANDARD=(
    "https://nodejs.org/dist/v${NODE_VERSION}/${NODE_DISTRO_ARM}"
    "https://mirrors.huaweicloud.com/nodejs/release/v${NODE_VERSION}/${NODE_DISTRO_ARM}"
    "https://npmmirror.com/mirrors/node/v${NODE_VERSION}/${NODE_DISTRO_ARM}"
    "https://registry.npmmirror.com/-/binary/node/v${NODE_VERSION}/${NODE_DISTRO_ARM}"
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

check_node() {
    if command -v node &>/dev/null; then
        CURRENT_NODE=$(node --version 2>&1)
        echo -e "${YELLOW}Node.js 已检测到: ${CURRENT_NODE}${NC}"
        return 1
    else
        echo -e "${YELLOW}未检测到 Node.js，将进行安装...${NC}"
    fi
    return 0
}

download_node() {
    local arch=$1
    local urls
    local dist
    if [[ "$arch" == "arm64" ]]; then
        if [[ "$MODE" == "standard" ]]; then
            urls=("${NODE_URLS_ARM64_STANDARD[@]}")
        else
            urls=("${NODE_URLS_ARM64_MAINLAND[@]}")
        fi
        dist="$NODE_DISTRO_ARM"
    else
        if [[ "$MODE" == "standard" ]]; then
            urls=("${NODE_URLS_X64_STANDARD[@]}")
        else
            urls=("${NODE_URLS_X64_MAINLAND[@]}")
        fi
        dist="$NODE_DISTRO"
    fi
    echo -e "${BLUE}尝试下载 Node.js ${NODE_VERSION} (${arch})...${NC}"
    for url in "${urls[@]}"; do
        echo -e "${BLUE}尝试镜像: ${url}${NC}"
        if curl -fL --progress-bar "$url" -o "${TEMP_DIR}/$dist"; then
            echo -e "${GREEN}下载成功!${NC}"
            return 0
        else
            echo -e "${RED}该镜像下载失败。${NC}"
        fi
    done
    echo -e "${RED}所有镜像下载失败，请手动下载 Node.js: https://nodejs.org/en/download${NC}"
    exit 1
}

install_node() {
    local arch=$1
    local dist
    if [[ "$arch" == "arm64" ]]; then
        dist="$NODE_DISTRO_ARM"
    else
        dist="$NODE_DISTRO"
    fi
    echo -e "${BLUE}正在解压 Node.js...${NC}"
    tar -xzf "${TEMP_DIR}/$dist" -C "$TEMP_DIR"
    NODE_DIR=$(find "$TEMP_DIR" -type d -name "node-v${NODE_VERSION}-darwin-*" | head -n 1)
    sudo mkdir -p /usr/local/nodejs
    sudo cp -R "$NODE_DIR"/* /usr/local/nodejs/
    TERMINAL_SHELL=$(ps -o comm= -p $(ps -o ppid= -p $$))
    if [[ "$TERMINAL_SHELL" == *"zsh"* ]]; then
        SHELL_RC="$HOME/.zshrc"
    elif [[ "$TERMINAL_SHELL" == *"bash"* ]]; then
        SHELL_RC="$HOME/.bashrc"
    else
        SHELL_RC="$HOME/.zshrc"
    fi
    if ! grep -q 'export PATH="/usr/local/nodejs/bin:$PATH"' "$SHELL_RC"; then
        echo 'export PATH="/usr/local/nodejs/bin:$PATH"' >> "$SHELL_RC"
    fi
    export PATH="/usr/local/nodejs/bin:$PATH"
    echo -e "${GREEN}Node.js ${NODE_VERSION} 安装完成!${NC}"
}

configure_npm() {
    echo -e "${BLUE}正在配置 npm 源为淘宝镜像...${NC}"
    npm config set registry https://registry.npmmirror.com
    npm config set disturl https://npmmirror.com/mirrors/node
    echo -e "${GREEN}npm 已配置为淘宝镜像${NC}"
}

echo 'install' > install.log
if check_node; then
    ARCH=$(uname -m)
    if [[ "$ARCH" == "arm64" ]]; then
        download_node arm64 && install_node arm64
    else
        download_node x64 && install_node x64
    fi
fi

if command -v npm &>/dev/null; then
    if [[ "$MODE" != "standard" ]]; then
        configure_npm
    fi
    echo -e "${CYAN}Node.js 和 npm 已准备就绪！${NC}"
    node -v
    npm -v
else
    echo -e "${RED}npm 未检测到，Node.js 安装可能失败。${NC}"
fi

rm -rf "$TEMP_DIR"
rm install.log 