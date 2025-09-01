//go:build windows
// +build windows

package ability

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"swiflow/config"
	"swiflow/support"
	"sync"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

var (
	wsbPath string
	wsbOnce sync.Once
)

// 检查 WindowsSandbox.exe 是否存在
func hasWindowsSandbox() bool {
	wsbOnce.Do(func() {
		if path, err := exec.LookPath("WindowsSandbox.exe"); err == nil {
			wsbPath = path
		}
	})
	return wsbPath != ""
}

type DevCommonAbility struct {
	home string

	pid int32

	logs []string
}

// 生成 .wsb 配置文件，映射 home 目录，禁用网络，自动运行命令
func (m *DevCommonAbility) genSandboxProfile(home, command string) (string, error) {
	wsbContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Configuration>
  <MappedFolders>
    <MappedFolder>
      <HostFolder>%s</HostFolder>
      <ReadOnly>false</ReadOnly>
    </MappedFolder>
  </MappedFolders>
  <LogonCommand>
    <Command>cmd.exe /C %s</Command>
  </LogonCommand>
  <Networking>Disable</Networking>
</Configuration>
`, home, command)
	tmpfile := filepath.Join(os.TempDir(), fmt.Sprintf("sandbox-%d.wsb", time.Now().UnixNano()))
	if err := os.WriteFile(tmpfile, []byte(wsbContent), 0644); err != nil {
		return "", err
	}
	return tmpfile, nil
}

// Windows下 cmdWithSandbox 支持 Windows Sandbox (WSB)
func (m *DevCommonAbility) cmdWithSandbox(ctx context.Context, cmd string, args ...string) *exec.Cmd {
	profile := config.GetStr("SANDBOX_PROFILE", "")
	if profile != "" && hasWindowsSandbox() {
		// 只支持单条命令，拼接参数
		fullCmd := cmd
		if len(args) > 0 {
			fullCmd = fullCmd + " " + strings.Join(args, " ")
		}
		wsbFile, err := m.genSandboxProfile(m.home, fullCmd)
		if err == nil {
			// 启动 WindowsSandbox.exe 并加载配置
			return exec.CommandContext(ctx, "cmd", "/C", "start", "", "WindowsSandbox.exe", wsbFile)
		}
		// 生成失败则降级为无沙箱
	}
	command := exec.CommandContext(ctx, cmd, args...)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	command.Dir = m.home
	return command
}

func (m *DevCommonAbility) cmd(cmd string, args ...string) *exec.Cmd {
	return m.cmdWithSandbox(context.Background(), cmd, args...)
}

func (m *DevCommonAbility) run(cmd string, timeout time.Duration, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	command := m.cmdWithSandbox(ctx, cmd, args...)

	stdoutPipe, err := command.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderrPipe, err := command.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := command.Start(); err != nil {
		return nil, err
	}

	stdoutBytes, _ := io.ReadAll(stdoutPipe)
	stderrBytes, _ := io.ReadAll(stderrPipe)

	err = command.Wait()

	output := append(stdoutBytes, stderrBytes...)

	argStr := " " + strings.Join(args, " ")
	m.logs = append(m.logs, "$ "+cmd+argStr)
	m.logs = append(m.logs, support.Quote(string(output)))

	return output, err
}

func (m *DevCommonAbility) exec(command string, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := m.cmdWithSandbox(ctx, "cmd", "/C", command)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	stdoutBytes, _ := io.ReadAll(stdoutPipe)
	stderrBytes, _ := io.ReadAll(stderrPipe)

	err = cmd.Wait()

	output := append(stdoutBytes, stderrBytes...)

	m.logs = append(m.logs, "$ "+command)
	m.logs = append(m.logs, support.Quote(string(output)))

	return output, err
}

func (m *DevCommonAbility) start(command string) error {
	ctx := context.Background()
	cmd := m.cmdWithSandbox(ctx, "cmd", "/C", "start /b "+command)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 设置工作目录
	cmd.Dir = m.home
	if err := cmd.Start(); err != nil {
		if strings.TrimSpace(stdout.String()) != "" {
			m.logs = append(m.logs, stdout.String())
		}
		if strings.TrimSpace(stderr.String()) != "" {
			m.logs = append(m.logs, stderr.String())
		}
		return err
	}
	m.pid = int32(cmd.Process.Pid)

	// 创建一个 channel 用于接收 cmd.Wait() 的结果
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		// 等待 300ms，如果命令在 300ms 内完成且失败，则返回错误
		if err != nil {
			if strings.TrimSpace(stdout.String()) != "" {
				m.logs = append(m.logs, stdout.String())
			}
			if strings.TrimSpace(stderr.String()) != "" {
				m.logs = append(m.logs, stderr.String())
			}
			return err
		}
	case <-time.After(300 * time.Millisecond):
		// 300ms 后命令仍在运行，不返回错误
	}

	// 继续在后台等待命令完成（不影响主流程）
	// go func() {
	// 	err := <-done
	// 	log.Printf("stdout: %v, %s", err, stdout.String())
	// }()

	return nil
}

// 获取进程状态
func (m *DevCommonAbility) Status() string {
	proc, err := process.NewProcess(m.pid)
	if err != nil {
		return ""
	}

	status, err := proc.Status()
	if err != nil || len(status) == 0 {
		return ""
	}
	if status[0] == "zombie" {
		m.Terminate()
		return ""
	}
	return strings.Join(status, ",")
}

// 停止进程
func (m *DevCommonAbility) Terminate() error {
	if m.pid == 0 {
		return nil
	}

	proc, err := process.NewProcess(m.pid)
	if err != nil {
		// 进程可能已经不存在
		if m.IsNotFoundError(err) {
			m.pid = 0
			return nil
		}
		return err
	}

	// 先尝试终止子进程
	m.KillChildren(proc)

	// 尝试优雅终止主进程
	if err := proc.Terminate(); err != nil {
		if m.IsNotFoundError(err) {
			m.pid = 0
			return nil
		}

		// 如果优雅终止失败，尝试强制终止
		if err := proc.Kill(); err != nil {
			if m.IsNotFoundError(err) {
				m.pid = 0
				return nil
			}
			return err
		}
	}

	m.pid = 0
	return nil
}

// 终止进程的所有子进程
func (m *DevCommonAbility) KillChildren(proc *process.Process) {
	children, err := proc.Children()
	if err != nil {
		return // 忽略错误，可能没有子进程
	}

	// 先尝试正常终止所有子进程
	for _, child := range children {
		_ = child.Terminate()
	}

	// 给子进程一点时间来完成终止
	time.Sleep(100 * time.Millisecond)

	// 如果子进程仍然存在，强制终止
	for _, child := range children {
		if exists, _ := process.PidExists(child.Pid); exists {
			_ = child.Kill()
		}
	}
}

// 判断是否为"进程不存在"错误
func (m *DevCommonAbility) IsNotFoundError(err error) bool {
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "process not found") ||
		strings.Contains(errMsg, "no such process") ||
		strings.Contains(errMsg, "no process found")
}
