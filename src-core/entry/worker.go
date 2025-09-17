package entry

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"swiflow/action"
	"swiflow/agent"
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
	if err != nil {
		log.Printf("[WORKER] Error: failed to discover agents: %v", err)
		return
	}

	if len(agents) == 0 {
		log.Printf("[WORKER] Error: no agents found in %s", agentDir)
		return
	}

	// Initialize agent manager
	var manager *agent.Manager
	if manager, err = agent.NewManager(); err != nil {
		log.Printf("[WORKER] Error: failed to initialize agent manager: %v", err)
		return
	}

	// Find master agent (master.md has priority)
	var masterAgent *AgentInfo
	var subAgents []*AgentInfo

	for _, agentInfo := range agents {
		if agentInfo.Name == "master" {
			masterAgent = agentInfo
		} else {
			subAgents = append(subAgents, agentInfo)
		}
	}

	// If no master.md found, use the first agent as master
	if masterAgent == nil && len(agents) > 0 {
		masterAgent, subAgents = agents[0], agents[1:]
		log.Printf("[WORKER] No master.md found, using %s as master agent", masterAgent.Name)
	}

	// Create task with unique ID
	taskUUID, _ := support.UniqueID()

	// Create user input with task content and working directory context
	input := &action.UserInput{
		Content: fmt.Sprintf("Working Directory: %s\n\nTask Description:\n%s\n\nAvailable Sub-Agents: %s",
			workDir, taskContent, getSubAgentNames(subAgents)),
	}

	// Get or create master bot
	bot, err := manager.SelectBot("master")
	if err != nil {
		// If master bot doesn't exist, try to get the first available bot
		bot, err = manager.SelectBot("")
		if err != nil {
			log.Printf("[WORKER] Error: failed to get bot: %v", err)
			return
		}
	}

	// Set working directory for the bot
	bot.Home = workDir

	// Initialize task
	task, err := manager.InitTask(fmt.Sprintf("Agent Work: %s", filepath.Base(workDir)), taskUUID)
	if err != nil {
		log.Printf("[WORKER] Error: failed to initialize task: %v", err)
		return
	}

	// Set task working directory
	task.Home = workDir

	log.Printf("[WORKER] Starting agent work in directory: %s", workDir)
	log.Printf("[WORKER] Master Agent: %s", masterAgent.Name)
	log.Printf("[WORKER] Sub-Agents: %v", getSubAgentNames(subAgents))
	log.Printf("[WORKER] Task: %s", taskContent)

	// Start agent execution (similar to ws_handle.go OnMessage pattern)
	go manager.Handle(input, task, bot)
}

// AgentInfo holds information about discovered agents
type AgentInfo struct {
	Type string
	Name string
	Path string
	Desc string
}

// discoverAgents finds all .md files in the .agent directory
func discoverAgents(agentDir string) ([]*AgentInfo, error) {
	var agents []*AgentInfo

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
		agentType := support.If(agentName == "master", "master", "slave")
		agents = append(agents, &AgentInfo{
			Name: agentName, Path: agentPath,
			Type: agentType, Desc: content,
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

// getSubAgentNames returns a comma-separated list of sub-agent names
func getSubAgentNames(subAgents []*AgentInfo) string {
	if len(subAgents) == 0 {
		return "None"
	}

	var names []string
	for _, agent := range subAgents {
		names = append(names, agent.Name)
	}
	return strings.Join(names, ", ")
}
