package observe

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

const (
	// RelLogFile is the log file name under the data directory.
	RelLogFile = "swiflow.log"
)

var (
	fileMu     sync.Mutex
	fileSink   *os.File
	logFileAbs string
)

// fileConsoleWriter writes to the log file first (required), then best-effort to
// stdout. On Windows GUI builds (-H windowsgui) stdout often errors; io.MultiWriter
// would abort and skip the file if stdout were listed first (or at all on failure).
type fileConsoleWriter struct {
	file *os.File
}

func (w *fileConsoleWriter) Write(p []byte) (int, error) {
	n, err := w.file.Write(p)
	if err != nil {
		return n, err
	}
	// Flush so Explore / Settings can see lines without waiting for process exit.
	_ = w.file.Sync()
	_, _ = os.Stdout.Write(p)
	return n, nil
}

// SetupFileLog appends slog output to <dataDir>/swiflow.log while still attempting
// to write to stdout. Safe to call once at process start.
// dataDir is typically the same directory as swiflow.db (e.g. %APPDATA%\Swiflow\data).
func SetupFileLog(dataDir string, level slog.Level) (string, error) {
	if dataDir == "" {
		return "", fmt.Errorf("data dir required")
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dataDir, RelLogFile)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}

	fileMu.Lock()
	if fileSink != nil {
		_ = fileSink.Close()
	}
	fileSink = f
	logFileAbs = path
	fileMu.Unlock()

	slog.SetDefault(slog.New(slog.NewTextHandler(&fileConsoleWriter{file: f}, &slog.HandlerOptions{Level: level})))
	slog.Info("file logging enabled", "path", path)
	return path, nil
}

// LogFileRelPath returns the log file basename.
func LogFileRelPath() string {
	return RelLogFile
}

// LogFileAbsPath returns the absolute path set by SetupFileLog, or empty if unset.
func LogFileAbsPath() string {
	fileMu.Lock()
	defer fileMu.Unlock()
	return logFileAbs
}
