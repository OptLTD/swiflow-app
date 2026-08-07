# Getting started

本地开发、构建、Docker 与 Skills 配置。产品介绍见根目录 [`README.md`](../README.md)。

## Quick start

```bash
cp config.example.json config.json   # 按需改 db / listen / tools
make dev      # API :8000 + UI http://localhost:5173
```

`serve` 默认会应用数据库 schema（`--migrate` 开启）。跳过迁移用 `--migrate=false`。

## Postgres（可选）

设置 `database_dsn` 为 pgx URL（或环境变量 `SWIFLOW_DATABASE_DSN`）。Schema 来自 `embed/schema.pg.sql`。

```json
{
  "host_address": "127.0.0.1:8000",
  "database_dsn": "postgres://swiflow:swiflow@localhost:5432/swiflow?sslmode=disable"
}
```

## Build & run

```bash
make build    # webui + Go binary → ./swiflow
./swiflow serve -v
```

## Docker

镜像内含 Chromium / Python3 / Node，供 browser 与脚本工具使用：

```bash
cp config.example.json config.json   # 填入密钥等
make image
docker compose up
```

Chromium 需要足够的 `/dev/shm`（`compose.yml` 已设 `shm_size: 256mb`）。直接 `docker run` 时请加 `--shm-size=256m`。

## Skills

内置 Skills 打进二进制（`embed/init-skills/`）。用户覆盖放在 `./data/user-skills/`（见 `config.example.json`）。本地改 Skill 不想每次 rebuild，可设 `init_skills_dir` 或 `SWIFLOW_INIT_SKILLS` 指向目录。

## Desktop release（GitHub Actions）

推送符合 `v*` 的 tag 后，[.github/workflows/release.yml](../.github/workflows/release.yml) 会自动：

- 在 **macOS** 构建 `Swiflow.app`，打包为 `Swiflow-<version>-arm64.zip`
- 在 **Windows** 构建 NSIS 安装包 `Swiflow-<version>-installer.exe`，以及可供应用内更新替换的便携包 `Swiflow-<version>-{amd64,arm64}.exe`
- 生成并上传 `SHA256SUMS`（更新时校验下载完整性）
- 创建 GitHub Release 并挂上上述产物（当前为**未签名**包；macOS 为 ad-hoc codesign）

桌面端内置 [Wails updater](https://v3.wails.io/zh-cn/tutorials/04-self-update-a-wails-app/)（`endpoint` provider）：读取 `https://dl.option.ltd/swiflow/update.json`，产物从 `https://dl.option.ltd/swiflow/...` 下载（与 option-worth 共用 `dl.option.ltd` Worker，源站 R2 `swiflow/`）。打 tag 发版时，`release.yml` 的 GitHub Release job **末尾直接上传 R2**（因默认 `GITHUB_TOKEN` 创建的 Release 不会触发其它 workflow）。手动改 Release 后可用 [.github/workflows/sync-to-r2.yml](../.github/workflows/sync-to-r2.yml) 补同步。版本号通过 `-ldflags` 注入 `internal/version.Version`（与 tag 对齐，勿带 `v` 前缀）。可用环境变量 `SWIFLOW_UPDATE_MANIFEST` 覆盖 manifest 地址。

仓库需配置 Variables：`R2_BUCKET_NAME`、`R2_ENDPOINT_URL`；Secrets：`R2_ACCESS_KEY_ID`、`R2_SECRET_ACCESS_KEY`。

CDN 说明见 [cloudflare-worker-dl.md](./cloudflare-worker-dl.md)；Worker 脚本见 [cloudflare-worker-dl.js](./cloudflare-worker-dl.js)。

```bash
git tag v0.2.0
git push origin v0.2.0
```

本地等价命令：`make macos APP_VERSION=0.2.0`、`make windows APP_VERSION=0.2.0`（见根目录 Makefile）。

---

| 文档 | 内容 |
|------|------|
| [`SPEC.md`](SPEC.md) | 产品 / API / Schema 约定与路线图 |
| [`AGENT_ARCHITECTURE.md`](AGENT_ARCHITECTURE.md) | Agent 运行时实现说明 |
| [`AGENT_WORKFLOW_PATTERNS.md`](AGENT_WORKFLOW_PATTERNS.md) | 委派 / 队列 / Clarify / Skill 草稿等模式 |
| [`IMPLEMENTATION_STATUS.md`](IMPLEMENTATION_STATUS.md) | 已落地 vs 延期清单 |
| [`README.md`](README.md) | docs 索引 |
