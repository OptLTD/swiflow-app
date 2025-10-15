package ability

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileTypeRestriction(t *testing.T) {
	// 创建临时目录用于测试
	tempDir, err := os.MkdirTemp("", "filetype_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name        string
		filePath    string
		shouldAllow bool
	}{
		{"允许 .txt 文件", "test.txt", true},
		{"允许 .md 文件", "test.md", true},
		{"允许 .py 文件", "test.py", true},
		{"允许 .js 文件", "test.js", true},
		{"允许 .ts 文件", "test.ts", true},
		{"允许 .html 文件", "test.html", true},
		{"允许 .css 文件", "test.css", true},
		{"允许 .json 文件", "test.json", true},
		{"允许 .yaml 文件", "test.yaml", true},
		{"允许 .yml 文件", "test.yml", true},
		{"允许 .xml 文件", "test.xml", true},
		{"允许 .csv 文件", "test.csv", true},
		{"允许 .sql 文件", "test.sql", true},
		{"允许 .sh 文件", "test.sh", true},
		{"允许 .bat 文件", "test.bat", true},
		{"允许 .ps1 文件", "test.ps1", true},
		{"允许 .dockerfile 文件", "test.dockerfile", true},
		{"允许 .gitignore 文件", "test.gitignore", true},
		{"允许 .env 文件", "test.env", true},
		{"允许 .ini 文件", "test.ini", true},
		{"允许 .cfg 文件", "test.cfg", true},
		{"允许 .conf 文件", "test.conf", true},
		{"允许 .log 文件", "test.log", true},
		{"允许 .rst 文件", "test.rst", true},
		{"允许 .toml 文件", "test.toml", true},
		{"允许 .lock 文件", "test.lock", true},
		{"拒绝 .exe 文件", "test.exe", false},
		{"拒绝 .dll 文件", "test.dll", false},
		{"拒绝 .bin 文件", "test.bin", false},
		{"拒绝无扩展名文件", "test", false},
		{"拒绝 .php 文件", "test.php", false},
		{"拒绝 .java 文件", "test.java", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := &FileSystemAbility{
				Base: tempDir,
				Path: tt.filePath,
			}

			err := fs.isAllowedFileType()
			if tt.shouldAllow && err != nil {
				t.Errorf("期望允许文件类型 %s，但被拒绝: %v", tt.filePath, err)
			}
			if !tt.shouldAllow && err == nil {
				t.Errorf("期望拒绝文件类型 %s，但被允许", tt.filePath)
			}
		})
	}
}

func TestWriteWithFileTypeRestriction(t *testing.T) {
	// 创建临时目录用于测试
	tempDir, err := os.MkdirTemp("", "filetype_write_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 测试允许的文件类型
	fs := &FileSystemAbility{
		Base: tempDir,
		Path: "test.txt",
	}

	err = fs.Write("Hello, World!")
	if err != nil {
		t.Errorf("写入允许的文件类型失败: %v", err)
	}

	// 验证文件是否被创建
	filePath := filepath.Join(tempDir, "test.txt")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("文件应该被创建但不存在: %s", filePath)
	}

	// 测试不允许的文件类型
	fs.Path = "test.exe"
	err = fs.Write("Hello, World!")
	if err == nil {
		t.Errorf("写入不允许的文件类型应该失败，但没有失败")
	}
}

func TestReadWithFileTypeRestriction(t *testing.T) {
	// 创建临时目录用于测试
	tempDir, err := os.MkdirTemp("", "filetype_read_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建一个允许的文件类型
	filePath := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(filePath, []byte("Hello, World!"), 0644)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 测试读取允许的文件类型
	fs := &FileSystemAbility{
		Base: tempDir,
		Path: "test.txt",
	}

	data, err := fs.Read()
	if err != nil {
		t.Errorf("读取允许的文件类型失败: %v", err)
	}
	if string(data) != "Hello, World!" {
		t.Errorf("读取的内容不匹配，期望 'Hello, World!'，实际 '%s'", string(data))
	}

	// 测试读取不允许的文件类型
	fs.Path = "test.exe"
	_, err = fs.Read()
	if err == nil {
		t.Errorf("读取不允许的文件类型应该失败，但没有失败")
	}
}
