package httpd

import (
	"log"
	"swiflow/action"
	"swiflow/agent"
)

type socketInput struct {
	SessID string `json:"sessid,omitempty"`
	TaskID string `json:"taskid,omitempty"`
	Method string `json:"method,omitempty"`
	Action string `json:"action,omitempty"`
	Detail any    `json:"detail,omitempty"`
}

type WebSocketHandler struct {
	source string
	sessid string

	manager *agent.Manager
	taskMap map[string]*agent.MyTask
}

func NewWsHandler(m *agent.Manager, sessid, source string) *WebSocketHandler {
	return &WebSocketHandler{
		source: source, sessid: sessid, manager: m,
		taskMap: map[string]*agent.MyTask{},
	}
}

func (m *WebSocketHandler) shouldHandle(tid string, _ any) bool {
	// cache session id, from task info
	if _, ok := m.taskMap[tid]; !ok {
		taskInfo, err := m.manager.QueryTask(tid)
		if err == nil && taskInfo != nil {
			m.taskMap[tid] = taskInfo
		}
	}

	var task *agent.MyTask
	if t, ok := m.taskMap[tid]; !ok {
		return false
	} else if t != nil {
		task = t
	}

	if m.source == "im-proxy" {
		return task.Source == m.source
	} else {
		return task.SessID == m.sessid
	}
}
func (m *WebSocketHandler) getSessID(tid string) string {
	var task *agent.MyTask
	if t, ok := m.taskMap[tid]; !ok {
		return ""
	} else if t != nil {
		task = t
	}
	if m.source == "im-proxy" {
		return task.SessID
	}
	return ""
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
	resp, ok := data.(*action.SuperAction)
	if !ok {
		log.Println("[WS] resp errors: ", "未知错误")
		return nil
	}

	// handle errmsg
	if resp.ErrMsg != nil {
		err := resp.ErrMsg.Error()
		log.Println("[WS] resp errors:", err)
		return &socketInput{
			Method: "message", Action: "errors",
			Detail: err, TaskID: task,
			SessID: m.getSessID(task),
		}
	}
	// handle success
	return &socketInput{
		Method: "message",
		Action: "respond",
		Detail: resp.ToMap(),
		TaskID: task,
		SessID: m.getSessID(task),
	}
}

func (m *WebSocketHandler) DoStream(task string, state any) *socketInput {
	return &socketInput{
		Method: "message", Action: "stream",
		Detail: state, TaskID: task,
		SessID: m.getSessID(task),
	}
}

func (m *WebSocketHandler) DoControl(task string, state any) *socketInput {
	return &socketInput{
		Method: "message", Action: "control",
		Detail: state, TaskID: task,
		SessID: m.getSessID(task),
	}
}

func (m *WebSocketHandler) HandleErr(task string, data any) *socketInput {
	msg := &socketInput{
		Method: "message", Action: "errors",
		Detail: data, TaskID: task,
		SessID: m.getSessID(task),
	}
	if err, ok := data.(error); ok {
		msg.Detail = err.Error()
	}
	return msg
}

// 新增文件监控处理方法
func (m *WebSocketHandler) DoChange(task string, data any) *socketInput {
	return &socketInput{
		Method: "message", Action: "change",
		Detail: data, TaskID: task,
		SessID: m.getSessID(task),
	}
}
