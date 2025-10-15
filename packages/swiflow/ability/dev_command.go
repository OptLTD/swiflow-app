package ability

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"swiflow/config"
	"time"
)

type DevCommandAbility struct {
	Home string

	// base ability
	base DevCommonAbility
}

func (m *DevCommandAbility) Cmd(name string, args []string) *exec.Cmd {
	// @todo: check uvx\npx full path
	return m.base.cmd(name, args...)
}

func (m *DevCommandAbility) Logs() string {
	return strings.TrimSpace(strings.Join(m.base.logs, "\n"))
}

func (m *DevCommandAbility) Run(cmd string, timeout time.Duration, args ...string) (string, error) {
	if cmdPath, err := config.GetMcpEnv(cmd); err != nil {
		return "", fmt.Errorf("xec cmd: %v", err)
	} else if cmdPath != "" {
		cmd = cmdPath
	}

	m.base.home, m.base.logs = m.Home, []string{}
	if data, err := m.base.run(cmd, timeout, args...); err != nil {
		log.Printf("[CMD] exec cmd fail: %v %s", err, cmd)
		return string(data), fmt.Errorf("exec cmd: %v", err)
	} else {
		return string(data), nil
	}
}

func (m *DevCommandAbility) Exec(cmd string, timeout time.Duration) (string, error) {
	// @todo: check uvx\npx full path
	m.base.home, m.base.logs = m.Home, []string{}
	if data, err := m.base.exec(cmd, timeout); err != nil {
		log.Printf("[CMD] exec cmd fail: %v %s", err, cmd)
		return string(data), fmt.Errorf("exec cmd: %v", err)
	} else {
		return string(data), nil
	}
}

func (m *DevCommandAbility) Start(cmd string, logFile ...string) (int32, error) {
	// @todo: check uvx\npx full path
	m.base.home, m.base.logs = m.Home, []string{}
	if err := m.base.start(cmd, logFile...); err != nil {
		log.Printf("[CMD] start cmd fail: %v", err)
		return 0, fmt.Errorf("start cmd: %v", err)
	}
	return m.base.pid, nil
}

func (m *DevCommandAbility) Status(pid int32) string {
	m.base.pid, m.base.home = pid, m.Home
	return m.base.Status()
}

func (m *DevCommandAbility) Stop(pid int32) error {
	m.base.pid, m.base.home = pid, m.Home
	return m.base.Terminate()
}
