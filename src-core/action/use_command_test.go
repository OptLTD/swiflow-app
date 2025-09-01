package action

import (
	"strings"
	"testing"
	"time"
)

func TestExecuteCommand_HandleBasic(t *testing.T) {
	tempDir := t.TempDir()

	useCmd := &ExecuteCommand{
		Command: "echo 'test'",
	}

	super := &SuperAction{
		Payload: &Payload{
			UUID: "test-uuid",
			Home: tempDir,
			Time: time.Now(),
		},
	}

	// 执行前 Result 应该为 nil
	if useCmd.Result != nil {
		t.Errorf("执行前 Result 应该为 nil")
	}

	result := useCmd.Handle(super)

	// 执行后 Result 应该被初始化
	if useCmd.Result == nil {
		t.Errorf("执行后 Result 应该被初始化")
	}

	// 验证返回的 Result 和 ExecuteCommand.Result 是同一个实例
	if result != useCmd.Result {
		t.Errorf("返回的 Result 和 ExecuteCommand.Result 应该是同一个实例")
	}
}

func TestExecuteCommand_ParseXML(t *testing.T) {
	tests := []struct {
		name     string
		xmlData  string
		expected *ExecuteCommand
	}{
		{
			name: "基本命令解析",
			xmlData: `<execute-command>
				<command>echo "hello world"</command>
			</execute-command>`,
			expected: &ExecuteCommand{
				Command: `echo "hello world"`,
			},
		},
		{
			name: "复杂命令解析",
			xmlData: `<execute-command>
				<command>ls -la | grep "\.go$" | wc -l</command>
			</execute-command>`,
			expected: &ExecuteCommand{
				Command: `ls -la | grep "\.go$" | wc -l`,
			},
		},
		{
			name: "带结果的命令解析",
			xmlData: `<execute-command>
				<command>pwd</command>
				<result>/Users/test</result>
			</execute-command>`,
			expected: &ExecuteCommand{
				Command: `pwd`,
			},
		},
		{
			name: "带错误信息的命令解析",
			xmlData: `<execute-command>
				<command>nonexistentcommand</command>
				<errmsg>command not found</errmsg>
			</execute-command>`,
			expected: &ExecuteCommand{
				Command: `nonexistentcommand`,
			},
		},
		{
			name: "空命令解析",
			xmlData: `<execute-command>
				<command></command>
			</execute-command>`,
			expected: &ExecuteCommand{
				Command: ``,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 使用 action.Parse 解析 XML
			super := Parse(tt.xmlData)

			// 验证解析结果
			if len(super.UseTools) == 0 {
				t.Errorf("期望解析出工具，但没有解析出任何工具")
				return
			}

			// 查找 ExecuteCommand 类型的工具
			var useCmd *ExecuteCommand
			for _, tool := range super.UseTools {
				if cmd, ok := tool.(*ExecuteCommand); ok {
					useCmd = cmd
					break
				}
			}

			if useCmd == nil {
				t.Errorf("期望解析出 ExecuteCommand，但没有找到")
				return
			}

			// 验证命令内容
			if useCmd.Command != tt.expected.Command {
				t.Errorf("命令不匹配，期望: %q, 实际: %q", tt.expected.Command, useCmd.Command)
			}

			// 验证 XML 标签
			if useCmd.XMLName.Local != "execute-command" {
				t.Errorf("XML 标签不匹配，期望: execute-command, 实际: %s", useCmd.XMLName.Local)
			}
		})
	}
}

func TestExecuteCommand_ParseComplexXML(t *testing.T) {
	// 测试复杂的 XML 结构，包含多个 action
	complexXML := `<execute-command>
		<command>echo "test command"</command>
	</execute-command>
	<thinking>这是一个测试</thinking>
	<execute-command>
		<command>ls -la</command>
	</execute-command>`

	super := Parse(complexXML)

	// 验证解析出了多个工具
	if len(super.UseTools) < 2 {
		t.Errorf("期望解析出至少2个工具，实际解析出 %d 个", len(super.UseTools))
	}

	// 统计 ExecuteCommand 的数量
	ExecuteCommandCount := 0
	for _, tool := range super.UseTools {
		if _, ok := tool.(*ExecuteCommand); ok {
			ExecuteCommandCount++
		}
	}

	if ExecuteCommandCount != 2 {
		t.Errorf("期望解析出2个 ExecuteCommand，实际解析出 %d 个", ExecuteCommandCount)
	}

	// 验证 thinking 内容
	if super.Thinking != "这是一个测试" {
		t.Errorf("thinking 内容不匹配，期望: 这是一个测试, 实际: %s", super.Thinking)
	}
}

func TestExecuteCommand_ParseWithMixedContent(t *testing.T) {
	// 测试混合内容的 XML
	mixedXML := `一些文本内容
	<execute-command>
		<command>echo "mixed content test"</command>
	</execute-command>
	更多文本内容`

	super := Parse(mixedXML)

	// 验证解析出了工具
	if len(super.UseTools) == 0 {
		t.Errorf("期望解析出工具，但没有解析出任何工具")
		return
	}

	// 验证第一个工具是 BotReply（文本内容）
	if botReply, ok := super.UseTools[0].(*ToolResult); ok {
		if !strings.Contains(botReply.Content, "一些文本内容") {
			t.Errorf("期望包含文本内容，实际: %s", botReply.Content)
		}
	} else {
		t.Errorf("期望第一个工具是 BotReply")
	}

	// 验证包含 ExecuteCommand
	var foundExecuteCommand bool
	for _, tool := range super.UseTools {
		if cmd, ok := tool.(*ExecuteCommand); ok {
			if cmd.Command == `echo "mixed content test"` {
				foundExecuteCommand = true
				break
			}
		}
	}

	if !foundExecuteCommand {
		t.Errorf("期望找到 ExecuteCommand，但没有找到")
	}
}
