# CDN：dl.option.ltd

多产品共用 Worker：`/{product}/…` → R2 `{product}/…`。不设产品白名单，R2 没有对应对象时原样 404。

| 公开 URL | R2 key |
|----------|--------|
| `https://dl.option.ltd/{product}/update.json` | `{product}/update.json` |
| `https://dl.option.ltd/{product}/<asset>` | `{product}/<asset>` |

Swiflow：

- Manifest：`https://dl.option.ltd/swiflow/update.json`
- 资源：`https://dl.option.ltd/swiflow/Swiflow-0.9.3-arm64.zip`

Worker 源码：[cloudflare-worker-dl.js](./cloudflare-worker-dl.js)。部署在 `dl.option.ltd` zone；源站为 `r2.option.ltd`（与 option-worth 等共用）。

发版：`release.yml` 生成 `update.json` + `SHA256SUMS`，上传 GitHub Release，并写 R2（`R2_PREFIX=swiflow`）。手动补同步用 `sync-to-r2.yml`。

仓库需配置：

- Secrets：`R2_ACCESS_KEY_ID`、`R2_SECRET_ACCESS_KEY`
- Variables：`R2_BUCKET_NAME`、`R2_ENDPOINT_URL`
