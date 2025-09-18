package entry

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"swiflow/action"
	"swiflow/agent"
	"swiflow/amcp"
	"swiflow/config"
	"swiflow/entity"
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
	if data, err := readFileContent(taskFile); err != nil {
		log.Printf("[WORKER] Warning: failed to read task.md: %v", err)
		taskContent = "No task description provided"
	} else {
		taskContent = data
	}

	// Discover agents in .agent directory
	agents, err := discoverAgents(workDir)
	if err != nil || len(agents) == 0 {
		log.Printf("[WORKER] Error: failed to discover agents: %v", err)
		return
	}

	// Initialize agent manager
	var manager *agent.Manager
	manager, err = agent.FromAgents(agents)
	if err != nil || manager == nil {
		log.Printf("[WORKER] Error: failed to initialize agent manager: %v", err)
		return
	}

	store, err := manager.GetStorage()
	mcpServ := amcp.GetMcpService(store)
	for _, agent := range agents {
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

	leader, err := manager.GetWorker("leader-office")
	if err != nil || leader == nil {
		log.Println("[AGENT] get worker error", err)
		return
	}

	// Start the task execution
	manager.Start(input, task, leader)

	// Wait for either task completion or interrupt signal
	select {
	case <-taskDone:
		log.Printf("[WORKER] Task %s completed successfully", taskUUID)
	case sig := <-sigChan:
		log.Printf("[WORKER] Received signal %v, shutting down gracefully...", sig)
		log.Printf("[WORKER] Task %s interrupted by user", taskUUID)
	}
}

// discoverAgents finds all .md files in the .agent directory
func discoverAgents(workDir string) ([]*entity.BotEntity, error) {
	var agents []*entity.BotEntity

	// Look for .agent directory
	agentDir := filepath.Join(workDir, ".agent")
	if _, err := os.Stat(agentDir); os.IsNotExist(err) {
		log.Printf("[WORKER] Error: .agent directory not found in %s", workDir)
		return agents, fmt.Errorf(".agent directory not found in: %v", workDir)
	}

	var err error
	var files []os.DirEntry
	if files, err = os.ReadDir(agentDir); err != nil {
		return nil, fmt.Errorf("failed to read agent directory: %v", err)
	}

	for _, file := range files {
		// Skip task.md as it's not an agent definition
		if file.IsDir() || file.Name() == "task.md" {
			continue
		}

		// Only process .md files
		if !strings.HasSuffix(file.Name(), ".md") {
			continue
		}

		agentPath := filepath.Join(agentDir, file.Name())
		content, err := readFileContent(agentPath)
		if err != nil {
			log.Printf("[WORKER] Warning: failed to read agent file %s: %v", file.Name(), err)
			continue
		}

		// Extract agent name from filename (remove .md extension)
		name := strings.TrimSuffix(file.Name(), ".md")
		info := &entity.BotEntity{
			Name: name, UUID: name,
			Home: workDir, SysPrompt: content,
		}
		if strings.HasPrefix(name, "leader-") {
			info.Type = agent.AGENT_LEADER
		} else {
			info.Type = agent.AGENT_WORKER
		}

		jsonPath := strings.Replace(agentPath, ".md", ".json", 1)
		if data, err := os.ReadFile(jsonPath); err == nil {
			var value = map[string]any{}
			json.Unmarshal(data, &value)
			for key, val := range value {
				switch key {
				case "name":
					info.Name, _ = val.(string)
				case "desc":
					info.Desc, _ = val.(string)
				case "mcp", "servers", "mcpServers":
					// 如果有mcps，Tools为[uuid:*]格式
					if mcps, _ := val.(map[string]any); len(mcps) > 0 {
						tools := make([]string, 0, len(mcps))
						for uuid := range mcps {
							tools = append(tools, uuid+":*")
						}
						info.Tools = tools
						info.McpServers = mcps
					}
				}
			}
		}

		agents = append(agents, info)
	}
	return agents, nil
}

// readFileContent reads the content of a file
func readFileContent(filePath string) (string, error) {
	if data, err := os.ReadFile(filePath); err != nil {
		return "", err
	} else {
		return string(data), nil
	}
}
