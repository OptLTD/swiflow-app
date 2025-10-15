//go:build windows
// +build windows

package ability

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
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
// ensureLocalBinInPath adds user's local bin directory to PATH if it exists and not already present
func (m *DevCommonAbility) ensureLocalBinInPath(command *exec.Cmd) {
	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}

	// Check if ~/.local/bin directory exists
	localBinPath := filepath.Join(homeDir, ".local", "bin")
	if _, err := os.Stat(localBinPath); err != nil {
		return
	}

	exists, idx := false, -1
	for i, env := range os.Environ() {
		hasPath := strings.HasPrefix(env, "Path=")
		hasPATH := strings.HasPrefix(env, "PATH=")
		if !hasPath && !hasPATH {
			continue
		}

		idx = i
		path := env[5:] // Remove "PATH=" prefix
		// Check if local bin is already in PATH (Windows uses semicolon as separator)
		pathEntries := strings.Split(path, ";")
		for _, entry := range pathEntries {
			if strings.EqualFold(entry, localBinPath) { // Case-insensitive comparison for Windows
				exists = true
				break
			}
		}
		break
	}

	// Add local bin to PATH if not already present
	if !exists && idx >= 0 {
		if command.Env == nil {
			command.Env = os.Environ()
		}
		command.Env[idx] += ";" + localBinPath
	}
}

func (m *DevCommonAbility) cmdWithSandbox(ctx context.Context, cmd string, args ...string) *exec.Cmd {
	profile := config.GetStr("SANDBOX_PROFILE", "")
	command := exec.CommandContext(ctx, cmd, args...)

	if config.GetStr("DEBUG_MODE", "no") == "yes" {
		fullCmd := strings.TrimSpace(cmd + " " + strings.Join(args, " "))
		log.Printf("[DEBUG] Creating command: %s", fullCmd)
	}

	if profile != "" && hasWindowsSandbox() {
		fullCmd := strings.TrimSpace(cmd + " " + strings.Join(args, " "))
		if wsbFile, err := m.genSandboxProfile(m.home, fullCmd); err == nil {
			command = exec.CommandContext(ctx, "cmd", "/C", "start", "WindowsSandbox.exe", wsbFile)
			if config.GetStr("DEBUG_MODE", "no") == "yes" {
				log.Printf("[DEBUG] Using Windows Sandbox: cmd /C start WindowsSandbox.exe %s", wsbFile)
			}
		}
	}
	command.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true, CreationFlags: 0x08000000,
	}
	command.Dir = m.home
	m.ensureLocalBinInPath(command)
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

func (m *DevCommonAbility) start(command string, logFile ...string) error {
	log.Printf("[CMD] Starting command: %s", command)
	if len(logFile) > 0 && logFile[0] != "" {
		log.Printf("[CMD] Log file specified: %s", logFile[0])
	}

	ctx := context.Background()
	cmd := m.cmdWithSandbox(ctx, "cmd", "/C", "start /b "+command)

	var stdout, stderr bytes.Buffer
	var logFileHandle *os.File

	// Check if log file is specified
	if len(logFile) > 0 && logFile[0] != "" {
		// Create or open log file in home directory
		logPath := filepath.Join(m.home, logFile[0])
		log.Printf("[CMD] Creating log file at: %s", logPath)
		var err error
		logFileHandle, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Printf("[CMD] Failed to open log file %s: %v", logPath, err)
			return fmt.Errorf("failed to open log file %s: %v", logPath, err)
		}
		log.Printf("[CMD] Log file opened successfully")

		// Redirect stdout and stderr to both buffer and log file
		cmd.Stdout = io.MultiWriter(&stdout, logFileHandle)
		cmd.Stderr = io.MultiWriter(&stderr, logFileHandle)
	} else {
		// Use buffer only when no log file specified
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	}

	// Set working directory
	cmd.Dir = m.home
	log.Printf("[CMD] Starting command in path: %s", m.home)
	if err := cmd.Start(); err != nil {
		log.Printf("[CMD] Failed to start command: %v", err)
		if logFileHandle != nil {
			logFileHandle.Close()
		}
		if strings.TrimSpace(stdout.String()) != "" {
			m.logs = append(m.logs, stdout.String())
		}
		if strings.TrimSpace(stderr.String()) != "" {
			m.logs = append(m.logs, stderr.String())
		}
		return err
	}
	m.pid = int32(cmd.Process.Pid)

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		log.Printf("[CMD] Command completed within 300ms")
		if logFileHandle != nil {
			logFileHandle.Close()
		}
		if err != nil {
			log.Printf("[CMD] Command failed with error: %v", err)
			if strings.TrimSpace(stdout.String()) != "" {
				m.logs = append(m.logs, stdout.String())
			}
			if strings.TrimSpace(stderr.String()) != "" {
				m.logs = append(m.logs, stderr.String())
			}
			return err
		}
	case <-time.After(300 * time.Millisecond):
		// Command still running after 300ms, do not return error
		log.Printf("[CMD] Command still running after 300ms")
	}

	go func() {
		err := <-done
		// Close log file when command actually completes
		if logFileHandle != nil {
			logFileHandle.Close()
		}
		log.Printf("[CMD]: %v, %s", err, stdout.String())
	}()

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
