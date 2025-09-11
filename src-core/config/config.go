package config

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"swiflow/support"

	"github.com/joho/godotenv"
)

func LoadEnv() error {
	err := godotenv.Load()
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func Get(key string) string {
	return os.Getenv(key)
}

func Set(key string, val string) error {
	return os.Setenv(key, val)
}

func GetStr(key string, dft string) string {
	value := os.Getenv(key)
	return support.Or(value, dft)
}

func GetInt(key string, dft int) int {
	value := os.Getenv(key)
	if value == "" {
		return dft
	}

	num, _ := strconv.Atoi(value)
	return support.Or(num, dft)
}

func GetDataHome() (home string) {
	base, _ := os.UserConfigDir()
	home = filepath.Join(base, "App.Swiflow")
	if _, err := os.Stat(home); os.IsNotExist(err) {
		os.MkdirAll(home, fs.ModeDir|0755)
	}
	return home
}

// 获取current bot home
func CurrentHome() (home string) {
	return os.Getenv("CURRENT_HOME")
}

func GetWorkHome() (home string) {
	home = os.Getenv("SWIFLOW_HOME")
	if home != "" && home != "/" {
		return home
	}

	if base, err := os.UserHomeDir(); err == nil {
		home = filepath.Join(base, ".swiflow")
	} else {
		log.Fatal("get home err", err)
		return
	}
	if _, err := os.Stat(home); os.IsNotExist(err) {
		os.MkdirAll(home, fs.ModeDir|0755)
	}
	return home
}

func GetWorkPath(args ...string) string {
	path := []string{GetWorkHome()}
	path = append(path, args...)
	return filepath.Join(path...)
}

func GetDataPath(args ...string) string {
	path := []string{GetDataHome()}
	path = append(path, args...)
	return filepath.Join(path...)
}

func GetLogFile(filename string) (*os.File, error) {
	path := filepath.Join(GetDataHome(), filename)
	mode := os.O_RDWR | os.O_CREATE | os.O_APPEND
	return os.OpenFile(path, mode, 0644)
}

func GetAuthGate() string {
	return GetStr("AUTH_GATE", "https://auth.swiflow.cc")
}

func NotifyLock() (*os.File, error) {
	return GetLogFile("notify.lock")
}

func ServerLog() (*os.File, error) {
	return GetLogFile("serve.log")
}

func SetVersion(ver string) error {
	return os.Setenv("SWIFLOW_VERSION", ver)
}

func GetVersion() string {
	return os.Getenv("SWIFLOW_VERSION")
}

func InContainer() bool {
	return os.Getenv("IN_CONTAINER") != ""
}
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

func EpigraphInfo() map[string]any {
	info := os.Getenv("EPIGRAPH")
	if info == "" {
		return nil
	}
	arr := strings.Split(info, "|")
	result := map[string]any{}
	for i, item := range arr {
		switch i {
		case 0:
			result["name"] = item
		case 1:
			result["text"] = item
		}
	}
	return result
}

func NeedUpgrade() bool {
	version := GetDataPath("version")
	val, err := os.ReadFile(version)
	if os.IsNotExist(err) {
		os.WriteFile(version, []byte(GetVersion()), 0644)
		log.Println("[MIGRATION] need initial~")
		return true
	}

	if string(val) != GetVersion() {
		os.WriteFile(version, []byte(GetVersion()), 0644)
		log.Println("[MIGRATION] need upgrade~", string(val))
		return true
	}
	return false
}

func GetShellName() (string, string) {
	switch runtime.GOOS {
	case "windows":
		shell := os.Getenv("ComSpec")
		return "windows", shell
	default:
		shell := os.Getenv("SHELL")
		return runtime.GOOS, shell
	}
}

// GetMcpEnv checks if npx or uvx command is available and returns the appropriate path
func GetMcpEnv(cmd string) (string, error) {
	depends := []string{
		"python", "python3", "uvx", "uv",
		"node", "npx", "npm", "yarn", "pnpm",
	}
	if !slices.Contains(depends, cmd) {
		return cmd, nil
	}

	if _, err := exec.LookPath(cmd); err == nil {
		checkCmd := exec.Command(cmd, "--version")
		if checkCmd.Run() == nil {
			return cmd, nil
		}
	}

	var localBinPath string // Try ~/.local/bin
	if homeDir, err := os.UserHomeDir(); err != nil {
		return "", fmt.Errorf("failed to get user home directory: %v", err)
	} else if homeDir != "" {
		if IsWindows() {
			cmd += ".exe"
		}
		localBinPath = filepath.Join(homeDir, ".local", "bin", cmd)
	}

	if _, err := os.Stat(localBinPath); err != nil {
		log.Printf(" %s not found in PATH or ~/.local/bin: %v", localBinPath, err)
		return "", fmt.Errorf("%s not found in PATH or ~/.local/bin: %w", cmd, err)
	}

	// Check if the local binary works
	if checkCmd := exec.Command(localBinPath, "--version"); checkCmd.Run() != nil {
		return "", fmt.Errorf("%s found in ~/.local/bin but version check failed", cmd)
	}

	return localBinPath, nil
}
