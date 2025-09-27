package builtin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"swiflow/ability"
	"swiflow/config"
	"time"
)

type Python3Tool struct {
}

func (a *Python3Tool) Prompt() string {
	var prompt strings.Builder
	prompt.WriteString("### **python3**\n")
	prompt.WriteString("- 描述：执行python3代码\n")
	prompt.WriteString("- 入参：python code\n")
	prompt.WriteString("- 示例：\n")
	prompt.WriteString("```xml\n")
	prompt.WriteString("<use-builtin-tool>\n")
	prompt.WriteString("<desc>获取当前日期</desc>\n")
	prompt.WriteString("<tool>python3</tool>\n")
	prompt.WriteString("<args>\n")
	prompt.WriteString("import datetime\n")
	prompt.WriteString("print(datetime.datetime.now())\n")
	prompt.WriteString("</args>\n")
	prompt.WriteString("</use-builtin-tool>\n")
	prompt.WriteString("```\n\n")
	return prompt.String()
}

func (a *Python3Tool) Handle(args string) (string, error) {
	code := strings.TrimSpace(args)
	if code == "" {
		return "", fmt.Errorf("no python code provided")
	}

	// Determine working directory
	home := config.CurrentHome()
	if strings.TrimSpace(home) == "" {
		home = config.GetWorkHome()
	}
	cmd := &ability.DevCommandAbility{Home: home}

	// Write code to a default file
	filename := "entry.py"
	fullpath := filepath.Join(home, filename)
	if err := os.MkdirAll(filepath.Dir(fullpath), 0755); err != nil {
		return "", fmt.Errorf("prepare dir: %v", err)
	}
	if err := os.WriteFile(fullpath, []byte(code), 0644); err != nil {
		return "", fmt.Errorf("write file: %v", err)
	}

	// Default timeout
	timeout := 30 * time.Second
	return cmd.Run("python", timeout, filename)
}
