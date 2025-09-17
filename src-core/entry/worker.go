package entry

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"swiflow/action"
	"swiflow/agent"
	"swiflow/entity"
	"swiflow/support"
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
	var workDir, agentDir = "", ""
	if absPath, err := filepath.Abs(path); err != nil {
		log.Printf("[WORKER] Error: failed to get absolute path for %s: %v", path, err)
		return
	} else if _, err := os.Stat(absPath); os.IsNotExist(err) {
		log.Printf("[WORKER] Error: path does not exist: %s", absPath)
		return
	} else {
		workDir = absPath
	}

	// Look for .agent directory
	agentDir = filepath.Join(workDir, ".agent")
	if _, err := os.Stat(agentDir); os.IsNotExist(err) {
		log.Printf("[WORKER] Error: .agent directory not found in %s", workDir)
		return
	}

	// Read task description from task.md
	taskFile := filepath.Join(agentDir, "task.md")
	taskContent, err := readFileContent(taskFile)
	if err != nil {
		log.Printf("[WORKER] Warning: failed to read task.md: %v", err)
		taskContent = "No task description provided"
	}

	// Discover agents in .agent directory
	agents, err := discoverAgents(agentDir)
	if err != nil || len(agents) == 0 {
		log.Printf("[WORKER] Error: failed to discover agents: %v", err)
		return
	}

	// Initialize agent manager
	var manager *agent.Manager
	if manager = agent.FromAgents(agents); manager == nil {
		log.Printf("[WORKER] Error: failed to initialize agent manager")
		return
	}

	// Create task with unique ID
	taskUUID, _ := support.UniqueID()
	bot, err := manager.SelectBot("leader")
	task, err := manager.InitTask("Agent Work: Demo", taskUUID)
	if err != nil {
		log.Printf("[WORKER] Error: failed to initialize task: %v", err)
		return
	}

	// Set task working directory
	task.Home = workDir

	log.Printf("[WORKER] Task: %s", taskContent)
	input := &action.UserInput{Content: taskContent}
	go manager.Handle(input, task, bot)
}

// discoverAgents finds all .md files in the .agent directory
func discoverAgents(agentDir string) ([]*entity.BotEntity, error) {
	var agents []*entity.BotEntity

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
		agentName := strings.TrimSuffix(file.Name(), ".md")
		agents = append(agents, &entity.BotEntity{
			Name: agentName, UUID: agentName,
			Home: agentDir, SysPrompt: content,
		})
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
