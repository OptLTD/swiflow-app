package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/duke-git/lancet/v2/fileutil"
)

// DiscoverWorkers discovers and loads agent definitions from the .agent directory
// This function scans for .md files in the .agent directory and creates BotEntity instances
func DiscoverWorkers(scanDir string) ([]*Worker, error) {
	var workers []*Worker

	var err error
	var files []os.DirEntry
	if files, err = os.ReadDir(scanDir); err != nil {
		return nil, fmt.Errorf("failed to read agent directory: %v", err)
	}

	var leaderId string
	for _, file := range files {
		// Skip task.md as it's not an agent definition
		if file.IsDir() || file.Name() == "task.md" {
			continue
		}

		// Only process .md files
		if !strings.HasSuffix(file.Name(), ".md") {
			continue
		}

		agentPath := filepath.Join(scanDir, file.Name())
		prompt, err := fileutil.ReadFileToString(agentPath)
		if err != nil {
			log.Printf("[AGENT] Warning: failed to read agent file %s: %v", file.Name(), err)
			continue
		}

		// Extract agent name from filename (remove .md extension)
		name := strings.TrimSuffix(file.Name(), ".md")
		entity := &Worker{
			Name: name, UUID: name,
			UsePrompt: prompt,
		}
		if strings.HasPrefix(name, "leader-") {
			entity.Type = AGENT_LEADER
		} else {
			entity.Type = AGENT_WORKER
		}

		jsonPath := strings.Replace(agentPath, ".md", ".json", 1)
		if data, err := os.ReadFile(jsonPath); err == nil {
			var scan = &Worker{}
			json.Unmarshal(data, scan)
			if scan.Name != "" {
				entity.Name = scan.Name
			}
			if scan.UUID != "" {
				entity.UUID = scan.UUID
			}
			if scan.Desc != "" {
				entity.Desc = scan.Desc
			}
			if scan.Emoji != "" {
				entity.Emoji = scan.Emoji
			}
			if len(scan.Tools) > 0 {
				entity.Tools = scan.Tools
			}
			if len(scan.McpServers) > 0 {
				entity.McpServers = scan.McpServers
				for uuid := range scan.McpServers {
					entity.Tools = append(entity.Tools, uuid+":*")
				}
			}
		}
		if entity.Type == AGENT_LEADER {
			leaderId = entity.UUID
		}
		workers = append(workers, entity)
	}

	for _, worker := range workers {
		if worker.Type == AGENT_WORKER {
			worker.Leader = leaderId
		}
	}
	return workers, nil
}
