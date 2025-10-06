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
	STATE_SEEK_HELP = "seek-help" // 任务完成
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
		if r.context.HasMcpError() {
			r.currentState = STATE_FAILED
			log.Println("[EXEC] task", r.UUID, errors.ErrListMcpToolsError)
			support.Emit("errors", r.UUID, errors.ErrListMcpToolsError)
			break
		}

		currMsgId, _ := support.UniqueID()
		messages := r.context.GetContext()

		var lastOp, currOp = "", ""
		var merged, content = "", ""
		for _, queued := range r.msgsQueue {
			content, currOp = queued.Input()
			role := r.context.GetMsgRole(currOp)
			messages = append(messages, &model.Message{
				Content: content, Role: role,
			})
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
				UniqId: currMsgId, PrevId: prevMsgId,
				SendAt: convertor.ToPointer(time.Now()),
			})
		}

		// 调用LLM
		resp := r.GetLLMResp(messages, currMsgId)
		// step 1. save response message
		if resp != nil && resp.Origin != "" {
			r.context.WriteMsg(&MyMsg{
				IsSend: false, Respond: resp.Origin,
				OpType: lastOp, TaskId: r.UUID,
				UniqId: currMsgId, PrevId: prevMsgId,
				RecvAt: convertor.ToPointer(time.Now()),
			})
		}

		// step 2. handle error response
		if resp != nil && resp.ErrMsg != nil {
			r.currentState = STATE_FAILED
			log.Println("[EXEC] task", r.UUID, resp.ErrMsg)
			support.Emit("errors", r.UUID, resp.ErrMsg)
			continue
		}
		// step 3. handle empty response
		if resp != nil && resp.Origin == "" {
			r.currentState = STATE_FAILED
			log.Println("[EXEC] task", r.UUID, errors.ErrEmptyLlmResponse)
			support.Emit("errors", r.UUID, errors.ErrEmptyLlmResponse)
			break
		}

		r.currentTurns += 1
		prevMsgId = currMsgId
		resp.Payload = r.payload
		resp.WorkerID = r.context.GetWorkerId()

		// step 4. execute actions
		toolResult := r.PlayAction(resp)

		// step 5. emit respond event
		support.Emit("respond", r.UUID, resp)
		if r.isTerminated {
			r.currentState = STATE_CANCELED
			log.Println("[EXEC] task", r.UUID, errors.ErrTaskTerminatedByUser)
			support.Emit("errors", r.UUID, errors.ErrTaskTerminatedByUser)
			break
		}

		// step 6. handle complete state
		for _, tool := range resp.UseTools {
			switch tool.(type) {
			case *action.Complete:
				r.currentState = STATE_COMPLETED
			case *action.MakeAsk:
				r.currentState = STATE_SEEK_HELP
			default:
				r.currentState = STATE_WAITING
			}
		}

		// step 7. handle waiting state
		if strings.TrimSpace(toolResult) != "" {
			r.currentState = STATE_WAITING
			r.queueLock.Lock() // tool call has result
			input := []action.Input{&action.ToolResult{
				Content: action.TOOL_RESULT_TAG + "\n" + toolResult,
			}}
			r.msgsQueue = append(input, r.msgsQueue...)
			r.queueLock.Unlock()
			continue
		}
		// step 8. handle waiting state
		switch r.currentState {
		case STATE_SEEK_HELP, STATE_COMPLETED:
			log.Println("[EXEC] task", r.UUID, r.currentState)
		default:
			if resp.Context == nil || resp.Origin == "" {
				continue
			}
			r.queueLock.Lock() // @todo 需要强制继续
			input := []action.Input{&action.UserInput{
				Content: "continue handle task",
			}}
			r.msgsQueue = append(input, r.msgsQueue...)
			r.queueLock.Unlock()
		}
	}
	r.stopFileWatcher()
	r.context.SetState(r.currentState)
	r.currentState, r.currentTurns = "", 0
	return nil
}

func (r *Executor) PlayAction(super *action.SuperAction) string {
	// 并行执行 IAct.Handle，并保持结果顺序
	// 通过有界并发控制最大并发数，避免资源过载
	maxWorkers := config.GetInt("CONCURRENCY", 4)
	sem := make(chan struct{}, maxWorkers)
	wg, replyMsgs := sync.WaitGroup{}, []string{}
	replies := make([]string, len(super.UseTools))
	for idx, tool := range super.UseTools {
		if act, ok := tool.(action.IAct); ok {
			wg.Add(1)
			sem <- struct{}{}
			go func(i int, a action.IAct) {
				defer wg.Done()
				defer func() { <-sem }()
				result := a.Handle(super)
				if support.Bool(result) {
					reply := support.ToXML(a, nil)
					replies[i] = reply
				}
			}(idx, act)
		}
	}
	wg.Wait()

	for _, rpl := range replies {
		if rpl != "" {
			replyMsgs = append(replyMsgs, rpl)
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

func (r *Executor) GetLLMResp(msgs []*model.Message, msgid string) *action.SuperAction {
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
		// first stream of llm response
		if msgid != "" && stream.Idx == 0 {
			worker := r.context.GetWorkerId()
			parts := []string{"data", worker, msgid}
			stream.Str = strings.Join(parts, ":")
			support.Emit("stream", r.UUID, stream)
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
	if r.context.mytask.IsDebug {
		return
	}

	// Get worker directory path
	workPath := r.context.GetWorkHome()
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
