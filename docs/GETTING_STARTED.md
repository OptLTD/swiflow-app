# Getting started

本地开发、构建、Docker 与 Skills 配置。产品介绍见根目录 [`README.md`](../README.md)。

## Quick start

```bash
cp config.example.json config.json   # 按需改 db / listen / tools
make dev      # API :8000 + UI http://localhost:5173
```

`serve` 默认会应用数据库 schema（`--migrate` 开启）。跳过迁移用 `--migrate=false`。

## Postgres（可选）

设置 `db_driver` 为 `postgres`，`db_dsn` 为 pgx URL（或环境变量 `SWIFLOW_DB_DRIVER` / `SWIFLOW_DB_DSN`）。Schema 来自 `embed/schema.pg.sql`。

```json
{
  "db_driver": "postgres",
  "db_dsn": "postgres://swiflow:swiflow@localhost:5432/swiflow?sslmode=disable"
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

桌面端内置 [Wails updater](https://v3.wails.io/zh-cn/tutorials/04-self-update-a-wails-app/)：菜单 **Check for Updates…** / 关于页「检查更新」，以及每日后台检查。版本号通过 `-ldflags` 注入 `internal/version.Version`（与 tag 对齐，勿带 `v` 前缀）。资源按扩展名区分平台（`.zip`→macOS，`.exe`→Windows），文件名只保留架构。

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
