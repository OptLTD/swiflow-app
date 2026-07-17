// Package store defines the persistence interface and domain types for Swiflow.
// Spec §6.2, §5.
package store

import (
	"context"
)

// Provider is an LLM endpoint configuration. ApiKey is stored plaintext for now.
// Model is the default model id used when an agent references this provider via txt_model/img_model.
type Provider struct {
	ID        string `json:"id" db:"id"`
	Tid       string `json:"tid" db:"tid"`
	Name      string `json:"name" db:"name"`
	Display   string `json:"display" db:"display"`
	ApiBase   string `json:"api_base" db:"api_base"`
	ApiKey    string `json:"api_key,omitempty" db:"-"`
	Model     string `json:"model" db:"model"`
	Enabled   bool   `json:"enabled" db:"enabled"`
	CreatedAt string `json:"created_at" db:"created_at"`
	UpdatedAt string `json:"updated_at" db:"updated_at"`
}

// Agent is a runnable agent configuration.
// TxtModel / ImgModel reference llm_provider.name.
type Agent struct {
	ID        string `json:"id" db:"id"`
	Tid       string `json:"tid" db:"tid"`
	Key       string `json:"key" db:"key"`
	Display   string `json:"display" db:"display"`
	TxtModel  string `json:"txt_model" db:"txt_model"`
	ImgModel  string `json:"img_model" db:"img_model"`
	SysPrompt string `json:"sys_prompt" db:"sys_prompt"`
	CreatedAt string `json:"created_at" db:"created_at"`
	UpdatedAt string `json:"updated_at" db:"updated_at"`
}

// Session is a conversation thread.
type Session struct {
	ID        string `json:"id" db:"id"`
	Tid       string `json:"tid" db:"tid"`
	Agent     string `json:"agent" db:"agent"`
	Title     string `json:"title" db:"title"`
	Parent    string `json:"parent" db:"parent"`
	CreatedAt string `json:"created_at" db:"created_at"`
	UpdatedAt string `json:"updated_at" db:"updated_at"`
}

// Message is one turn in a session.
// ToolCalls is decoded for the API; the sqlstore persists it as JSON text.
type Message struct {
	ID         string     `json:"id" db:"id"`
	Tid        string     `json:"tid" db:"tid"`
	Sid        string     `json:"sid" db:"sid"`
	Seq        int        `json:"seq" db:"seq"`
	Role       string     `json:"role" db:"role"`
	Content    string     `json:"content" db:"content"`
	Thinking   string     `json:"thinking" db:"thinking"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty" db:"-"`
	ToolCallId string     `json:"tool_call_id,omitempty" db:"tool_call_id"`
	ToolName   string     `json:"tool_name,omitempty" db:"tool_name"`
	CreatedAt  string     `json:"created_at" db:"created_at"`
}

// ToolCall is a model-requested tool invocation persisted with an assistant message.
type ToolCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// ToolPolicy is the enable state for a tool (not a direct table row).
type ToolPolicy struct {
	ToolName string `json:"tool_name"`
	Enabled  bool   `json:"enabled"`
}

// Experience is a persisted agent learning record.
type Experience struct {
	ID        string   `json:"id" db:"id"`
	Tid       string   `json:"tid" db:"tid"`
	Sid       string   `json:"sid" db:"sid"`
	Agent     string   `json:"agent" db:"agent"`
	Summary   string   `json:"summary" db:"summary"`
	Outcome   string   `json:"outcome" db:"outcome"` // success|partial|failure|unknown
	Tags      []string `json:"tags" db:"-"`
	CreatedAt string   `json:"created_at" db:"created_at"`
}

// MCPServer is a configured MCP server connection (Phase 2).
// Args / Env are decoded for the API; the sqlstore persists them as JSON text.
type MCPServer struct {
	ID   string   `json:"id" db:"id"`
	Tid  string   `json:"tid" db:"tid"`
	Name string   `json:"name" db:"name"`
	Type string   `json:"type" db:"type"` // stdio|sse|streamable
	Cmd  string   `json:"cmd,omitempty" db:"cmd"`
	Args []string `json:"args,omitempty" db:"-"`
	URL  string   `json:"url,omitempty" db:"url"`

	Env map[string]string `json:"env,omitempty" db:"-"`

	Enabled   bool   `json:"enabled" db:"enabled"`
	CreatedAt string `json:"created_at" db:"created_at"`
	UpdatedAt string `json:"updated_at" db:"updated_at"`
}

// CronJob is a scheduled agent task (Phase 2).
type CronJob struct {
	ID        string `json:"id" db:"id"`
	Tid       string `json:"tid" db:"tid"`
	Name      string `json:"name" db:"name"`
	Agent     string `json:"agent" db:"agent"`
	Message   string `json:"message" db:"message"`
	Schedule  string `json:"schedule" db:"schedule"`
	Enabled   bool   `json:"enabled" db:"enabled"`
	LastRunAt string `json:"last_run_at,omitempty" db:"last_run_at"`
	CreatedAt string `json:"created_at" db:"created_at"`
	UpdatedAt string `json:"updated_at" db:"updated_at"`
}

// Store is the persistence interface.
type Store interface {
	Close() error

	// Providers
	CreateProvider(ctx context.Context, p *Provider) error
	ListProviders(ctx context.Context) ([]Provider, error)
	GetProviderByName(ctx context.Context, name string) (*Provider, error)
	GetProviderByID(ctx context.Context, id string) (*Provider, error)
	ProviderCreds(ctx context.Context, name string) (apiBase, apiKey, model string, err error)
	UpdateProvider(ctx context.Context, id string, fields map[string]any) error
	DeleteProvider(ctx context.Context, id string) error

	// Agents
	CreateAgent(ctx context.Context, a *Agent) error
	ListAgents(ctx context.Context) ([]Agent, error)
	GetAgentByKey(ctx context.Context, key string) (*Agent, error)
	UpdateAgent(ctx context.Context, id string, fields map[string]any) error

	// Sessions + messages
	CreateSession(ctx context.Context, s *Session) error
	GetSessionByID(ctx context.Context, id string) (*Session, error)
	ListSessions(ctx context.Context) ([]Session, error)
	UpdateSessionTitle(ctx context.Context, id, title string) error
	// DeleteSession removes a session, its child (subagent) sessions, and related rows.
	DeleteSession(ctx context.Context, id string) error
	AppendMessage(ctx context.Context, sessionID string, msg Message) (Message, error)
	// UpdateToolMessageByCallID patches a tool message content after soft-async completion.
	UpdateToolMessageByCallID(ctx context.Context, sessionID, toolCallID, content string) error
	ListMessages(ctx context.Context, sessionID string) ([]Message, error)

	// Policy
	ToolEnabled(ctx context.Context, name string) bool
	SetToolEnabled(ctx context.Context, name string, enabled bool) error
	ListToolPolicy(ctx context.Context) ([]ToolPolicy, error)
	DisabledSkills(ctx context.Context) ([]string, error)
	SetSkillEnabled(ctx context.Context, slug string, enabled bool) error
	GetSysSetting(ctx context.Context, key string) (value string, ok bool, err error)
	SetSysSetting(ctx context.Context, key, value string) error

	// MCP servers (Phase 2)
	CreateMCPServer(ctx context.Context, s *MCPServer) error
	ListMCPServers(ctx context.Context) ([]MCPServer, error)
	GetMCPServerByID(ctx context.Context, id string) (*MCPServer, error)
	GetMCPServerByName(ctx context.Context, name string) (*MCPServer, error)
	UpdateMCPServer(ctx context.Context, id string, fields map[string]any) error
	DeleteMCPServer(ctx context.Context, id string) error

	// Cron jobs (Phase 2)
	CreateCronJob(ctx context.Context, j *CronJob) error
	ListCronJobs(ctx context.Context) ([]CronJob, error)
	GetCronJobByID(ctx context.Context, id string) (*CronJob, error)
	UpdateCronJob(ctx context.Context, id string, fields map[string]any) error
	DeleteCronJob(ctx context.Context, id string) error
	SetCronJobLastRun(ctx context.Context, id string, at string) error

	// Experience (Phase 3)
	CreateExperience(ctx context.Context, e *Experience) error
	ListExperiences(ctx context.Context, agentKey string, limit int) ([]Experience, error)
	DeleteExperience(ctx context.Context, id string) error

	// Session todos (Phase 3)
	SaveTodos(ctx context.Context, sessionID string, itemsJSON string) error
	LoadTodos(ctx context.Context, sessionID string) (string, error)
}
