package builtin

import (
	"fmt"
	"strings"
	"swiflow/ability"
	"swiflow/config"
	"time"
)

type CmdAliasTool struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
	Desc string `json:"desc"`
	Args string `json:"args"`
}

func (a *CmdAliasTool) Prompt() string {
	// Build a human-readable usage prompt for the cmd_alias builtin tool
	var b strings.Builder
	b.WriteString("### **cmd_alias**\n")
	b.WriteString("- 描述：执行预设的命令别名，可追加自定义参数\n")
	b.WriteString("- 入参：追加参数（可选）\n")
	// Show stored alias details if available
	if strings.TrimSpace(a.Name) != "" || strings.TrimSpace(a.Args) != "" {
		b.WriteString("- 预设别名：" + strings.TrimSpace(a.Name) + " " + strings.TrimSpace(a.Args) + "\n")
	}
	b.WriteString("- 示例：\n")
	b.WriteString("```xml\n")
	b.WriteString("<use-builtin-tool>\n")
	b.WriteString("<desc>运行别名并追加参数</desc>\n")
	b.WriteString("<tool>cmd_alias</tool>\n")
	b.WriteString("<args>\n")
	b.WriteString("--help\n")
	b.WriteString("</args>\n")
	b.WriteString("</use-builtin-tool>\n")
	b.WriteString("```\n")
	return b.String()
}

func (a *CmdAliasTool) Handle(args string) (string, error) {
	// Treat args as extra tokens appended to alias; no JSON parsing
	base := strings.TrimSpace(a.Name)
	combined := strings.TrimSpace(base)
	if strings.TrimSpace(a.Args) != "" {
		combined = strings.TrimSpace(combined + " " + strings.TrimSpace(a.Args))
	}
	if strings.TrimSpace(args) != "" {
		combined = strings.TrimSpace(combined + " " + strings.TrimSpace(args))
	}
	if combined == "" {
		return "", fmt.Errorf("no alias command provided")
	}
	// Determine working directory
	home := config.CurrentHome()
	if strings.TrimSpace(home) == "" {
		home = config.GetWorkHome()
	}
	cmd := &ability.DevCommandAbility{Home: home}
	timeout := 15 * time.Second
	// Execute combined alias via shell
	return cmd.Exec(combined, timeout)
}
