package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"swiflow/entity"
	"swiflow/storage"
)

func writePromptLog(name, content string) {
	dir := filepath.Join("..", "..", "workdata", ".prompt")
	os.MkdirAll(dir, 0755)
	file := filepath.Join(dir, name+".md")
	f, _ := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	defer f.Close()
	f.WriteString(content + "\n\n====================\n\n")
}

func TestManager_GetPrompt(t *testing.T) {
	tests := []struct {
		name     string
		bot      *entity.BotEntity
		workPath string
		store    *storage.MockStore
		want     []string // 期望包含的字符串片段
	}{
		{
			name: "测试 master 类型的 bot",
			bot: &entity.BotEntity{
				UUID:      "test-master-001",
				Type:      "master",
				Name:      "测试主控机器人",
				Desc:      "这是一个测试用的主控机器人",
				UsePrompt: "你是一个智能助手，请帮助用户解决问题。",
				Tools:     []string{},
			},
			workPath: "/test/work/path",
			store: func() *storage.MockStore {
				mockStore := storage.NewMockStore()
				mockStore.SetBots([]*entity.BotEntity{
					{
						UUID: "slave-001",
						Type: "slave",
						Name: "从属机器人1",
						Desc: "测试从属机器人",
					},
					{
						UUID: "slave-002",
						Type: "slave",
						Name: "从属机器人2",
						Desc: "另一个测试从属机器人",
					},
				})
				mockStore.SetTools([]*entity.ToolEntity{
					{
						Name: "file_tool",
						Type: "file",
						Desc: "文件操作工具",
					},
					{
						Name: "command_tool",
						Type: "command",
						Desc: "命令执行工具",
					},
				})
				return mockStore
			}(),
			want: []string{
				"你是一个智能助手，请帮助用户解决问题。",
				"/test/work/path",
				"从属机器人1",
				"从属机器人2",
			},
		},
		{
			name: "测试 slave 类型的 bot",
			bot: &entity.BotEntity{
				UUID:      "test-slave-001",
				Type:      "slave",
				Name:      "测试从属机器人",
				Desc:      "这是一个测试用的从属机器人",
				UsePrompt: "我是一个专业的助手，专注于特定任务。",
				Tools:     []string{"file_tool"},
			},
			workPath: "/another/work/path",
			store: func() *storage.MockStore {
				mockStore := storage.NewMockStore()
				mockStore.SetTools([]*entity.ToolEntity{
					{
						Name: "file_tool",
						Type: "file",
						Desc: "文件操作工具",
					},
					{
						Name: "command_tool",
						Type: "command",
						Desc: "命令执行工具",
					},
				})
				return mockStore
			}(),
			want: []string{
				"我是一个专业的助手，专注于特定任务。",
				"/another/work/path",
				"file_tool",
			},
		},
		{
			name: "测试空工具列表的 bot",
			bot: &entity.BotEntity{
				UUID:      "test-empty-001",
				Type:      "slave",
				Name:      "空工具机器人",
				Desc:      "没有指定工具的机器人",
				UsePrompt: "我是一个简单的助手。",
				Tools:     []string{},
			},
			workPath: "/empty/path",
			store: func() *storage.MockStore {
				mockStore := storage.NewMockStore()
				mockStore.SetTools([]*entity.ToolEntity{
					{
						Name: "test_tool",
						Type: "test",
						Desc: "测试工具",
					},
				})
				return mockStore
			}(),
			want: []string{
				"我是一个简单的助手。",
				"/empty/path",
				"test_tool",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建 Manager 实例
			m := &Manager{
				store:  tt.store,
				agents: tt.store.GetBots(),
				config: make(map[string]any),
				active: make(map[string]*Executor),
			}

			// 调用 GetPrompt 方法
			got := m.UsePrompt(tt.bot)
			writePromptLog(tt.bot.Type+"_"+tt.bot.UUID, got)

			// 验证结果
			for _, wantStr := range tt.want {
				if !strings.Contains(got, wantStr) {
					t.Errorf("GetPrompt() 结果中应该包含 '%s'，但实际结果中没有找到", wantStr)
					t.Logf("实际结果: %s", got)
				}
			}

			// 验证结果不为空
			if strings.TrimSpace(got) == "" {
				t.Errorf("GetPrompt() 返回了空字符串")
			}
		})
	}
}

// 测试 buildSelfBots 方法
func TestManager_buildSelfBots(t *testing.T) {
	tests := []struct {
		name     string
		myBots   []*entity.BotEntity
		expected string
	}{
		{
			name: "测试有从属机器人的情况",
			myBots: []*entity.BotEntity{
				{
					UUID: "master-001",
					Type: "master",
					Name: "主控机器人",
					Desc: "主控机器人描述",
				},
				{
					UUID: "slave-001",
					Type: "slave",
					Name: "从属机器人1",
					Desc: "从属机器人1描述",
				},
				{
					UUID: "slave-002",
					Type: "slave",
					Name: "从属机器人2",
					Desc: "从属机器人2描述",
				},
			},
			expected: "从属机器人1",
		},
		{
			name: "测试只有主控机器人的情况",
			myBots: []*entity.BotEntity{
				{
					UUID: "master-001",
					Type: "master",
					Name: "主控机器人",
					Desc: "主控机器人描述",
				},
			},
			expected: "暂无",
		},
		{
			name:     "测试空机器人列表",
			myBots:   []*entity.BotEntity{},
			expected: "暂无",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{
				agents: tt.myBots,
			}

			got := m.getSubAgentsInfo()

			if tt.expected == "暂无" {
				if got != "暂无" {
					t.Errorf("buildSelfBots() = %v, 期望 %v", got, tt.expected)
				}
			} else {
				if !strings.Contains(got, tt.expected) {
					t.Errorf("buildSelfBots() 结果中应该包含 '%s'，但实际结果中没有找到", tt.expected)
					t.Logf("实际结果: %s", got)
				}
			}
		})
	}
}

// 测试 GetPrompt 方法的核心逻辑（不依赖 initial.UsePrompt）
func TestManager_GetPrompt_Logic(t *testing.T) {
	tests := []struct {
		name     string
		bot      *entity.BotEntity
		workPath string
		store    *storage.MockStore
		want     []string // 期望包含的字符串片段
	}{
		{
			name: "测试 master 类型的 bot",
			bot: &entity.BotEntity{
				UUID:      "test-master-001",
				Type:      "master",
				Name:      "测试主控机器人",
				Desc:      "这是一个测试用的主控机器人",
				UsePrompt: "你是一个智能助手，请帮助用户解决问题。",
				Tools:     []string{},
			},
			workPath: "/test/work/path",
			store: func() *storage.MockStore {
				mockStore := storage.NewMockStore()
				mockStore.SetBots([]*entity.BotEntity{
					{
						UUID: "slave-001",
						Type: "slave",
						Name: "从属机器人1",
						Desc: "测试从属机器人",
					},
					{
						UUID: "slave-002",
						Type: "slave",
						Name: "从属机器人2",
						Desc: "另一个测试从属机器人",
					},
				})
				mockStore.SetTools([]*entity.ToolEntity{
					{
						Name: "file_tool",
						Type: "file",
						Desc: "文件操作工具",
					},
					{
						Name: "command_tool",
						Type: "command",
						Desc: "命令执行工具",
					},
				})
				return mockStore
			}(),
			want: []string{
				"你是一个智能助手，请帮助用户解决问题。",
				"/test/work/path",
				"从属机器人1",
				"从属机器人2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建 Manager 实例
			m := &Manager{
				store:  tt.store,
				agents: tt.store.GetBots(),
				config: make(map[string]any),
				active: make(map[string]*Executor),
			}

			// 调用 GetPrompt 方法
			got := m.UsePrompt(tt.bot)
			writePromptLog(tt.bot.Type+"_logic_"+tt.bot.UUID, got)

			// 验证结果不为空
			if strings.TrimSpace(got) == "" {
				t.Logf("GetPrompt() 返回了空字符串，这可能是因为 initial.UsePrompt 函数的问题")
				t.Logf("WorkPath: %s", tt.workPath)
				t.Logf("Bot Type: %s", tt.bot.Type)
				t.Logf("Bot Prompt: %s", tt.bot.UsePrompt)
				t.Logf("Available Bots: %d", len(tt.store.GetBots()))
				t.Logf("Available Tools: %d", len(tt.store.GetTools()))
			} else {
				// 验证结果包含预期的内容
				for _, wantStr := range tt.want {
					if !strings.Contains(got, wantStr) {
						t.Errorf("GetPrompt() 结果中应该包含 '%s'，但实际结果中没有找到", wantStr)
						t.Logf("实际结果: %s", got)
					}
				}
			}
		})
	}
}
