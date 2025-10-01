package httpd

import (
	"log"
	"swiflow/action"
	"swiflow/agent"
)

type socketInput struct {
	TaskID string `json:"taskid"`
	Method string `json:"method"`
	Action string `json:"action"`
	Detail any    `json:"detail"`
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
	return msg
}

func (m *WebSocketHandler) DoRespond(task string, data any) *socketInput {
	if resp, ok := data.(*action.SuperAction); !ok {
		log.Println("[WS] resp errors: ", "未知错误")
		return nil
	} else if resp.ErrMsg != nil {
		err := resp.ErrMsg.Error()
		log.Println("[WS] resp errors:", err)
		return &socketInput{"message", "errors", err, task}
	} else {
		return &socketInput{
			Method: "message",
			Action: "respond",
			Detail: resp.ToMap(),
			TaskID: task,
		}
	}
}

func (m *WebSocketHandler) DoStream(task string, state any) *socketInput {
	return &socketInput{
		Method: "message",
		Action: "stream",
		Detail: state,
		TaskID: task,
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
		TaskID: task,
	}
}

func (m *WebSocketHandler) HandleErr(task string, data any) *socketInput {
	msg := &socketInput{
		Method: "message",
		Action: "errors",
		Detail: data,
		TaskID: task,
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
		TaskID: task,
	}
}
