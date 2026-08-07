# Swiflow 自主学习能力

## 背景

Swiflow 的 agent 本质上是一个有工具调用能力的 LLM 运行循环。在引入自主学习机制之前，每次对话都从零开始——agent 做过的事、走过的弯路、找到的好方法，全部在对话结束后消失。

本功能为 agent 补齐了持久化经验层和反思回路，使其可以跨会话积累知识，并通过 skill 系统将经验固化为可复用的工作流。

---

## 架构概览

```
对话 / Run
   │
   ├── system 注入 ## Ways of working（agent.charter 或默认种子）
   ├── 显著任务宣称完成 → reflect 闸（自检补课后再交）
   │
   ▼
agent 自主调用 experience_write
   │ 记录：可复用的处理逻辑（可一条任务多条，也可零条）+ outcome + tags + weight
   ▼
agent_experience 表
   │
   ├── 下次对话：experience_list（按 weight）/ experience_use 加权
   │
   └── 定期反思：reflection-loop skill
            │
            ├── 高频流程 → skill_draft
            └── 方向原则 → 建议写入 Ways of working（charter）
```

宗旨：**经验 = 值得跨任务复用的处理逻辑**，不是任务流水账，也不强制「一任务一条」。**反思把本轮事情做好**；**Ways of life（charter）朝正确方向多走**。同一 Runner，不是第二 agent。关键节点用 `observe`/slog 记入 `swiflow.log`（`reflect_*` / `charter_*` / `run_end`）。

---

## 新增组件

### 数据库表（`agent_experience`）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | TEXT | 主键 |
| sid | TEXT | 来源会话 |
| agent | TEXT | 所属 agent（查询时按此过滤）|
| summary | TEXT | 一句话可复用学习 |
| outcome | TEXT | success \| partial \| failure \| unknown |
| tags | TEXT/JSONB | JSON 字符串数组 |
| **weight** | INTEGER | 默认 1；在其他场景被用到时 +1（上限 100）|
| created_at | TEXT | 创建时间 |

列表按 **`weight DESC, created_at DESC`** 排序，高价值经验优先出现。

---

### 工具（`internal/tool/experience.go`）

**`experience_write`**

写入一条**可复用的处理逻辑**。不限每任务一条：同一任务可写多条互不相同的教训，没有通用价值就不写。

```json
{
  "summary": "OCR often swaps 皮重/毛重; prefer filename when values are impossible",
  "outcome": "success",
  "tags": ["excel", "ocr"],
  "used_ids": ["019f…"]
}
```

**`experience_list`**

按权重列出经验（默认 10 条）。

**`experience_use`**

标记旧经验在当前任务中有用（`id` 或 `ids`），`weight += 1`。

返回按权重排序的经验列表；agent 在复杂任务开始前应检索并复用高权重先验。

---

### System Prompt 引导（`internal/agent/agent.go`）

每次 Run 注入：

```
## Ways of working
<agent.charter or default seed>

## Learning & memory
Experiences = reusable handling logic (not one-per-task).
experience_write when generalizable; experience_list by weight; experience_use / used_ids …
… promote workflows via skill_manage; refine Ways of working via clear user corrections.
```

显著任务在宣称完成前会进入 **reflect 闸**（最多有限次）：对照本轮 goal 自检，能补则继续调用工具，而不是问「可否交卷」。偏好类跟进消息（如「以后都…」）可能把短原则 append 进 `agent_config.charter`（可在 Agent 设置里编辑）。

---

### Todo 持久化（`internal/tool/delegate.go`）

`todo_write` / `todo_read` 原先将任务清单存在进程内存的全局 map 中，进程重启后丢失。现在改为通过 `session_todos` 表持久化，重启后任务状态完整保留。

---

### reflection-loop Skill（`embed/init-skills/reflection-loop/SKILL.md`）

内置 skill，用户说"帮我建立自学习循环"即可激活。

流程：
1. 查询最近 20 条经验（高权重优先）
2. 按 tag 分组，找出出现 ≥ 3 次的模式
3. 高频**流程** → `skill_draft`
4. 高频**方向原则** → 建议写入 Ways of working（charter）；无逐步 Accept 队列
5. 可选 `schedule_create` 每周定期反思

---

## 使用示例

### 手动记录经验

在对话末尾，agent 会自动（或用户可手动触发）调用：

```
experience_write:
  summary: "抓取京东商品价格时需要先等待 JS 渲染，直接 fetch 拿不到数据"
  outcome: success
  tags: ["browser", "scraping", "javascript"]
```

### 查询历史经验

```
experience_list:
  limit: 5
```

### 激活自学习循环

在对话中说：**"帮我建立自学习循环"**，agent 会运行 `reflection-loop` skill，分析经验模式并设置每周定期反思任务。

---

## 实施文件索引

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `embed/schema.sql` / `schema.pg.sql` | 修改 | `agent_experience.weight` |
| `internal/store/store.go` | 修改 | Experience.Weight + BumpExperienceWeight |
| `internal/store/sqlstore/experience.go` | 修改 | 按权重排序 / 加权 |
| `internal/tool/experience.go` | 修改 | experience_use + used_ids |
| `internal/migrate/canonical.go` | 修改 | 补 weight 列 |
| `internal/tool/delegate.go` | 修改 | todo 从内存改为 store 持久化 |
| `internal/agent/agent.go` | 修改 | Learning 段引导加权复用 |
| `embed/init-skills/reflection-loop/SKILL.md` | 新建 | 内置反思 skill |
| `cmd/swiflow/serve.go` | 修改 | 注册 experience 工具 |
| `cmd/desktop/main.go` | 修改 | 同上 |

---

## 验证步骤

1. `go build ./...` — 编译通过
2. 启动服务（`go run ./cmd/swiflow serve`）
3. 发送一条消息，让 agent 完成任务后调用 `experience_write`
4. 重启服务，调用 `experience_list`，确认经验持久化
5. 调用 `todo_write` 写入任务，重启后调用 `todo_read`，确认不丢失
6. 说"帮我建立自学习循环"，验证 `reflection-loop` skill 触发并创建定时任务
