package agent

import (
	"fmt"
	"log"
	"swiflow/action"
)

type SubAgent struct {
	parent *Manager
	worker *Worker // worker
	mytask *MyTask // mytask
	leader *Worker // leader
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
		sa.parent.Handle(act, sa.ldtask, sa.leader)
		return
	}
	log.Println("[SUBTASK] bot", worker.UUID)
	// leader arrange subtask to worker
	// need push subtask to worker
	subtask, err := sa.parent.InitSubtask(
		worker.UUID, sa.ldtask.Group,
	)
	if err == nil && subtask != nil {
		// ensure work in same dir
		worker.Home = sa.leader.Home
		sa.worker, sa.mytask = worker, subtask
		log.Println("[SUBTASK] OnStart", worker.UUID)
		sa.parent.Handle(act, subtask, worker)
	}
}

// OnAbort handles the abortion of a subtask by terminating the active executor
func (sa *SubAgent) OnAbort(act *action.AbortSubtask) {
	if sa.worker == nil || sa.worker.UUID != act.SubAgent {
		act.Result = fmt.Sprintf("agent(%s) not found", act.SubAgent)
		sa.parent.Handle(act, sa.ldtask, sa.leader)
		return
	}

	// terminate subtask
	executor := sa.parent.LoadExecutor(sa.ldtask, sa.leader)
	if executor != nil && executor.IsRunning() {
		executor.Terminate()
		act.Result = "success"
		sa.parent.Handle(act, sa.ldtask, sa.leader)
	} else {
		act.Result = "no subtask found"
		sa.parent.Handle(act, sa.mytask, sa.worker)
	}
}

// OnComplete handles the completion of a subtask by updating the leader task context
func (sa *SubAgent) OnComplete(act *action.Complete) {
	// task of worker context update
	// need push context to leader
	subtask := &action.StartSubtask{
		SubAgent: sa.worker.UUID,
		Result:   act.Content,
	}
	log.Println("[SUBTASK] OnComplete", sa.worker.UUID)
	sa.parent.Handle(subtask, sa.ldtask, sa.worker)
}

// OnTimeout handles the timeout of a subtask by updating the leader task context
func (sa *SubAgent) OnTimeout(act *action.Complete) {
	// task of worker context update
	// need push context to leader
	subtask := &action.StartSubtask{
		SubAgent: sa.worker.UUID,
		Result:   act.Content,
	}
	log.Println("[SUBTASK] OnTimeout", sa.worker.UUID)
	sa.parent.Handle(subtask, sa.ldtask, sa.worker)
}
