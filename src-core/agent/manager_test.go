package agent

import (
	"os"
	"path/filepath"
)

func writePromptLog(name, content string) {
	dir := filepath.Join("..", "..", "workdata", ".prompt")
	os.MkdirAll(dir, 0755)
	file := filepath.Join(dir, name+".md")
	f, _ := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	defer f.Close()
	f.WriteString(content + "\n\n====================\n\n")
}
