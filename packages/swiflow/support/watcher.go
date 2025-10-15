package support

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

type FileWatcher struct {
	watcher *fsnotify.Watcher
	path    string
	taskID  string
	active  bool
}

func WatchOutput(name string, stdout io.Reader) {
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				Emit("stream", name, string(buf[:n]))
			}
			if err != nil {
				if err != io.EOF {
					log.Printf("[%s] %s", name, err.Error())
				}
				break
			}
		}
	}()
}

// 创建新的文件监控器
func NewFileWatcher(path string, taskID string) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &FileWatcher{
		watcher: watcher,
		path:    path,
		taskID:  taskID,
		active:  false,
	}, nil
}

// 开始监控
func (fw *FileWatcher) Start() error {
	if fw.active {
		return nil
	}

	// 确保目录存在
	if _, err := os.Stat(fw.path); os.IsNotExist(err) {
		if err := os.MkdirAll(fw.path, 0755); err != nil {
			return err
		}
	}

	// 添加监控路径
	if err := fw.watcher.Add(fw.path); err != nil {
		return err
	}

	fw.active = true
	log.Printf("[FILE] start watch: %s, task: %s", fw.path, fw.taskID)

	// 启动监控协程
	go fw.watchLoop()

	return nil
}

// 停止监控
func (fw *FileWatcher) Stop() {
	if !fw.active {
		return
	}

	fw.active = false
	fw.watcher.Close()
	log.Printf("[FILE] stop watch: %s, task: %s", fw.path, fw.taskID)
}

// 监控循环
func (fw *FileWatcher) watchLoop() {
	for fw.active {
		select {
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}
			fw.handleEvent(event)
		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("[FILE] watch error: %v", err)
		}
	}
}

// 处理文件事件
func (fw *FileWatcher) handleEvent(event fsnotify.Event) {
	// 过滤掉临时文件和隐藏文件
	if fw.shouldIgnore(event.Name) {
		return
	}

	// 获取相对路径
	relPath, err := filepath.Rel(fw.path, event.Name)
	if err != nil {
		relPath = event.Name
	}

	// 确定操作类型
	operation := fw.getOperation(event.Op)

	// 确定文件类型
	fileType := "file"
	if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
		fileType = "directory"
	}

	// 构建文件变动详情
	detail := map[string]any{
		"type": fileType, "path": relPath,
		"operation": operation,
		"timestamp": time.Now().Unix(),
	}

	// 发送modify消息
	Emit("change", fw.taskID, detail)

	log.Printf("[FILE] file change: %s %s (%s)", operation, relPath, fw.taskID)
}

// 判断是否应该忽略该文件
func (fw *FileWatcher) shouldIgnore(path string) bool {
	// 忽略临时文件
	if strings.Contains(path, ".tmp") || strings.Contains(path, ".temp") {
		return true
	}

	// 忽略隐藏文件
	base := filepath.Base(path)
	if strings.HasPrefix(base, ".") {
		return true
	}

	// 忽略系统文件
	if strings.Contains(path, ".DS_Store") || strings.Contains(path, "Thumbs.db") {
		return true
	}

	// 忽略日志文件
	if strings.HasSuffix(path, ".log") {
		return true
	}

	return false
}

// 获取操作类型描述
func (fw *FileWatcher) getOperation(op fsnotify.Op) string {
	switch {
	case op&fsnotify.Write == fsnotify.Write:
		return "modify"
	case op&fsnotify.Create == fsnotify.Create:
		return "create"
	case op&fsnotify.Remove == fsnotify.Remove:
		return "delete"
	case op&fsnotify.Rename == fsnotify.Rename:
		return "rename"
	case op&fsnotify.Chmod == fsnotify.Chmod:
		return "chmod"
	default:
		return "unknown"
	}
}
