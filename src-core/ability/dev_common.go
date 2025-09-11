//go:build !windows
// +build !windows

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
	"runtime"
	"strings"
	"swiflow/config"
	"swiflow/support"
	"sync"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

var (
	sandboxExecPath string
	sandboxExecOnce sync.Once
	firejailPath    string
	firejailOnce    sync.Once
)

// 检查sandbox-exec是否存在，只查一次
func hasSandboxExec() bool {
	sandboxExecOnce.Do(func() {
		if path, err := exec.LookPath("sandbox-exec"); err == nil {
			sandboxExecPath = path
		}
	})
	return sandboxExecPath != ""
}

// 检查firejail是否存在，只查一次
func hasFirejail() bool {
	firejailOnce.Do(func() {
		if path, err := exec.LookPath("firejail"); err == nil {
			firejailPath = path
		}
	})
	return firejailPath != ""
}

type DevCommonAbility struct {
	home string

	pid int32

	logs []string
}

// 动态生成sandbox profile，只允许读写home目录
//
// profile参数为用户自定义的profile内容（可为空或包含如(version 1)、(allow default)、网络/进程等限制）
// home为允许读写的根目录。
//
// 生成的profile文件内容为：
//
//	<profile内容>
//	(deny file-read* file-write* file-ioctl*)
//	(allow file-read* file-write* file-ioctl* (subpath "<home>"))
//
// 示例：
//
//	用户配置的SANDBOX_PROFILE内容：
//	  (version 1)
//	  (allow default)
//	  (deny network*)
//	home为 /Users/xxx/Works/swiflow-app/tmp
//	则最终profile内容为：
//	  (version 1)
//	  (allow default)
//	  (deny network*)
//	  (deny file-read* file-write* file-ioctl*)
//	  (allow file-read* file-write* file-ioctl* (subpath "/Users/xxx/Works/swiflow-app/tmp"))
//
// 这样子进程只能读写home目录，且无法联网。
func (m *DevCommonAbility) genMacSandboxProfile(profile string, home string) (string, error) {
	content := fmt.Sprintf(`%s
(deny file-read* file-write* file-ioctl*)
(allow file-read* file-write* file-ioctl* (subpath "%s"))
`, profile, home)
	tmpfile := filepath.Join(os.TempDir(), fmt.Sprintf("sandbox-%d.sb", time.Now().UnixNano()))
	err := os.WriteFile(tmpfile, []byte(content), 0644)
	if err != nil {
		return "", err
	}
	return tmpfile, nil
}

// 生成firejail参数，限制只能访问home目录且无网络
func (m *DevCommonAbility) genFirejailArgs(home string) []string {
	return []string{"--private=" + home, "--net=none", "--quiet"}
}

// ensureLocalBinInPath adds ~/.local/bin to PATH if it exists and is not already present
func (m *DevCommonAbility) ensureLocalBinInPath(command *exec.Cmd) {
	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}

	localBinPath := filepath.Join(homeDir, ".local", "bin")
	// Check if ~/.local/bin directory exists
	if _, err := os.Stat(localBinPath); err != nil {
		return
	}

	exists, idx := false, -1
	for i, env := range os.Environ() {
		if !strings.HasPrefix(env, "PATH=") {
			continue
		}

		idx = i
		path := env[5:] // Remove "PATH=" prefix
		// Check if ~/.local/bin is already in PATH
		pathEntries := strings.SplitSeq(path, ":")
		for entry := range pathEntries {
			if entry == localBinPath {
				exists = true
				break
			}
		}
		break
	}

	// Add ~/.local/bin to PATH if not already present
	if !exists && idx >= 0 {
		if command.Env == nil {
			command.Env = os.Environ()
		}
		currentPath := command.Env[idx][5:] // Remove "PATH=" prefix
		command.Env[idx] = "PATH=" + localBinPath + ":" + currentPath
	}
}

// 公共方法：根据配置决定是否用sandbox-exec包装
func (m *DevCommonAbility) cmdWithSandbox(ctx context.Context, cmd string, args ...string) *exec.Cmd {
	profile := config.GetStr("SANDBOX_PROFILE", "")
	switch runtime.GOOS {
	case "darwin":
		if profile != "" && hasSandboxExec() {
			if path, err := m.genMacSandboxProfile(profile, m.home); err == nil {
				allArgs := []string{"-f", path, "sh", "-c", cmd + " " + strings.Join(args, " ")}
				return exec.CommandContext(ctx, "sandbox-exec", allArgs...)
			}
		}
	case "linux":
		if profile != "" && hasFirejail() {
			fjArgs := m.genFirejailArgs(m.home)
			fjArgs = append(fjArgs, cmd)
			fjArgs = append(fjArgs, args...)
			return exec.CommandContext(ctx, "firejail", fjArgs...)
		}
	}
	return exec.CommandContext(ctx, cmd, args...)
}

func (m *DevCommonAbility) cmd(command string, args ...string) *exec.Cmd {
	cmd := m.cmdWithSandbox(context.Background(), command, args...)
	m.ensureLocalBinInPath(cmd)
	return cmd
}

func (m *DevCommonAbility) run(cmd string, timeout time.Duration, args ...string) ([]byte, error) {
	return m.exec(cmd+" "+strings.Join(args, " "), timeout)
}

func (m *DevCommonAbility) exec(command string, timeout time.Duration) ([]byte, error) {
	if cmdPath, err := config.GetMcpEnv(command); err != nil {
		return nil, fmt.Errorf("command preparation failed: %v", err)
	} else if cmdPath != "" {
		command = cmdPath
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := m.cmdWithSandbox(ctx, "sh", "-c", command)
	cmd.Dir = m.home
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	m.ensureLocalBinInPath(cmd)

	// 获取输出管道
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	// 启动命令
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	// 在 goroutine 中监控超时并强制杀进程组
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			if pgid, err := syscall.Getpgid(cmd.Process.Pid); err == nil {
				syscall.Kill(-pgid, syscall.SIGKILL)
			}
		case <-done:
		}
	}()

	// 读取输出
	outBytes, _ := io.ReadAll(stdout)
	errBytes, _ := io.ReadAll(stderr)

	err = cmd.Wait()
	close(done)

	output := append(outBytes, errBytes...)

	m.logs = append(m.logs, "$ "+command)
	m.logs = append(m.logs, support.Quote(string(output)))

	if ctx.Err() == context.DeadlineExceeded {
		return nil, ctx.Err()
	}

	return output, err
}

func (m *DevCommonAbility) start(command string, logFile ...string) error {
	log.Printf("[CMD] Starting command: %s", command)
	if len(logFile) > 0 && logFile[0] != "" {
		log.Printf("[CMD] Log file specified: %s", logFile[0])
	}

	ctx := context.Background()
	cmd := m.cmdWithSandbox(ctx, "sh", "-c", command)
	m.ensureLocalBinInPath(cmd)

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
