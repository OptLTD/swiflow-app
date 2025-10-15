package ability

import (
	"fmt"
	"log"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/go-cmd/cmd"
)

var store = struct {
	sync.Mutex
	pool map[string]*cmd.Cmd
}{pool: make(map[string]*cmd.Cmd)}

type DevAsyncCmdAbility struct {
	Home string
	Name string
	logs []string
}

func (m *DevAsyncCmdAbility) Logs() string {
	return strings.Join(m.logs, "\n")
}

func (m *DevAsyncCmdAbility) Start(command string) error {
	m.logs = append(m.logs, command)
	var newCmd *cmd.Cmd
	switch runtime.GOOS {
	case "windows":
		newCmd = cmd.NewCmd("cmd", "/C", command)
	default:
		newCmd = cmd.NewCmd("sh", "-c", command)
	}
	newCmd.Dir = m.Home

	newCmd.Start()

	store.Lock()
	store.pool[m.Name] = newCmd
	store.Unlock()

	// Wait for command output with timeout
	var hasOutput = false
	var maxWait = 5 // 1s * 5 = 5s
	for i := 0; i < maxWait; i++ {
		time.Sleep(1000 * time.Millisecond)

		// Check stdout
		stdoutLines := newCmd.Status().Stdout
		if len(stdoutLines) > 0 {
			output := strings.Join(stdoutLines, "\n")
			if output != "" {
				m.logs = append(m.logs, output)
				hasOutput = true
			}
		}

		// Check stderr - some commands output to stderr even when successful
		stderrLines := newCmd.Status().Stderr
		if len(stderrLines) > 0 {
			errOutput := strings.Join(stderrLines, "\n")
			if errOutput != "" {
				m.logs = append(m.logs, errOutput)
				hasOutput = true
			}
		}

		// If we have any output, break the waiting loop
		if hasOutput {
			break
		}
	}

	// Check if command is still running or has finished
	status := newCmd.Status()

	// If command has finished with an exit code, check the exit code
	if status.Complete {
		if status.Exit != 0 {
			// Command completed with non-zero exit code - this is an actual error
			errOutput := strings.Join(status.Stderr, "\n")
			if errOutput != "" {
				return fmt.Errorf("command failed with exit code %d: %s", status.Exit, errOutput)
			}
			return fmt.Errorf("command failed with exit code %d", status.Exit)
		}
		// Command completed successfully (exit code 0)
		return nil
	}

	// Command is still running - check if we got any output during the wait period
	if !hasOutput {
		return fmt.Errorf("command start timeout - no output received within %d seconds", maxWait)
	}

	return nil
}

func (m *DevAsyncCmdAbility) Query() (string, error) {
	store.Lock()
	cmd, ok := store.pool[m.Name]
	store.Unlock()
	if !ok || cmd == nil {
		return "", fmt.Errorf("session not found")
	}
	// 直接返回 Status().Stdout 内容
	return strings.Join(cmd.Status().Stdout, "\n"), nil
}

func (m *DevAsyncCmdAbility) Abort() error {
	if runtime.GOOS == "windows" {
		log.Println("abort not support")
		return fmt.Errorf("not support")
	}
	store.Lock()
	cmd, ok := store.pool[m.Name]
	store.Unlock()
	if !ok || cmd == nil {
		return fmt.Errorf("session not found")
	}
	if err := cmd.Stop(); err != nil {
		return err
	}
	delete(store.pool, m.Name)
	return nil
}

func (m *DevAsyncCmdAbility) Clear() error {
	if runtime.GOOS == "windows" {
		log.Println("clear not support")
		return fmt.Errorf("not support")
	}
	store.Lock()
	defer store.Unlock()
	for name, c := range store.pool {
		if c != nil {
			_ = c.Stop()
		}
		delete(store.pool, name)
	}
	return nil
}
