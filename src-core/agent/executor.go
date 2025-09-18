package agent

import (
	"fmt"
	"log"
	"slices"
	"strings"
	"swiflow/action"
	"swiflow/config"
	"swiflow/errors"
	"swiflow/model"
	"swiflow/support"
	"time"

	"sync"

	"github.com/duke-git/lancet/v2/convertor"
)

type Executor struct {
	UUID string

	context *Context
	payload *Payload

	queueLock sync.Mutex
	msgsQueue []action.Input

	isTerminated bool   // 终止任务
	currentTurns int    // 当前轮次
	currentState string // 当前状态

	modelClient model.LLMClient
	fileWatcher *support.FileWatcher
}

const (
	STATE_RUNNING = "running" // 程序正在运行
	STATE_WAITING = "waiting" // 等待人工干预

	STATE_FAILED    = "failed"    // 执行失败
	STATE_CANCELED  = "canceled"  // 取消执行
	STATE_COMPLETED = "completed" // 任务完成
)

const (
	AGENT_DEBUG = "debug"
	AGENT_BASIC = "basic"

	AGENT_LEADER = "leader"
	AGENT_WORKER = "worker"
)

func (r *Executor) Resume() error {
	r.currentTurns = 0
	r.isTerminated = false
	r.Handle()
	return nil
}

func (r *Executor) Terminate() error {
	r.isTerminated = true
	return nil
}

func (r *Executor) IsRunning() bool {
	return r.currentTurns > 0
}

func (r *Executor) Enqueue(input action.Input) {
	r.queueLock.Lock()
	defer r.queueLock.Unlock()
	if r.msgsQueue == nil {
		r.msgsQueue = make([]action.Input, 0)
	}
	r.isTerminated = false
	r.msgsQueue = append(r.msgsQueue, input)
	if !r.IsRunning() {
		go r.Handle()
	}
}

func (r *Executor) Handle() *action.SuperAction {
	var prevMsgId string
	if r.isTerminated {
		r.context.SetState(STATE_CANCELED)
		return nil
	}

	r.startFileWatcher()
	r.currentState = STATE_RUNNING
	r.context.SetState(STATE_RUNNING)
	for {
		if len(r.msgsQueue) == 0 {
			break
		}

		maxTurns := config.GetInt("MAX_CALL_TURNS", 25)
		if maxTurns > 0 && r.currentTurns > maxTurns {
			r.currentState = STATE_WAITING
			log.Println("[EXEC] task", r.UUID, errors.ErrExceededMaximumTurns)
			support.Emit("errors", r.UUID, errors.ErrExceededMaximumTurns)
			break
		}
		if r.isTerminated {
			r.currentState = STATE_CANCELED
			log.Println("[EXEC] task", r.UUID, errors.ErrTaskTerminatedByUser)
			support.Emit("errors", r.UUID, errors.ErrTaskTerminatedByUser)
			break
		}
		msgid, _ := support.UniqueID()
		messages := r.context.GetContext()

		var lastOp, currOp = "", ""
		var merged, content = "", ""
		for _, queued := range r.msgsQueue {
			content, currOp = queued.Input()
			role := r.context.GetRole(currOp)
			messages = append(messages, &model.Message{
				Content: content, Role: role,
			})
			// taskContext := r.context.TaskContext()
			// isToolResult := currOp == action.TOOL_RESULT
			// if taskContext != "" && isToolResult { // 强制继续推理
			// 	messages = append(messages, &model.Message{
			// 		Content: taskContext, Role: "user",
			// 	})
			// }
			if merged != "" {
				merged += "\n"
			}
			merged += content
			lastOp = currOp
		}
		r.msgsQueue = nil
		if merged != "" {
			r.context.WriteMsg(&MyMsg{
				IsSend: true, Request: merged,
				OpType: lastOp, TaskId: r.UUID,
				MsgId: msgid, PreMsg: prevMsgId,
				SendAt: convertor.ToPointer(time.Now()),
			})
		}

		// 调用LLM
		resp := r.CallLLM(messages)
		if err := len(resp.Errors); err > 0 {
			r.currentState = STATE_FAILED
			log.Println("[EXEC] task", r.UUID, resp.Errors[0])
			support.Emit("errors", r.UUID, resp.Errors[0])
			continue
		}

		// 保存LLM响应内容
		if resp != nil && resp.Origin != "" {
			r.context.WriteMsg(&MyMsg{
				IsSend: false, Respond: resp.Origin,
				OpType: lastOp, TaskId: r.UUID,
				MsgId: msgid, PreMsg: prevMsgId,
				RecvAt: convertor.ToPointer(time.Now()),
			})
		} else if resp != nil && resp.Origin == "" {
			r.currentState = STATE_FAILED
			log.Println("[EXEC] task", r.UUID, errors.ErrEmptyLlmResponse)
			support.Emit("errors", r.UUID, errors.ErrEmptyLlmResponse)
			break
		}

		r.currentTurns += 1
		resp.Payload = r.payload

		// 执行动作
		prevMsgId = msgid
		toolResult := r.DoPlay(resp)

		// 触发 respond 事件
		support.Emit("respond", r.UUID, resp)
		if r.isTerminated {
			r.currentState = STATE_CANCELED
			log.Println("[EXEC] task", r.UUID, errors.ErrTaskTerminatedByUser)
			support.Emit("errors", r.UUID, errors.ErrTaskTerminatedByUser)
			break
		}

		// handle complete state
		for _, tool := range resp.UseTools {
			switch tool.(type) {
			case *action.Complete:
				r.currentState = STATE_COMPLETED
			default:
				r.currentState = STATE_WAITING
			}
		}

		if strings.TrimSpace(toolResult) != "" {
			r.currentState = STATE_WAITING
			r.queueLock.Lock() // tool call has result
			input := []action.Input{&action.ToolResult{
				Content: action.TOOL_RESULT_TAG + "\n" + toolResult,
			}}
			r.msgsQueue = append(input, r.msgsQueue...)
			r.queueLock.Unlock()
			continue
		} else if resp.Context != nil {
			// @todo 需要判断是否任务结束了
			// r.queueLock.Lock() // maybe update context
			// input := []action.Input{&action.ToolResult{}}
			// r.msgsQueue = append(input, r.msgsQueue...)
			// r.queueLock.Unlock()
			continue
		} else {
			log.Println("[EXEC] task", r.UUID, STATE_COMPLETED)
		}
	}
	r.stopFileWatcher()
	r.context.SetState(r.currentState)
	r.currentState, r.currentTurns = "", 0
	return nil
}

func (r *Executor) DoPlay(super *action.SuperAction) string {
	replyMsgs := []string{}
	for _, tool := range super.UseTools {
		switch act := tool.(type) {
		case action.IAct:
			result := act.Handle(super)
			if support.Bool(result) {
				reply := support.ToXML(act, nil)
				replyMsgs = append(replyMsgs, reply)
			}
		}
	}
	if super.Context != nil {
		r.context.Annotate(super.Context)
	}
	for _, tool := range super.UseTools {
		switch act := tool.(type) {
		case *action.Memorize:
			r.context.Memorize(act)
		case *action.WaitTodo:
			r.context.WaitTodo(act)
		case *action.Complete:
			r.SendNotify("complete")
			support.Emit("complete", r.UUID, act)
		case *action.StartSubtask:
			support.Emit("subtask", r.UUID, act)
		case *action.QuerySubtask:
			support.Emit("subtask", r.UUID, act)
		case *action.AbortSubtask:
			support.Emit("subtask", r.UUID, act)
		}
	}
	if len(replyMsgs) == 0 {
		return ""
	}
	return strings.Join(replyMsgs, "\n\n")
}

func (r *Executor) CallLLM(msgs []*model.Message) *action.SuperAction {
	var choice = new(model.Choice)
	var data = make([]model.Message, 0)
	for _, msg := range msgs {
		data = append(data, *msg)
	}
	r.context.DebugCall("start call llm", msgs)
	if config.GetStr("STREAM_OUTPUT", "yes") != "yes" {
		if resp, err := r.modelClient.Respond(r.UUID, data); err != nil {
			return action.Errors(err)
		} else if len(resp) > 0 {
			choice.Message.Content = resp[0].Message.Content
		}
	} else {
		var stream struct {
			Idx uint32 `json:"idx"`
			Str string `json:"str"`
		}
		err := r.modelClient.Stream(r.UUID, data, func(choices []model.Choice) {
			choice.Message.Content += choices[0].Message.Content
			stream.Idx, stream.Str = stream.Idx+1, choices[0].Message.Content
			support.Emit("stream", r.UUID, stream)
		})
		if err != nil {
			return action.Errors(err)
		}
	}

	reply := choice.Message.Content
	msgs = append(msgs, &model.Message{
		Role: "assistant", Content: reply,
	})
	r.context.DebugCall("get llm reply", msgs)
	if reply != "" {
		return action.Parse(reply)
	}
	return action.Errors(
		fmt.Errorf("empty response of llm"),
	)
}

func (r *Executor) SendNotify(kind string) error {
	sendNotify := config.GetStr("SEND_NOTIFY_ON", "")
	conditions := strings.Split(sendNotify, ",")
	if !slices.Contains(conditions, kind) {
		return nil
	}
	go func() {
		if file, e := config.NotifyLock(); e == nil {
			defer file.Close()
			file.WriteString(r.context.GetSubject())
		}
	}()
	return nil
}

// Start file watcher
func (r *Executor) startFileWatcher() {
	if r.fileWatcher != nil {
		return
	}

	// Get worker directory path
	workPath := r.context.worker.Home
	if workPath == "" {
		log.Printf("[EXEC] task Unable to get worker dir path, skipping file monitoring")
		return
	}

	// Create file watcher
	watcher, err := support.NewFileWatcher(workPath, r.UUID)
	if err != nil {
		log.Printf("[EXEC] task Failed to create file watcher: %v", err)
		return
	}

	// Start monitoring
	if err := watcher.Start(); err != nil {
		log.Printf("[EXEC] task Failed to start file monitoring: %v", err)
		return
	}

	r.fileWatcher = watcher
	log.Printf("[EXEC] task File monitoring started: %s", workPath)
}

// Stop file watcher
func (r *Executor) stopFileWatcher() {
	if r.fileWatcher != nil {
		r.fileWatcher.Stop()
		r.fileWatcher = nil
		log.Printf("[EXEC] task File monitoring stopped")
	}
}
