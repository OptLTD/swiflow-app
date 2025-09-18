package httpd

import (
	"fmt"
	"log"
	"strings"
	"swiflow/action"
	"swiflow/agent"
	"swiflow/config"
	"swiflow/support"

	"github.com/duke-git/lancet/v2/convertor"
)

type socketInput struct {
	Method string `json:"method"`
	Action string `json:"action"`
	Detail any    `json:"detail"`
	ChatID string `json:"chatid"`
}

type WebSocketHandler struct {
	manager *agent.Manager
}

func NewWsHandler(m *agent.Manager) *WebSocketHandler {
	return &WebSocketHandler{m}
}

func (m *WebSocketHandler) OnSystem(msg *socketInput) *socketInput {
	var res = &socketInput{Method: msg.Method}
	switch msg.Action {
	case "ping":
		res.Action = "pong"
	case "hello":
		res.Action = "welcome"
	}
	return res
}

func (m *WebSocketHandler) OnControl(msg *socketInput) *socketInput {
	var res = &socketInput{Method: msg.Method}
	switch msg.Action {
	case "cancel": // cancel running request
	}
	return res
}

func (m *WebSocketHandler) OnMessage(msg *socketInput) *socketInput {
	input := &action.UserInput{}
	detail, _ := msg.Detail.(map[string]any)
	if content, ok := detail["content"].(string); ok {
		input.Content = strings.TrimSpace(content)
	}
	if uploads, ok := detail["uploads"].([]any); ok {
		input.Uploads = []string{}
		for _, item := range uploads {
			input.Uploads = append(input.Uploads, fmt.Sprint(item))
		}
	}

	uuid := config.GetStr("USE_WORKER", "")
	worker, err := m.manager.GetWorker(uuid)
	if err != nil || worker == nil {
		support.Emit("errors", "", "get worker error")
		return msg
	}

	var task *agent.MyTask
	if yes, _ := detail["newTask"].(string); yes == "yes" {
		task, err = m.manager.InitTask(input.Content, msg.ChatID)
	} else if strings.HasPrefix(msg.ChatID, "#debug#") {
		worker = convertor.DeepClone(worker)
		task, err = m.manager.NewMcpTask(msg.ChatID)
	} else {
		task, err = m.manager.QueryTask(msg.ChatID)
	}
	if task == nil || err != nil {
		support.Emit("errors", "", "query task error")
		return msg
	}

	go m.manager.Handle(input, task, worker)
	return msg
}

func (m *WebSocketHandler) DoRespond(task string, data any) *socketInput {
	if resp, ok := data.(*action.SuperAction); !ok {
		log.Println("[WS] resp errors: ", "未知错误")
		return nil
	} else if len(resp.Errors) > 0 {
		err := resp.Errors[0].Error()
		log.Println("[WS] resp errors:", resp.Errors)
		return &socketInput{"message", "errors", err, task}
	} else {
		return &socketInput{
			Method: "message",
			Action: "respond",
			Detail: resp.ToMap(),
			ChatID: task,
		}
	}
}

func (m *WebSocketHandler) DoStream(task string, state any) *socketInput {
	return &socketInput{
		Method: "message",
		Action: "stream",
		Detail: state,
		ChatID: task,
	}
}

func (m *WebSocketHandler) DoExecute(task string, data any) *socketInput {
	return nil
}

func (m *WebSocketHandler) DoControl(task string, state any) *socketInput {
	return &socketInput{
		Method: "message",
		Action: "control",
		Detail: state,
		ChatID: task,
	}
}

func (m *WebSocketHandler) HandleErr(task string, data any) *socketInput {
	msg := &socketInput{
		Method: "message",
		Action: "errors",
		Detail: data,
		ChatID: task,
	}
	if err, ok := data.(error); ok {
		msg.Detail = err.Error()
	}
	return msg
}

// 新增文件监控处理方法
func (m *WebSocketHandler) DoChange(task string, data any) *socketInput {
	return &socketInput{
		Method: "message",
		Action: "change",
		Detail: data,
		ChatID: task,
	}
}
