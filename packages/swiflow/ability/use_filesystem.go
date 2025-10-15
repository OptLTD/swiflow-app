package ability

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// 允许的文件扩展名列表
var allowedExtensions = map[string]bool{
	".txt":        true,
	".md":         true,
	".py":         true,
	".js":         true,
	".ts":         true,
	".html":       true,
	".css":        true,
	".json":       true,
	".yaml":       true,
	".yml":        true,
	".xml":        true,
	".csv":        true,
	".sql":        true,
	".sh":         true,
	".bat":        true,
	".ps1":        true,
	".dockerfile": true,
	".gitignore":  true,
	".env":        true,
	".ini":        true,
	".cfg":        true,
	".conf":       true,
	".log":        true,
	".rst":        true,
	".toml":       true,
	".lock":       true,
}

// 隐藏文件列表
var hiddenFiles = map[string]bool{
	"._*":              true,
	".*":               true,
	".git":             true,
	"Icon?":            true,
	"Thumbs.db":        true,
	".gitignore":       true,
	".DS_Store":        true,
	".DS_Store?":       true,
	".Trashes":         true,
	".fseventsd":       true,
	".AppleDouble":     true,
	".LSOverride":      true,
	".Spotlight-V100":  true,
	".TemporaryItems":  true,
	".VolumeIcon.icns": true,

	".com.apple.timemachine.donotpresent": true,
}

// 验证文件类型是否被允许
func (m *FileSystemAbility) isAllowedFileType() error {
	ext := strings.ToLower(filepath.Ext(m.Path))
	if ext == "" {
		return fmt.Errorf("文件必须具有扩展名")
	}
	if !allowedExtensions[ext] {
		return fmt.Errorf("不支持的文件类型: %s。允许的类型包括: %s", ext, strings.Join(getAllowedExtensionsList(), ", "))
	}
	return nil
}

// 获取允许的文件扩展名列表（用于错误信息）
func getAllowedExtensionsList() []string {
	var extensions []string
	for ext := range allowedExtensions {
		extensions = append(extensions, ext)
	}
	sort.Strings(extensions)
	return extensions
}

// 检查是否为隐藏文件
func isHiddenFile(name string) bool {
	// 检查是否在隐藏文件列表中
	if hiddenFiles[name] {
		return true
	}

	// 检查是否以点开头
	if strings.HasPrefix(name, ".") {
		return true
	}

	// 检查是否匹配通配符模式
	if strings.HasPrefix(name, "._") {
		return true
	}

	// msoffice template file
	if strings.HasPrefix(name, "~$") {
		return true
	}
	return false
}

type FileSystemAbility struct {
	Base string
	Path string
}

func (m *FileSystemAbility) fullpath(path string) string {
	if strings.HasPrefix(path, m.Base) {
		return path
	}
	return filepath.Join(m.Base, path)
}

func (m *FileSystemAbility) initBaseDir(path string) error {
	if filepath.IsAbs(path) && !strings.HasPrefix(path, m.Base) {
		return fmt.Errorf("Absolute path must be under the Work Path (%s)", m.Base)
	}
	base := filepath.Dir(m.fullpath(path))
	if _, err := os.Stat(base); os.IsNotExist(err) {
		if err := os.MkdirAll(base, 0755); err != nil {
			return err
		}
	}
	return nil
}

func (m *FileSystemAbility) IsDir() bool {
	if m.Path == "" || m.Path == "." {
		return true
	}
	if strings.HasSuffix(m.Path, "/") {
		return true
	}
	if strings.HasSuffix(m.Path, "\\") {
		return true
	}
	path := m.fullpath(m.Path)
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

func (m *FileSystemAbility) Write(data string) error {
	if err := m.isAllowedFileType(); err != nil {
		return err
	}
	if err := m.initBaseDir(m.fullpath(m.Path)); err != nil {
		return err
	}
	return os.WriteFile(m.fullpath(m.Path), []byte(data), 0644)
}

func (m *FileSystemAbility) Read() ([]byte, error) {
	if filepath.IsAbs(m.Path) && !strings.HasPrefix(m.Path, m.Base) {
		return nil, fmt.Errorf("Absolute path must be under the Work Path (%s)", m.Base)
	}
	if err := m.isAllowedFileType(); err != nil {
		return nil, err
	}
	return os.ReadFile(m.fullpath(m.Path))
}

func (m *FileSystemAbility) Copy(file io.Reader) error {
	if err := m.initBaseDir(m.fullpath(m.Path)); err != nil {
		return err
	}
	dst, err := os.Create(m.fullpath(m.Path))
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return err
	}
	return nil
}

func (m *FileSystemAbility) List() ([]map[string]any, error) {
	files, err := os.ReadDir(m.fullpath(m.Path))
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %v", err)
	}
	var result []map[string]any
	for _, file := range files {
		// 过滤掉隐藏文件
		if isHiddenFile(file.Name()) {
			continue
		}

		if info, err := file.Info(); err != nil {
			continue
		} else {
			result = append(result, map[string]any{
				"mode": info.Mode().String(), "size": info.Size(), "name": file.Name(),
				// "user":  fmt.Sprintf("%d", info.Sys().(*syscall.Stat_t).Uid),
				// "group": fmt.Sprintf("%d", info.Sys().(*syscall.Stat_t).Gid),
				"time": info.ModTime().Local().Format(time.RFC822),
			})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i]["name"].(string) < result[j]["name"].(string)
	})
	return result, nil
}

func (m *FileSystemAbility) Rename(newPath string) error {
	if err := m.initBaseDir(m.Path); err != nil {
		return err
	}
	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("目标文件已存在")
	}
	if err := m.isAllowedFileType(); err != nil {
		return err
	}
	if err := os.Rename(m.Path, newPath); err != nil {
		return fmt.Errorf("重命名失败: %v", err)
	}
	m.Path = newPath
	return nil
}

func (m *FileSystemAbility) Replace(diff string) error {
	// Write modified content with original permissions
	if err := m.initBaseDir(m.Path); err != nil {
		return err
	}
	if err := m.isAllowedFileType(); err != nil {
		return err
	}
	var mode = fs.FileMode(0644)
	if val, err := os.Stat(m.fullpath(m.Path)); err != nil {
		return fmt.Errorf("reading file error: %v", err)
	} else {
		mode = val.Mode()
	}

	var data []byte
	if val, err := os.ReadFile(m.fullpath(m.Path)); err != nil {
		return fmt.Errorf("reading file error: %v", err)
	} else {
		data = val
	}

	// Parse and apply all SEARCH/REPLACE blocks
	content := string(data)
	datareg := regexp.MustCompile(
		`(?s)<<<<<<< SEARCH\n(.*?)\n=======\n(.*?)\n>>>>>>> REPLACE`,
	)

	// 查找所有的 SEARCH/REPLACE 块
	matches := datareg.FindAllStringSubmatch(diff, -1)

	// 遍历所有的匹配项，并在 text 中进行替换
	for _, match := range matches {
		diff = strings.Replace(diff, match[0], "", 1)
		content = strings.Replace(content, match[1], match[2], 1)
	}
	if diff = strings.TrimSpace(diff); diff != "" {
		if !strings.Contains(diff, "<<<<<<< SEARCH") {
			return fmt.Errorf("invalid diff format - missing SEARCH")
		}
		if !strings.Contains(diff, ">>>>>>> REPLACE") {
			return fmt.Errorf("invalid diff format - missing REPLACE")
		}
		if !strings.Contains(diff, "=======") {
			return fmt.Errorf("invalid diff format - missing separator")
		}
		return fmt.Errorf("invalid diff format, parse failure\n%s", diff)
	}
	if err := os.WriteFile(m.fullpath(m.Path), []byte(content), mode); err != nil {
		return fmt.Errorf("writing file error: %v", err)
	}
	return nil
}

func (m *FileSystemAbility) AbsPath() string {
	return m.fullpath(m.Path)
}
