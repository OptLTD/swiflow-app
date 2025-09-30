package agent

import (
	"fmt"
	"log"
	"swiflow/action"
	"swiflow/support"
)

type SubAgent struct {
	parent *Manager
	worker *Worker // current worker
	mytask *MyTask // current task
	leader *Worker // leader worker
	ldtask *MyTask // leader task
}

// OnStart handles the start of a subtask by selecting a worker bot and initializing the subtask
func (sa *SubAgent) OnStart(act *action.StartSubtask) {
	worker, err := sa.parent.GetWorker(act.SubAgent)
	if err != nil || worker == nil {
		act.Result = fmt.Sprintf(
			"agent(%s) not found %v",
			act.SubAgent, err,
		)
		toolResult := support.ToXML(act, act.Result)
		input := &action.ToolResult{
			Content: action.TOOL_RESULT_TAG + "\n" + toolResult,
		}
		sa.parent.Handle(input, sa.ldtask, sa.leader)
		return
	}
	log.Println("[SUBTASK] bot", worker.UUID)
	// leader arrange subtask to worker
	// need push subtask to worker
	newtask, err := sa.parent.InitSubtask(
		worker.UUID, sa.ldtask.Group,
	)
	if err == nil && newtask != nil {
		log.Println("[SUBTASK] OnStart", worker.UUID)
		// ensure work in same dir
		worker.Home = sa.leader.Home
		newtask.IsDebug = sa.ldtask.IsDebug
		sa.worker, sa.mytask = worker, newtask
		sa.parent.Handle(act.ToSubtask(), newtask, worker)
	}
}

// OnAbort handles the abortion of a subtask by terminating the active executor
func (sa *SubAgent) OnAbort(act *action.AbortSubtask) {
	toolResult := &action.ToolResult{}
	if sa.worker == nil || sa.worker.UUID != act.SubAgent {
		result := fmt.Sprintf("agent(%s) not found", act.SubAgent)
		toolResult.Content = support.ToXML(act, result)
		sa.parent.Handle(toolResult, sa.ldtask, sa.leader)
		return
	}

	// terminate subtask
	executor := sa.parent.LoadExecutor(sa.ldtask, sa.leader)
	if executor != nil && executor.IsRunning() {
		executor.Terminate()
		toolResult.Content = support.ToXML(act, "success")
		sa.parent.Handle(toolResult, sa.ldtask, sa.leader)
	} else {
		toolResult.Content = support.ToXML(act, "no subtask found")
		sa.parent.Handle(toolResult, sa.mytask, sa.worker)
	}
}

// OnComplete handles the completion of a subtask by updating the leader task context
func (sa *SubAgent) OnComplete(act *action.Complete) {
	// task of worker context update
	// need push context to leader
	toolResult := &action.ToolResult{}
	subtask := &action.StartSubtask{
		SubAgent: sa.worker.UUID,
	}
	toolResult.Content = support.ToXML(subtask, act.Content)
	log.Println("[SUBTASK] OnComplete", sa.worker.UUID)
	sa.parent.Handle(toolResult, sa.ldtask, sa.worker)
}

// OnTimeout handles the timeout of a subtask by updating the leader task context
func (sa *SubAgent) OnTimeout(act *action.Complete) {
	// task of worker context update
	// need push context to leader
	toolResult := &action.ToolResult{}
	subtask := &action.StartSubtask{
		SubAgent: sa.worker.UUID,
	}
	toolResult.Content = support.ToXML(subtask, act.Content)
	log.Println("[SUBTASK] OnTimeout", sa.worker.UUID)
	sa.parent.Handle(toolResult, sa.ldtask, sa.worker)
}
