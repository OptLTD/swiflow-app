# Runtime Harness（一期：观测 + 人工审查）

> 旁路观测主/子 agent 运行状态与 goal 偏离；**不**向 LLM loop 注入纠偏。  
> 二期再根据积累的 case 做 soft nudge 自动纠偏。

## 架构

```
agent.run publisher
        │
        ▼
 harness.Tracker  ──observe──► RunSnapshot 树 + Drift 规则
        │
        ├── forward ──► SessionHub ──► chat/watch SSE
        └── harness_warn ──► SessionHub ──► UI 横幅 / HarnessPanel
```

- 实现：`internal/harness/`
- 接线：`cmd/swiflow/serve.go`、`cmd/desktop/main.go`、`cmd/swiflow/runtime.go` 的 `RunnerDeps.Publish = tracker`
- Tracker 包装 `SessionHub`：先更新状态，再转发原事件；偏离时额外发 `type=harness_warn`

## API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/runs` | 根会话快照列表（含已结束，便于复盘） |
| GET | `/api/runs/{id}` | 单会话快照 + children |
| GET | `/api/runs/watch` | SSE：`run_snapshot` 推送 |
| GET | `/api/sessions/{id}/children` | DB parent 列 + live harness 状态 |

## Drift 规则 v1（无第二 LLM）

| Code | 触发 |
|------|------|
| `stall_repeat_tools` | 同一工具连续调用 ≥3 |
| `stall_tool_errors` | 连续 tool 错误 ≥3 |
| `budget_pressure` | 子会话过半预算且无 todo/交付进展 |
| `todo_stale` | ≥2 轮有工具但 checklist 未变 |
| `done_with_open_todos` | `done` 时仍有未完成 todo |
| `no_progress` | ≥45s 无 tool/delta 进展 |
| `goal_tool_mismatch` | goal 像批量 OCR/汇总，却长期只 list/experience |

`harness_warn` 事件字段：`name`=code，`content`=message，子会话偏离时带 `child`。

## UI

- 聊天顶栏 **H** 打开 `HarnessPanel`：树 / goal / todos / drift
- 运行中 `harness_warn` 在消息区上方显示琥珀色横幅（可关闭）
- streaming 时 watch 通道仍放行 `harness_warn`（事件只走 hub，不走 chat POST 流）

## 一期明确不做

- soft nudge 注入 `llmMsgs`
- 硬拦 `done`
- 第二个 LLM 语义裁判
- 扩展 agent 内生反思检查点（现有 `batchDelegateNudge` / `childWrapUpNudge` 保持不动）

## 二期方向

见计划：按误报/漏报调规则 → 模板 soft nudge 注入 loop → 可选 agent 内生反思增强。
