package agent

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"swiflow/action"
	"swiflow/amcp"
	"swiflow/builtin"
	"swiflow/config"
	"swiflow/entity"
	"swiflow/initial"
	"swiflow/model"
	"swiflow/storage"
	"swiflow/support"

	"github.com/duke-git/lancet/v2/fileutil"
)

type Context struct {
	usePrompt string
	useMemory string

	worker *Worker
	mytask *MyTask

	store storage.MyStore
}

func NewContext(store storage.MyStore, worker *Worker) *Context {
	return &Context{store: store, worker: worker}
}

func (c *Context) Get() []*model.Message {
	return c.GetContext()
}

func (c *Context) GetGroup() string {
	return c.mytask.Group
}

func (c *Context) GetWorkerId() string {
	return c.worker.UUID
}

func (c *Context) GetWorkHome() string {
	return c.mytask.Home
}

func (c *Context) GetMsgRole(op string) string {
	switch op {
	case "user-input":
		return "user"
	case "bot-reply":
		return "assistant"
	case "tool-result":
		return "user"
	case "subtask":
		return "user"
	default:
		return "system"
	}
}
func (c *Context) TaskContext() string {
	return c.mytask.Context
}

func (c *Context) ParseMsgs(msgs []*MyMsg) []*action.SuperAction {
	result := []*action.SuperAction{}
	resMap := map[string]*action.SuperAction{}
	layout := "2006-01-02 15:04:05"
	for _, item := range msgs {
		if item.Request != "" {
			resp := action.Parse(item.Request)
			if send := item.SendAt; send != nil {
				resp.Datetime = send.Format(layout)
			}

			if item.OpType == "user-input" {
				resp.TheMsgId = item.UniqId
				result = append(result, resp)
			}
			if last, ok := resMap[item.PrevId]; ok {
				last.Merge(resp)
			}
		}

		if item.Respond != "" {
			resp := action.Parse(item.Respond)
			if recv := item.RecvAt; recv != nil {
				resp.Datetime = recv.Format(layout)
			}
			resp.TheMsgId = item.UniqId
			if resp.ErrMsg == nil {
				resMap[resp.TheMsgId] = resp
				result = append(result, resp)
			}
		}
	}
	return result
}

func (c *Context) GetMsgs(count int) []*model.Message {
	messages := make([]*model.Message, 0)
	msgs, _ := c.store.LoadMsg(c.mytask)
	for i := 0; i < len(msgs); i++ {
		if count > 0 && i+count <= len(msgs) {
			continue
		}

		req := model.Message{Content: msgs[i].Request}
		req.Role = c.GetMsgRole(msgs[i].OpType)
		messages = append(messages, &req)
		if strings.TrimSpace(msgs[i].Respond) != "" {
			messages = append(messages, &model.Message{
				Role: "assistant", Content: msgs[i].Respond,
			})
		}
	}
	return messages
}

func (c *Context) GetContext() []*model.Message {
	messages := make([]*model.Message, 0)
	if prompt := c.GetPrompt(); prompt != "" {
		messages = append(messages, &model.Message{
			Role: "system", Content: prompt,
		})
	}
	if memory := c.GetMemory(); memory != "" {
		messages = append(messages, &model.Message{
			Role: "user", Content: memory,
		})
	}
	size := config.GetInt("CTX_MSG_SIZE", 100)
	if msgs := c.GetMsgs(size); len(msgs) > 0 {
		messages = append(messages, msgs...)
	}
	return messages
}

func (c *Context) HasMcpError() bool {
	if finalPrompt := c.GetPrompt(); finalPrompt == "" {
		return true
	} else {
		return strings.Contains(finalPrompt, amcp.NO_TOOL_MSG)
	}
}

func (c *Context) GetSubject() string {
	return c.mytask.Name
}

func (c *Context) DebugCall(op string, msgs []*model.Message) {
	log.Println("[EXEC]", c.worker.Name, c.mytask.UUID, op)
	if c.mytask.IsDebug == false {
		return
	}

	var s strings.Builder
	var sysPrompt string
	for i, msg := range msgs {
		if i == 0 {
			sysPrompt = msg.Content
			continue
		}
		s.WriteString("<--- " + msg.Role + " --->")
		s.WriteString("\n" + msg.Content + "\n\n")
	}
	path := support.Or(c.GetWorkHome(), config.GetWorkHome())
	if fileutil.CreateDir(filepath.Join(path, ".msgs")) != nil {
		return
	}

	history := filepath.Join(path, ".msgs", c.worker.UUID+".xml")
	fileutil.WriteStringToFile(history, s.String(), false)
	prompt := filepath.Join(path, ".msgs", c.worker.UUID+".md")
	fileutil.WriteStringToFile(prompt, sysPrompt, false)
}

func (c *Context) Memorize(act *action.Memorize) error {
	mem := &entity.MemEntity{
		Bot: c.worker.UUID, Type: "chat",
		Subject: act.Subject,
		Content: act.Content,
	}
	return c.store.SaveMem(mem)
}

func (c *Context) Annotate(act *action.Annotate) error {
	if act.Subject != "" {
		c.mytask.Name = act.Subject
	}
	c.mytask.Context = act.Context
	return nil
}

func (c *Context) WaitTodo(act *action.WaitTodo) (err error) {
	todo := &entity.TodoEntity{UUID: act.UUID, Task: c.mytask.UUID}
	if act.UUID == "" && (act.Time == "" || act.Todo == "") {
		return fmt.Errorf("error: wrong wait-todo format")
	}
	if act.UUID == "" && act.Time != "" && act.Todo != "" {
		todo.UUID, _ = support.UniqueID(12)
		todo.Time, todo.Todo = act.Time, act.Todo
		if err = c.store.SaveTodo(todo); err == nil {
			support.Emit("wait-todo", "create", todo)
			return
		}
		return fmt.Errorf("error: %w", err)
	}

	// find todo
	if err = c.store.FindTodo(todo); err != nil {
		return fmt.Errorf("error: %w", err)
	}
	if act.Time != "" && act.Todo != "" {
		todo.Time, todo.Todo = act.Time, act.Todo
	} else {
		todo.Done = 1
	}
	if err = c.store.SaveTodo(todo); err == nil {
		event := support.If(todo.Done, "remove", "update")
		support.Emit("wait-todo", event, todo)
	}
	return
}

func (c *Context) SetState(state string) error {
	c.mytask.State = state
	support.Emit("control", c.mytask.UUID, state)
	if err := c.store.SaveTask(c.mytask); err != nil {
		return fmt.Errorf("save state error: %v", err)
	}
	return nil
}

func (c *Context) WriteMsg(msg *MyMsg) error {
	if c.store == nil {
		log.Println("[EXEC]", msg.TaskId, "log msg fail: store nil")
		return fmt.Errorf("write msg error: store nil")
	}
	// add msg group of task
	if c.mytask.Group != "" {
		msg.Group = c.mytask.Group
	}
	if err := c.store.SaveMsg(msg); err != nil {
		log.Println("[EXEC]", msg.TaskId, "log msg fail:", err)
		return fmt.Errorf("write msg error: %v", err)
	}
	return nil
}

func (c *Context) GetMemory() string {
	if c.useMemory != "" {
		return c.useMemory
	}
	var memory strings.Builder
	for _, mem := range c.worker.Memories {
		memory.WriteString("\n")
		memorize := &action.Memorize{
			Content:  strings.TrimSpace(mem.Content),
			Subject:  strings.TrimSpace(mem.Subject),
			Datetime: mem.CreatedAt.String(),
		}
		memory.WriteString(support.ToXML(memorize, nil))
	}
	c.useMemory = memory.String()
	return c.useMemory
}

func (c *Context) GetPrompt() string {
	up := c.UsePrompt()
	if up == nil {
		return ""
	}
	prompt := *up
	prompt = strings.ReplaceAll(
		prompt, "${{WORK_PATH}}",
		c.GetWorkHome(),
	)
	prompt = strings.ReplaceAll(
		prompt, "${{SUBAGENTS}}",
		c.getSubAgents(c.worker),
	)
	return c.getSystemInfo(prompt)
}

func (c *Context) UsePrompt() *string {
	if c.usePrompt != "" {
		return &c.usePrompt
	}
	var prompt = initial.UsePrompt(c.worker.Type)
	prompt = strings.ReplaceAll(
		prompt, "${{USER_PROMPT}}", c.worker.UsePrompt,
	)

	var tools, _ = c.store.LoadTool()
	var manager = builtin.GetManager().Init(tools)
	prompt = strings.ReplaceAll(
		prompt, "${{BUILTIN_TOOLS}}",
		manager.GetPrompt(c.worker.Tools),
	)

	mcpServ := amcp.GetMcpService(c.store)
	mcpTools := mcpServ.GetPrompt(c.worker)
	if strings.Contains(mcpTools, amcp.NO_TOOL_MSG) {
		return &c.usePrompt
	}
	c.usePrompt = strings.ReplaceAll(
		prompt, "${{MCP_TOOLS}}", mcpTools,
	)
	return &c.usePrompt
}

func (c *Context) getSystemInfo(prompt string) string {
	tag := action.TOOL_RESULT_TAG
	osName, shell := config.GetShellName()
	prompt = strings.ReplaceAll(prompt, "${{OS_NAME}}", osName)
	prompt = strings.ReplaceAll(prompt, "${{SHELL_NAME}}", shell)
	return strings.ReplaceAll(prompt, "${{TOOL_RESULT_TAG}}", tag)
}

func (c *Context) getSubAgents(leader *Worker) string {
	var result strings.Builder
	workers, _ := c.store.LoadBot(
		"leader = ?", leader.UUID,
	)
	for _, worker := range workers {
		if worker.Leader != leader.UUID {
			continue
		}
		result.WriteString(fmt.Sprintf(
			"- **%s** (id: %s): %s\n",
			worker.Name, worker.UUID, worker.Desc,
		))
	}
	if result.Len() == 0 {
		return "empty list"
	}
	return result.String()
}
