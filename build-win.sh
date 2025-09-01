#!/bin/bash

# 配置部分
APP_NAME="main"
SOURCE_DIR="src-core"
MAIN_FILE="main.go"
USE_UPX=false

# 期望的无参数运行输出
EXPECTED_OUTPUT="nice to meet you~"
TAURI_JSON_CONF="src-tauri/tauri.conf.json"
CURR_VERSION=$(jq -r '.version' package.json)
# 铭文版
EPIGRAPH_INFO=""

# 目标平台配置
PLATFORMS=(
    # "darwin/amd64"
    # "darwin/arm64"
    # "linux/amd64"
    "windows/amd64"
)

# 准备构建环境
echo "准备构建环境..."
(cd $SOURCE_DIR && go mod tidy)

# 构建函数
build_for_macos() {
    local GOOS=$1
    local GOARCH=$2
    OS_NAME="apple-darwin"
    
    # 设置输出文件名
    case "$GOARCH" in
        "arm64") ARCH_NAME="aarch64" ;;
        "amd64") ARCH_NAME="x86_64" ;;
        *) ARCH_NAME="$GOARCH" ;;
    esac
    
    OUTPUT_PATH="bin/$APP_NAME-$ARCH_NAME-$OS_NAME"
    
    echo -e "\n构建 darwin/$GOARCH..."
    
    # 准备构建参数
    local LDFLAGS="-X 'main.Version=$CURR_VERSION' -X 'main.Epigraph=$EPIGRAPH_INFO' -s -w"
    local CGO_ENABLED=1  # 默认禁用CGO
    
    # 执行构建
    (cd $SOURCE_DIR && \
     env GOOS=darwin GOARCH=$GOARCH CGO_ENABLED=$CGO_ENABLED go build \
        -tags='!windows' -ldflags "$LDFLAGS" -o "../$OUTPUT_PATH")
    if [ $? -ne 0 ]; then
        echo "错误: darwin/$GOARCH 构建失败"
        return 1
    fi
    return 0
}

verify_binary() {
    local BINARY_PATH=$1
    
    echo -e "\n验证 $BINARY_PATH ..."
    
    if [ ! -f "$BINARY_PATH" ]; then
        echo "错误: 可执行文件不存在"
        return 1
    fi
    
    # 给执行权限
    chmod +x "$BINARY_PATH" 2>/dev/null
    
    # 运行测试（带超时防止卡死）
    ACTUALLY_OUTPUT=$("$BINARY_PATH" -m say-hello 2>&1)
    EXIT_CODE=$?

    if [ $EXIT_CODE -eq 124 ]; then
        echo "错误: 程序执行超时"
        return 1
    elif [ $EXIT_CODE -ne 0 ]; then
        echo "错误: 程序非正常退出 (代码 $EXIT_CODE)"
        echo "输出: $ACTUALLY_OUTPUT"
        return 1
    fi
    
    if ! echo "$ACTUALLY_OUTPUT" | grep -q "$EXPECTED_OUTPUT"; then
        echo "错误: 输出中未找到预期文本"
        echo "期望包含: '$EXPECTED_OUTPUT'"
        echo "实际输出: '$ACTUALLY_OUTPUT'"
        return 1
    fi
    
    echo "验证通过 ✔"
    return 0
}

# 主构建流程
mkdir -p bin output
# rm -f bin/$APP_NAME-*

# 1. 构建前端（仅执行一次）
echo "🚀 Building frontend..."
eval "yarn build" || { echo "❌ Frontend build failed"; exit 1; }

# echo "🚀 Copy frontend dist to core..."
# rm -rf "$SOURCE_DIR/initial/html/assets/"
# cp -r "dist/"* "$SOURCE_DIR/initial/html/"

for PLATFORM in "${PLATFORMS[@]}"; do
    GOOS=${PLATFORM%/*}
    GOARCH=${PLATFORM#*/}

    # darwin bin
    if [ "$GOOS" == "darwin" ] ; then
        build_for_macos $GOOS $GOARCH
        continue
    fi

    # windows bin
    if [ "$GOOS" == "windows" ]; then
        LDFLAGS="-X 'main.Version=$CURR_VERSION' -X 'main.Epigraph=$EPIGRAPH_INFO' -s -w"
        (cd $SOURCE_DIR && env GOOS=windows GOARCH=amd64 CGO_ENABLED=1 go build -tags='windows' \
            -ldflags "$LDFLAGS -H windowsgui" -o "../bin/main-x86_64-pc-windows-msvc.exe") \
            && echo "Windows 构建成功(禁用CGO)" || echo "Windows 构建最终失败"

        # 如果设置了 SWIFLOW_WIN_DIR，则复制可执行文件和 dist 目录
        if [ -n "$SWIFLOW_WIN_DIR" ] && [ -f "$SWIFLOW_WIN_DIR/$TAURI_JSON_CONF" ]; then
            echo "拷贝 main 文件到 $SWIFLOW_WIN_DIR/bin/ ..."
            mkdir -p "$SWIFLOW_WIN_DIR/bin"
            cp bin/main-x86_64-pc-windows-msvc.exe "$SWIFLOW_WIN_DIR/bin/" || echo "拷贝可执行文件失败"
            if [ -d "dist" ]; then
                rm -rf "$SWIFLOW_WIN_DIR/dist"
                echo "复制 dist 目录到 $SWIFLOW_WIN_DIR/dist ..."
                cp -r dist "$SWIFLOW_WIN_DIR/dist"
            fi
        fi
        continue
    fi
done

echo -e "\n构建完成！"
echo "可执行文件已输出到 bin/ 目录"
ls -lh bin/

CURRENT_BINARY="bin/$APP_NAME-$(uname -m)-apple-darwin"
if [ "$(uname -s)" == "Darwin" ] && [ "$(uname -m)" == "arm64" ]; then
    CURRENT_BINARY="bin/$APP_NAME-aarch64-apple-darwin"
fi
if [ "$(uname -s)" == "Darwin" ] && [ -f "$CURRENT_BINARY" ]; then
    echo -e "\n执行最终验证..."
    if verify_binary "$CURRENT_BINARY"; then
        echo -e "\n所有构建和验证成功完成！"
        exit 0
    else
        echo -e "\n构建完成但验证失败！"
        exit 1
    fi
else
    echo -e "\n构建完成！(跳过最终验证 - 无当前平台二进制)"
    exit 0
fi
