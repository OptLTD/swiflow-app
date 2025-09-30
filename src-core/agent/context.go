package agent

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"swiflow/action"
	"swiflow/config"
	"swiflow/entity"
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

func (r *Context) Get() []*model.Message {
	return r.GetContext()
}

func (r *Context) GetGroup() string {
	return r.mytask.Group
}

func (r *Context) GetWorkId() string {
	return r.worker.UUID
}

func (r *Context) GetWorkHome() string {
	return r.worker.Home
}

func (r *Context) GetMsgRole(op string) string {
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
func (r *Context) TaskContext() string {
	return r.mytask.Context
}

func (r *Context) ParseMsgs(msgs []*MyMsg) []*action.SuperAction {
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
			if len(resp.Errors) == 0 {
				resMap[resp.TheMsgId] = resp
				result = append(result, resp)
			}
		}
	}
	return result
}

func (r *Context) GetMsgs(count int) []*model.Message {
	messages := make([]*model.Message, 0)
	msgs, _ := r.store.LoadMsg(r.mytask)
	for i := 0; i < len(msgs); i++ {
		if count > 0 && i+count <= len(msgs) {
			continue
		}

		req := model.Message{Content: msgs[i].Request}
		req.Role = r.GetMsgRole(msgs[i].OpType)
		messages = append(messages, &req)
		if strings.TrimSpace(msgs[i].Respond) != "" {
			messages = append(messages, &model.Message{
				Role: "assistant", Content: msgs[i].Respond,
			})
		}
	}
	return messages
}

func (r *Context) GetContext() []*model.Message {
	messages := make([]*model.Message, 0)
	if support.Bool(r.usePrompt) {
		messages = append(messages, &model.Message{
			Role: "system", Content: r.usePrompt,
		})
	}
	if support.Bool(r.useMemory) {
		messages = append(messages, &model.Message{
			Role: "user", Content: r.useMemory,
		})
	}
	size := config.GetInt("CTX_MSG_SIZE", 100)
	if msgs := r.GetMsgs(size); len(msgs) > 0 {
		messages = append(messages, msgs...)
	}
	return messages
}

func (r *Context) GetSubject() string {
	return r.mytask.Name
}

func (r *Context) DebugCall(op string, msgs []*model.Message) {
	log.Println("[EXEC]", r.worker.Name, r.mytask.UUID, op)
	if r.mytask.IsDebug == false {
		return
	}

	var s strings.Builder
	for i, msg := range msgs {
		if i == 0 {
			continue
		}
		s.WriteString("<--- " + msg.Role + " --->")
		s.WriteString("\n" + msg.Content + "\n\n")
	}
	path := support.Or(r.mytask.Home, config.GetWorkHome())
	if fileutil.CreateDir(filepath.Join(path, ".msgs")) != nil {
		return
	}

	history := filepath.Join(path, ".msgs", r.worker.UUID+".xml")
	fileutil.WriteStringToFile(history, s.String(), false)
	prompt := filepath.Join(path, ".msgs", r.worker.UUID+".md")
	fileutil.WriteStringToFile(prompt, r.usePrompt, false)
}

func (r *Context) Memorize(act *action.Memorize) error {
	mem := &entity.MemEntity{
		Bot: r.worker.UUID, Type: "chat",
		Subject: act.Subject,
		Content: act.Content,
	}
	return r.store.SaveMem(mem)
}

func (r *Context) Annotate(act *action.Annotate) error {
	if act.Subject != "" {
		r.mytask.Name = act.Subject
	}
	r.mytask.Context = act.Context
	return nil
}

func (r *Context) WaitTodo(act *action.WaitTodo) (err error) {
	todo := &entity.TodoEntity{UUID: act.UUID, Task: r.mytask.UUID}
	if act.UUID == "" && (act.Time == "" || act.Todo == "") {
		return fmt.Errorf("error: wrong wait-todo format")
	}
	if act.UUID == "" && act.Time != "" && act.Todo != "" {
		todo.UUID, _ = support.UniqueID(12)
		todo.Time, todo.Todo = act.Time, act.Todo
		if err = r.store.SaveTodo(todo); err == nil {
			support.Emit("wait-todo", "create", todo)
			return
		}
		return fmt.Errorf("error: %w", err)
	}

	// find todo
	if err = r.store.FindTodo(todo); err != nil {
		return fmt.Errorf("error: %w", err)
	}
	if act.Time != "" && act.Todo != "" {
		todo.Time, todo.Todo = act.Time, act.Todo
	} else {
		todo.Done = 1
	}
	if err = r.store.SaveTodo(todo); err == nil {
		event := support.If(todo.Done, "remove", "update")
		support.Emit("wait-todo", event, todo)
	}
	return
}

func (r *Context) SetState(state string) error {
	r.mytask.State = state
	support.Emit("control", r.mytask.UUID, state)
	if err := r.store.SaveTask(r.mytask); err != nil {
		return fmt.Errorf("save state error: %v", err)
	}
	return nil
}

func (r *Context) WriteMsg(msg *MyMsg) error {
	if r.store == nil {
		log.Println("[EXEC]", msg.TaskId, "log msg fail: store nil")
		return fmt.Errorf("write msg error: store nil")
	}
	// add msg group of task
	if r.mytask.Group != "" {
		msg.Group = r.mytask.Group
	}
	if err := r.store.SaveMsg(msg); err != nil {
		log.Println("[EXEC]", msg.TaskId, "log msg fail:", err)
		return fmt.Errorf("write msg error: %v", err)
	}
	return nil
}
