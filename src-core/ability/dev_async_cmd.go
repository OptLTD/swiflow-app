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
	if runtime.GOOS == "windows" {
		log.Println("start not support")
		return fmt.Errorf("not support")
	}

	m.logs = append(m.logs, command)
	newCmd := cmd.NewCmd("sh", "-c", command)
	newCmd.Dir = m.Home

	newCmd.Start()

	store.Lock()
	store.pool[m.Name] = newCmd
	store.Unlock()

	// 0.5s后获取Stdout，若为空则继续，每1s最多等5s
	var output, maxWait = "", 5 // 1s * 5 = 5s
	for i := 0; i < maxWait; i++ {
		time.Sleep(1000 * time.Millisecond)
		output = strings.Join(newCmd.Status().Stdout, "\n")
		if output != "" {
			m.logs = append(m.logs, output)
			break
		}
	}

	if out := newCmd.Status().Stdout; len(out) == 0 {
		if err := newCmd.Status().Stderr; len(err) > 0 {
			return fmt.Errorf("command start error: %s", err)
		} else {
			return fmt.Errorf("command start timeout")
		}
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
