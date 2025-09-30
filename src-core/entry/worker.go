package entry

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"swiflow/action"
	"swiflow/agent"
	"swiflow/amcp"
	"swiflow/config"
	"swiflow/support"
	"syscall"
)

// StartWork initializes and runs agents from the specified directory
// Looks for .agent directory containing agent definitions and task.md
func StartWork(path string) {
	// Validate input path
	if path == "" {
		log.Println("[WORKER] Error: empty path provided")
		return
	}

	// Ensure path is absolute
	var workDir, taskContent = "", ""
	if absPath, err := filepath.Abs(path); err != nil {
		log.Printf("[WORKER] Error: failed to get absolute path for %s: %v", path, err)
		return
	} else if _, err := os.Stat(absPath); os.IsNotExist(err) {
		log.Printf("[WORKER] Error: path does not exist: %s", absPath)
		return
	} else {
		workDir = absPath
	}

	// Read task description from task.md
	taskFile := filepath.Join(workDir, "task.md")
	if data, err := os.ReadFile(taskFile); err != nil {
		log.Printf("[WORKER] Warning: failed to read task.md: %v", err)
		taskContent = "No task description provided"
	} else {
		taskContent = string(data)
	}

	// Look for .agent directory
	scanDir := filepath.Join(workDir, ".agent")
	if _, err := os.Stat(scanDir); os.IsNotExist(err) {
		log.Printf("[AGENT] Error: .agent directory not found in %s", workDir)
		return
	}

	// Discover workers in .agent directory
	workers, err := agent.DiscoverWorkers(scanDir)
	if err != nil || len(workers) == 0 {
		log.Printf("[WORKER] Error: failed to discover agents: %v", err)
		return
	}

	// Initialize agent manager
	var manager *agent.Manager
	manager, err = agent.FromAgents(workers)
	if err != nil || manager == nil {
		log.Printf("[WORKER] Error: failed to initialize agent manager: %v", err)
		return
	}

	store, err := manager.GetStorage()
	mcpServ := amcp.GetMcpService(store)
	for _, agent := range workers {
		go mcpServ.LoadMcpServer(agent.McpServers)
	}

	// Create task with unique ID
	taskUUID, _ := support.UniqueID()
	task, err := manager.InitTask("Agent Work: Demo", taskUUID)
	if err != nil {
		log.Printf("[WORKER] Error: failed to initialize task: %v", err)
		return
	} else {
		msgDir := filepath.Join(workDir, ".msgs")
		config.Set("DEBUG_MODE", "yes")
		config.Set("DEBUG_PATH", msgDir)
	}

	log.Printf("[WORKER] Task: %s", taskContent)
	input := &action.UserInput{Content: taskContent}
	var leader *agent.Worker
	for _, worker := range workers {
		if worker.Type == agent.AGENT_LEADER {
			leader = worker
			leader.Home = workDir
			break
		}
	}
	if leader == nil {
		log.Println("[AGENT] Error: no leader agent found")
		return
	}

	// Start the task execution
	manager.Start(input, task, leader)

	// Create channels for task completion and signal handling
	taskDone := make(chan bool, 1)
	sigChan := make(chan os.Signal, 1)

	// Register signal handler for Ctrl+C (SIGINT) and SIGTERM
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Listen for task completion event
	support.Listen("complete", func(uuid string, data any) {
		if uuid == taskUUID {
			log.Printf("[WORKER] Task %s completed via event", taskUUID)
			select {
			case taskDone <- true:
			default: // Non-blocking send in case channel is already closed
			}
		}
	})

	// Wait for either task completion or interrupt signal
	select {
	case <-taskDone:
		log.Printf("[WORKER] Task %s completed successfully", taskUUID)
	case sig := <-sigChan:
		log.Printf("[WORKER] Received signal %v, shutting down gracefully...", sig)
		log.Printf("[WORKER] Task %s interrupted by user", taskUUID)
	}
}
