package httpd

import (
	"encoding/json"
	"log"
	"swiflow/agent"
	"swiflow/support"

	"github.com/gorilla/websocket"
)

// WebSocketSession 管理WebSocket连接的生命周期
type WebSocketSession struct {
	conn  *websocket.Conn
	msgs  chan any
	done  chan struct{}
	logic *WebSocketHandler

	handlers map[string]support.EventHandler
}

// 创建一个新的WebSocket会话
func NewWSSession(conn *websocket.Conn, manager *agent.Manager) *WebSocketSession {
	session := &WebSocketSession{
		conn:  conn,
		msgs:  make(chan any, 10),
		done:  make(chan struct{}),
		logic: NewWsHandler(manager),

		handlers: make(map[string]support.EventHandler),
	}

	// 启动写消息的goroutine
	go session.writeLoop()

	// 设置事件监听
	session.setupEvent()

	return session
}

// 启动写消息循环
func (s *WebSocketSession) writeLoop() {
	defer close(s.msgs)

	for {
		select {
		case msg, ok := <-s.msgs:
			if !ok {
				return // 通道被关闭，退出 goroutine
			}
			if err := s.conn.WriteJSON(msg); err != nil {
				log.Println("[WS] Write message error:", err)
				return
			}
		case <-s.done:
			return // 收到关闭信号，退出 goroutine
		}
	}
}

// 设置事件监听器
func (s *WebSocketSession) setupEvent() {
	eventTypes := []string{
		"respond", "stream",
		"control", "errors",
		"change",
	}
	handlers := []func(task string, data any) *socketInput{
		s.logic.DoRespond, s.logic.DoStream,
		s.logic.DoControl, s.logic.HandleErr,
		s.logic.DoChange,
	}

	for i, eventType := range eventTypes {
		handlerFunc := s.createHandler(handlers[i])
		s.handlers[eventType] = handlerFunc
		support.Listen(eventType, handlerFunc)
	}
}

// 创建事件处理函数
func (s *WebSocketSession) createHandler(processor func(task string, data any) *socketInput) support.EventHandler {
	return func(task string, data any) {
		select {
		case <-s.done:
			return // 如果连接已关闭，不发送消息
		default:
			if msg := processor(task, data); msg != nil {
				select {
				case s.msgs <- msg: // 尝试发送消息
				case <-s.done: // 如果连接已关闭，不发送消息
					return
				}
			}
		}
	}
}

// 关闭会话并清理资源
func (s *WebSocketSession) Close() {
	s.conn.Close()
	close(s.done) // 发出关闭信号

	// 清理事件监听器
	for eventType, handler := range s.handlers {
		support.Remove(eventType, handler)
	}
}

// 处理接收到的消息
func (s *WebSocketSession) Handle() {
	for {
		msg, err := s.readMessage()
		if err != nil {
			break // 出错时退出循环
		}

		reply := s.processMessage(msg)
		s.msgs <- reply
	}
}

// 读取并解析消息
func (s *WebSocketSession) readMessage() (*socketInput, error) {
	var msg = new(socketInput)
	var expectedCodes = []int{
		websocket.CloseGoingAway,
		websocket.CloseAbnormalClosure,
	}

	_, data, err := s.conn.ReadMessage()
	if err != nil {
		if websocket.IsUnexpectedCloseError(err, expectedCodes...) {
			log.Println("[WS] Connect is close with error:", err)
		} else if websocket.IsCloseError(err, websocket.CloseGoingAway) {
			log.Println("[WS] Connect is going away, bye bye~")
		}
		return nil, err
	}

	if err := json.Unmarshal(data, msg); err != nil {
		log.Println("[WS] Unmarshal message error:", err)
		return nil, err
	}

	log.Println("[WS] receive msg: ", msg.Detail)
	return msg, nil
}

// 处理消息并返回响应
func (s *WebSocketSession) processMessage(msg *socketInput) *socketInput {
	var reply *socketInput

	switch msg.Method {
	case "system":
		reply = s.logic.OnSystem(msg)
	case "control":
		reply = s.logic.OnControl(msg)
	case "message":
		reply = s.logic.OnMessage(msg)
	default:
		log.Println("[WS] msg method error:", msg.Method)
	}

	return reply
}
