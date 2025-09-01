#!/bin/bash

# --------------------- 配置部分 ---------------------
# 前端构建命令（如 `yarn build` 或 `npm run build`）
FRONTEND_BUILD_CMD="yarn build"

# 目标平台（这里只构建 macOS 的两个架构）
PLATFORMS=(
  "macos-arm64"   # Apple Silicon (aarch64)
  "macos-amd64"   # Intel (x86_64)
)

# Rust 目标架构（分别对应 PLATFORMS）
TARGETS=(
  "aarch64-apple-darwin"    # Apple Silicon
  "x86_64-apple-darwin"     # Intel
)

IS_DEBUG_MODE="no"
SERVE_DST_NAME="main"
SERVE_DST_PATH="bin"
SERVE_SRC_PATH="src-core"
TAURI_SRC_PATH="src-tauri"
# 铭文版，铭文信息
EPIGRAPH_INFO=""

# 输出目录
TAURI_DST_PATH="output"
# ---------------------------------------------------

# debug mode or EPIGRAPH_INFO not empty, nochange version
if [ "$IS_DEBUG_MODE" == "yes" ]; then
  CURR_VERSION=$(jq -r '.version' package.json)
elif [ ! -z "$EPIGRAPH_INFO" ]; then
  CURR_VERSION=$(jq -r '.version' package.json)
  source ~/.apple_api_keys || { echo "❌ load-apple-keys failed"; exit 1; }
else
  CURR_VERSION=$(npm version patch --no-git-tag-version | sed 's/v//')
  # 0.1 加载APPLE_ID
  source ~/.apple_api_keys || { echo "❌ load-apple-keys failed"; exit 1; }

  # 0.2 替换版本号 Cargo.toml
  CARGO_TOML="$TAURI_SRC_PATH/Cargo.toml"
  sed -i '' "s/^version = \".*\"/version = \"$CURR_VERSION\"/" "$CARGO_TOML"

  # 0.3 开始替换版本号 tauri.conf.json
  TAURI_CONF="$TAURI_SRC_PATH/tauri.conf.json"
  sed -i '' "s/\"version\": \".*\"/\"version\": \"$CURR_VERSION\"/" "$TAURI_CONF"
fi
# 清理旧构建
rm -rf "$TAURI_DST_PATH/"*.dmg

# 1. 构建前端（仅执行一次）
echo "🚀 Building frontend..."
eval "$FRONTEND_BUILD_CMD" || { echo "❌ Frontend build failed"; exit 1; }

# echo "🚀 copy frontend dist to core..."
# rm -rf "$SERVE_SRC_PATH/initial/html/assets/"
# cp -r "dist/"* "$SERVE_SRC_PATH/initial/html/"

# 创建构建目录
mkdir -p "$TAURI_DST_PATH"

# 2. 并行构建两个 macOS 架构
for i in "${!PLATFORMS[@]}"; do
  (
    PLATFORM="${PLATFORMS[$i]}"
    TARGET="${TARGETS[$i]}"
    echo "🛠️  Building for $PLATFORM (target: $TARGET)..."

    # 2.1 准备构建 main
    LDFLAGS="-X 'main.Version=$CURR_VERSION' -X 'main.Epigraph=$EPIGRAPH_INFO' -s -w"

    # 2.11 执行构建
    OUTPUT_FILE="$SERVE_DST_PATH/$SERVE_DST_NAME-$TARGET"
    (cd $SERVE_SRC_PATH && env GOARCH=${PLATFORM##*-} CGO_ENABLED=1 \
      go build -tags='!windows' -ldflags "$LDFLAGS" -o "../$OUTPUT_FILE")

    echo "🛠️  Building golang success..."

    # 2.2 生成 Tauri 安装包（跳过前端构建）
    DEBUG_OPTION=$([ "$IS_DEBUG_MODE" != "yes" ] && echo "" || echo "--debug")
    yarn tauri build $DEBUG_OPTION --target "$TARGET" --config '{"build": {"beforeBuildCommand": "echo skip"}}'

    # 2.3 整理输出文件
    cp -r "$TAURI_SRC_PATH/target/$TARGET/release/bundle/dmg/"*".dmg" "$TAURI_DST_PATH/"
    if [ ! -z "$EPIGRAPH_INFO" ]; then
      # 铭文版：替换版本号为vip-$name
      EPIGRAPH_NAME=$(echo "$EPIGRAPH_INFO" | cut -d '|' -f 1)
      base_name=$(basename "$TAURI_SRC_PATH/target/$TARGET/release/bundle/dmg/"*".dmg")
      new_name=$(echo "$base_name" | sed "s/$CURR_VERSION/$EPIGRAPH_NAME/")
      mv "$TAURI_DST_PATH/$base_name" "$TAURI_DST_PATH/$new_name"
    fi
    echo "✅ $PLATFORM build done!" && cd "../"
  )
done

# 等待所有子进程完成
echo "🎉 All macOS builds finished! Output in $TAURI_DST_PATH"
