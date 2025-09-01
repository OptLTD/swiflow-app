package ability

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

type DevCommandAbility struct {
	Home string

	// base ability
	base DevCommonAbility
}

func (m *DevCommandAbility) Cmd(name string, args []string) *exec.Cmd {
	return m.base.cmd(name, args...)
}

func (m *DevCommandAbility) Logs() string {
	return strings.TrimSpace(strings.Join(m.base.logs, "\n"))
}

func (m *DevCommandAbility) Run(cmd string, timeout time.Duration, args ...string) (string, error) {
	m.base.home, m.base.logs = m.Home, []string{}
	if data, err := m.base.run(cmd, timeout, args...); err != nil {
		log.Printf("exec cmd fail: %v %s", err, cmd)
		return string(data), fmt.Errorf("exec cmd: %v", err)
	} else {
		return string(data), nil
	}
}

func (m *DevCommandAbility) Exec(cmd string, timeout time.Duration) (string, error) {
	m.base.home, m.base.logs = m.Home, []string{}
	if data, err := m.base.exec(cmd, timeout); err != nil {
		log.Printf("exec cmd fail: %v %s", err, cmd)
		return string(data), fmt.Errorf("exec cmd: %v", err)
	} else {
		return string(data), nil
	}
}

func (m *DevCommandAbility) Start(cmd string) (int32, error) {
	m.base.home, m.base.logs = m.Home, []string{}
	if err := m.base.start(cmd); err != nil {
		log.Printf("start cmd fail: %v", err)
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
