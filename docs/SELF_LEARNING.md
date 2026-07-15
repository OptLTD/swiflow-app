# Swiflow 自主学习能力

## 背景

Swiflow 的 agent 本质上是一个有工具调用能力的 LLM 运行循环。在引入自主学习机制之前，每次对话都从零开始——agent 做过的事、走过的弯路、找到的好方法，全部在对话结束后消失。

本功能为 agent 补齐了持久化经验层和反思回路，使其可以跨会话积累知识，并通过 skill 系统将经验固化为可复用的工作流。

---

## 架构概览

```
对话结束
   │
   ▼
agent 自主调用 experience_write
   │ 记录：一句话摘要 + outcome + tags
   ▼
experiences 表（SQLite）
   │
   ├── 下次对话：agent 调用 experience_list，检索相关经验
   │
   └── 定期反思：reflection-loop skill
            │
            ▼
       发现高频模式 → skill_draft → 人工确认 → 用户 skill
```

---

## 新增组件

### 数据库表（`embed/upgrades/0003_experience.sql`）

**`experiences`** — 经验记录表
| 字段 | 类型 | 说明 |
|------|------|------|
| id | TEXT | 主键 |
| session_key | TEXT | 来源会话 |
| agent | TEXT | 所属 agent（查询时按此过滤）|
| summary | TEXT | 一句话摘要 |
| outcome | TEXT | success \| partial \| failure \| unknown |
| tags | TEXT | JSON 字符串数组，如 `["data-analysis","excel"]` |
| created_at | TEXT | 创建时间 |

**`session_todos`** — 持久化任务清单（原先存内存，重启丢失）
| 字段 | 类型 | 说明 |
|------|------|------|
| session_key | TEXT | 主键 |
| items | TEXT | `[{id, text, done}]` JSON |
| updated_at | TEXT | 最后更新时间 |

---

### 工具（`internal/tool/experience.go`）

**`experience_write`**

记录一条经验到当前 agent 的经验库。

```json
{
  "summary": "用 pandas read_excel + groupby 处理运费明细表，注意编码要指定 gbk",
  "outcome": "success",
  "tags": ["excel", "pandas", "encoding"]
}
```

**`experience_list`**

查询当前 agent 最近的经验列表。

```json
{
  "limit": 10
}
```

返回按时间倒序的经验列表，agent 可在复杂任务开始前调用，检索是否有可复用的先验知识。

---

### System Prompt 引导（`internal/agent/agent.go`）

在每次 agent 运行的系统提示中追加了以下指导段：

```
## Learning & memory
After completing a significant task (success or failure), call experience_write to
record: a one-sentence summary, the outcome, and 1-3 tags.
Before starting a complex task, call experience_list to check if you have relevant
prior experience that could shortcut the work.
Use skill_manage to promote a frequently-needed experience into a reusable skill.
```

这段引导让 agent 在完成任务后主动记录经验，在开始复杂任务前主动检索历史，并在发现高频模式时自动生成 skill。

---

### Todo 持久化（`internal/tool/delegate.go`）

`todo_write` / `todo_read` 原先将任务清单存在进程内存的全局 map 中，进程重启后丢失。现在改为通过 `session_todos` 表持久化，重启后任务状态完整保留。

---

### reflection-loop Skill（`embed/init-skills/reflection-loop/SKILL.md`）

内置 skill，用户说"帮我建立自学习循环"即可激活。

流程：
1. 查询最近 20 条经验
2. 按 tag 分组，找出出现 ≥ 3 次的模式
3. 为每个高频模式调用 `skill_draft` 生成 skill 草稿（需人工确认）
4. 调用 `schedule_create` 设置每周定期反思任务

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
| `embed/upgrades/0003_experience.sql` | 新建 | experiences + session_todos 建表 |
| `internal/store/store.go` | 修改 | Experience 类型 + 5 个接口方法 |
| `internal/store/sqlite/experience.go` | 新建 | SQLite 实现 |
| `internal/tool/experience.go` | 新建 | experience_write + experience_list |
| `internal/tool/delegate.go` | 修改 | todo 从内存改为 store 持久化 |
| `internal/agent/agent.go` | 修改 | buildSystem 追加 Learning 段 |
| `embed/init-skills/reflection-loop/SKILL.md` | 新建 | 内置反思 skill |
| `cmd/swiflow/serve.go` | 修改 | 注册 experience 工具，传 st 给 RegisterTodo |
| `cmd/desktop/main.go` | 修改 | 同上 |

---

## 验证步骤

1. `go build ./...` — 编译通过
2. 启动服务（`go run ./cmd/swiflow serve`）
3. 发送一条消息，让 agent 完成任务后调用 `experience_write`
4. 重启服务，调用 `experience_list`，确认经验持久化
5. 调用 `todo_write` 写入任务，重启后调用 `todo_read`，确认不丢失
6. 说"帮我建立自学习循环"，验证 `reflection-loop` skill 触发并创建定时任务
