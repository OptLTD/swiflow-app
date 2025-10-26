# Swiflow 构建文档

本文档介绍 Swiflow 项目的构建和打包流程。

## 项目结构

Swiflow 项目包含三个主要代码模块：
- **src-front**: 前端代码，构建产物在 `dist/` 目录
- **src-core**: 后端 Go 代码，构建产物在 `bin/` 目录
- **src-tauri**: Tauri 应用代码，构建产物为 `.msi`、`.dmg` 文件

## 构建目标

项目支持三种主要构建目标：

### 1. macOS DMG
包含 `src-front` + `src-core` + `src-tauri` 的完整桌面应用

### 2. Windows MSI
包含 `src-front` + `src-core` + `src-tauri` 的完整桌面应用

### 3. Docker 镜像
包含 `src-front` (dist) + `src-core` (bin) 的服务器应用

## 目录

- [快速开始](#快速开始)
- [版本管理](#版本管理)
- [macOS DMG 构建](#macos-dmg-构建)
- [Windows MSI 构建](#windows-msi-构建)
- [Docker 镜像构建](#docker-镜像构建)
- [组件单独构建](#组件单独构建)
- [高级选项](#高级选项)

## 快速开始

### 环境要求

**基础环境**:
- Node.js 和 Yarn
- Go 1.19+

**桌面应用构建**（macOS DMG / Windows MSI）:
- Rust 和 Cargo
- Tauri CLI

**Docker 镜像构建**:
- Docker 和 Docker Buildx

### 构建流程

1. **更新版本**（可选）：
```bash
npm run build:version:patch  # 补丁版本 (x.y.Z)
npm run build:version:minor  # 次版本 (x.Y.0)
npm run build:version:major  # 主版本 (X.0.0)
```

2. **选择构建目标**：

**桌面应用**:
```bash
# macOS DMG
npm run build:mac        # 所有架构
npm run build:mac-arm    # ARM64
npm run build:mac-x86    # x86_64

# Windows MSI
npm run build:win        # 所有架构
npm run build:win-x64    # x64
npm run build:win-x86    # x86

# 所有平台
npm run build:all
```

**Docker 镜像**:
```bash
# 构建 Docker 镜像
npm run build:docker
```

## 版本管理

版本管理支持语义化版本控制：

```bash
# 更新补丁版本 (0.1.0 -> 0.1.1)
npm run build:version:patch

# 更新次版本 (0.1.1 -> 0.2.0)
npm run build:version:minor

# 更新主版本 (0.2.0 -> 1.0.0)
npm run build:version:major
```

版本更新会同时修改以下文件：
- `package.json`
- `src-tauri/tauri.conf.json`
- `src-tauri/Cargo.toml`

## macOS DMG 构建

构建包含完整桌面应用的 macOS DMG 安装包。

**组件**: `src-front` + `src-core` + `src-tauri`

### 使用构建脚本

```bash
# 构建所有架构
npm run build:mac

# 构建特定架构
npm run build:mac-arm    # Apple Silicon (M1/M2)
npm run build:mac-x86    # Intel 处理器
```

### 支持的架构

- `arm64` - Apple Silicon (M1/M2)
- `x86_64` - Intel 处理器
- `all` - 所有架构（默认）

### 构建输出

- **Go 二进制文件**: `bin/` 目录
- **前端构建产物**: `dist/` 目录
- **DMG 安装包**: `output/` 目录

### 自动公证配置

在 `~/.apple_api_keys` 文件中配置以下环境变量，构建脚本将自动执行公证流程：

```bash
APPLE_ID="your-apple-id@example.com"
APPLE_TEAM_ID="YOUR_TEAM_ID"
APPLE_API_KEY="YOUR_API_KEY_ID"
APPLE_API_ISSUER="YOUR_ISSUER_ID"
APPLE_API_KEY_PATH="/path/to/AuthKey_XXXXXXXXXX.p8"
```

### 手动公证流程

1. **构建应用**：
```bash
npm run build:mac
```

2. **压缩应用**（用于公证）：
```bash
ditto -c -k --keepParent "target/aarch64-apple-darwin/release/bundle/macos/Swiflow.app" "Swiflow.zip"
```

3. **提交公证**：
```bash
xcrun notarytool submit Swiflow.zip \
   --apple-id "$APPLE_ID" \
   --team-id "$APPLE_TEAM_ID" \
   --key-id "$APPLE_API_KEY" \
   --issuer "$APPLE_API_ISSUER" \
   --key "$APPLE_API_KEY_PATH" \
   --wait
```

4. **钉书机公证票证**：
```bash
xcrun stapler staple "target/aarch64-apple-darwin/release/bundle/macos/Swiflow.app"
```

### 公证状态示例

```text
Conducting pre-submission checks for Swiflow.zip and initiating connection to the Apple notary service...
Submission ID received
  id: 5d6b0a92-48a6-4630-9dc2-5ecc47dbf08e
Upload progress: 100.00% (9.48 MB of 9.48 MB)
Successfully uploaded file
  id: 5d6b0a92-48a6-4630-9dc2-5ecc47dbf08e
  path: /Users/chenwen/XCode/swiflow-app/src-tauri/Swiflow.zip
Waiting for processing to complete.
Current status: Accepted........
Processing complete
  id: 5d6b0a92-48a6-4630-9dc2-5ecc47dbf08e
  status: Accepted
```

## Windows MSI 构建

构建包含完整桌面应用的 Windows MSI 安装包。

**组件**: `src-front` + `src-core` + `src-tauri`

### 使用构建脚本

```bash
# 构建所有架构
npm run build:win

# 构建特定架构
npm run build:win-x64    # 64位
npm run build:win-x86    # 32位
```

### 支持的架构

- `x86` - 32位
- `x64` - 64位
- `all` - 所有架构（默认）

### 构建输出

- **Go 二进制文件**: `bin/` 目录
- **前端构建产物**: `dist/` 目录
- **MSI 安装包**: `output/` 目录

### 构建前置条件

Windows 构建需要确保以下组件已准备就绪：
- `frontend-dist`: 前端构建产物
- `golang-bin`: Go 后端二进制文件

**优化说明**: 为避免重复 `yarn install` 和环境不一致问题，项目采用分离式构建策略：
- **a1 环境**: 用于开发，处理前端构建
- **a2 环境**: 专用于 Windows 打包，复用已构建的产物

这种分离可显著加快 Windows 打包速度，避免在 Docker 容器内重复处理依赖。

### 使用 Docker 容器构建

1. **进入构建容器**：
```bash
# 一次性使用（退出后销毁）
docker run --rm -it -v $(pwd):/io -w /io websmurf/tauri-builder:1.1.0 bash

# 持久化容器（退出后不销毁）
docker run -it -v $(pwd):/io -w /io --name tauri-builder websmurf/tauri-builder:1.1.0 bash

# 重新进入已存在的容器
docker start -ai tauri-builder
# 或者
docker exec -it tauri-builder bash
```

2. **在容器中构建**：

```bash
# 确保前置条件已满足（frontend-dist 和 golang-bin 已准备）

# 正常构建（记得关闭代理）
yarn tauri build --runner cargo-xwin --target x86_64-pc-windows-msvc

# 跳过前端编译（推荐，因为已有 frontend-dist）
yarn tauri build --runner cargo-xwin --target x86_64-pc-windows-msvc --config '{"build": {"beforeBuildCommand": "echo skip"}}'

# 调试构建（会有控制台窗口）
yarn tauri build --runner cargo-xwin --target x86_64-pc-windows-msvc --debug --config '{"build": {"beforeBuildCommand": "echo skip"}}'
```

## Docker 镜像构建

构建包含服务器应用的 Docker 镜像。

**组件**: `src-front` (dist) + `src-core` (bin)

### 使用构建脚本（推荐）

```bash
# 基本构建
npm run build:docker
```

### 脚本功能

- 自动配置 `linux/amd64` 平台
- 支持自定义镜像标签
- 可选择构建后推送到仓库
- 提供干运行模式预览命令
- 详细的构建过程输出

### 使用 Docker 命令

```bash
# 构建镜像
docker buildx build --platform linux/amd64 -f ./Dockerfile -t optltd/swiflow:latest ./

# 推送镜像
docker push optltd/swiflow:latest

# 拉取镜像
docker pull optltd/swiflow:latest

# 运行容器
# 网络模式
docker run -d --name swiflow --network host optltd/swiflow:latest
# 端口模式
docker run -d --name swiflow --port 112358:11235 optltd/swiflow:latest
```

### 构建输出

- **Docker 镜像**: 包含前端 dist 和后端 bin 的完整服务器应用
- **镜像标签**: `optltd/swiflow:latest` 或自定义标签

## 组件单独构建

如需单独构建各个组件，可使用以下方法：

### src-core (Go 后端)

```bash
# 进入后端目录
cd src-core

# Linux AMD64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o swiflow -ldflags '-w -s' ./main.go && upx -9 swiflow

# macOS ARM64
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o swiflow -ldflags '-w -s' ./main.go

# macOS x86_64
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o swiflow -ldflags '-w -s' ./main.go

# Windows x64
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o swiflow.exe -ldflags '-w -s' ./main.go
```

**构建输出**: `bin/` 目录

### src-front (前端)

```bash
# 安装依赖
yarn install

# 开发模式
yarn dev

# 生产构建
yarn build

# 预览构建结果
yarn preview

# 运行测试
yarn test
```

**构建输出**: `dist/` 目录

### src-tauri (Tauri 应用)

```bash
# 进入 Tauri 目录
cd src-tauri

# 开发模式
cargo tauri dev

# 生产构建
cargo tauri build

# 特定平台构建
cargo tauri build --target x86_64-pc-windows-msvc  # Windows
cargo tauri build --target aarch64-apple-darwin    # macOS ARM64
cargo tauri build --target x86_64-apple-darwin     # macOS x86_64
```
**构建输出**: `.msi`、`.dmg` 等安装包文件

## 高级选项

### 调试模式

构建脚本支持调试模式，会生成包含调试信息的二进制文件，在 Tauri 构建中添加 `--debug` 选项，跳过某些优化步骤并提供详细的构建过程输出。

### 干运行模式

构建脚本支持干运行模式，可以测试构建配置而不实际执行构建操作。

### 环境变量

- `SWIFLOW_WIN_DIR`: Windows 构建输出目录
- `EPIGRAPH_INFO`: 铭文版本信息（格式：`name|text`）
- `APPLE_ID`, `APPLE_TEAM_ID` 等: macOS 公证相关配置

### 故障排除

**构建失败**：
- 检查依赖是否正确安装（Node.js, Go, Rust, Docker）
- 确认各工具版本符合要求
- 查看构建日志中的详细错误信息

**macOS 公证失败**：
- 验证 Apple API 密钥配置
- 检查开发者证书有效性
- 确认网络连接正常

**Windows 构建问题**：
- 确保 Docker 正常运行
- 检查容器网络设置
- 关闭可能影响构建的代理设置

**Docker 构建问题**：
- 确认 Docker Buildx 已安装
- 检查平台支持（linux/amd64）
- 验证 Dockerfile 路径和构建上下文

## 参考资料

- [Go 跨平台交叉编译](https://www.topgoer.com/%E5%85%B6%E4%BB%96/%E8%B7%A8%E5%B9%B3%E5%8F%B0%E4%BA%A4%E5%8F%89%E7%BC%96%E8%AF%91.html)
- [Tauri 官方文档](https://tauri.app/)
- [Apple 公证指南](https://developer.apple.com/documentation/security/notarizing_macos_software_before_distribution)