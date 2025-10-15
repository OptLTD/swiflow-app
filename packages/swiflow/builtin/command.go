package builtin

import (
	"fmt"
	"strings"
	"swiflow/ability"
	"swiflow/config"
	"time"
)

type CommandTool struct {
}

func (a *CommandTool) Prompt() string {
	var b strings.Builder
	b.WriteString("### **command**\n")
	b.WriteString("- 描述：执行系统shell命令\n")
	b.WriteString("- 入参：shell command\n")
	b.WriteString("- 示例：\n")
	b.WriteString("```xml\n")
	b.WriteString("<use-builtin-tool>\n")
	b.WriteString("<desc>列出当前目录文件</desc>\n")
	b.WriteString("<tool>command</tool>\n")
	b.WriteString("<args>ls -la</args>\n")
	b.WriteString("</use-builtin-tool>\n")
	b.WriteString("```\n\n")
	return b.String()
}

func (a *CommandTool) Handle(args string) (string, error) {
	cmd := strings.TrimSpace(args)
	if cmd == "" {
		return "", fmt.Errorf("no command specified")
	}

	dev := &ability.DevCommandAbility{
		Home: config.CurrentHome(),
	}
	timeout := 30 * time.Second
	_, err := dev.Exec(cmd, timeout)
	return dev.Logs(), err
}
