// Package store defines the persistence interface and domain types for Swiflow.
// Spec §6.2, §5.
package store

import (
	"context"
)

// Provider is an LLM endpoint configuration. APIKey is plaintext in memory;
// the storage layer encrypts/decrypts at the boundary.
type Provider struct {
	ID          string `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	DisplayName string `json:"display_name" db:"display_name"`
	APIBase     string `json:"api_base" db:"api_base"`
	APIKey      string `json:"api_key,omitempty" db:"-"`
	Enabled     bool   `json:"enabled" db:"enabled"`
	CreatedAt   string `json:"created_at" db:"created_at"`
	UpdatedAt   string `json:"updated_at" db:"updated_at"`
}

// Agent is a runnable agent configuration.
type Agent struct {
	ID          string `json:"id" db:"id"`
	Key         string `json:"key" db:"key"`
	DisplayName string `json:"display_name" db:"display_name"`
	Provider    string `json:"provider" db:"provider"`
	Model       string `json:"model" db:"model"`
	SystemExtra string `json:"system_extra" db:"system_extra"`
	CreatedAt   string `json:"created_at" db:"created_at"`
	UpdatedAt   string `json:"updated_at" db:"updated_at"`
}

// Session is a conversation thread.
type Session struct {
	ID        string `json:"id" db:"id"`
	Key       string `json:"key" db:"key"`
	AgentKey  string `json:"agent_key" db:"agent_key"`
	Title     string `json:"title" db:"title"`
	CreatedAt string `json:"created_at" db:"created_at"`
	UpdatedAt string `json:"updated_at" db:"updated_at"`
}

// Message is one turn in a session.
type Message struct {
	ID           string `json:"id" db:"id"`
	SessionID    string `json:"session_id" db:"session_id"`
	Seq          int    `json:"seq" db:"seq"`
	Role         string `json:"role" db:"role"`
	Content      string `json:"content" db:"content"`
	Thinking     string `json:"thinking" db:"thinking"`
	ToolCallsJSON string `json:"tool_calls_json" db:"tool_calls_json"`
	ToolCallID   string `json:"tool_call_id" db:"tool_call_id"`
	ToolName     string `json:"tool_name" db:"tool_name"`
	CreatedAt    string `json:"created_at" db:"created_at"`
}

// ToolPolicy is the enable state for a tool.
type ToolPolicy struct {
	ToolName string `json:"tool_name" db:"tool_name"`
	Enabled  bool   `json:"enabled" db:"enabled"`
}

// MCPServer is a configured MCP server connection (Phase 2).
type MCPServer struct {
	ID          string   `json:"id" db:"id"`
	Name        string   `json:"name" db:"name"`
	DisplayName string   `json:"display_name" db:"display_name"`
	Transport   string   `json:"transport" db:"transport"` // stdio|sse|streamable
	Command     string   `json:"command,omitempty" db:"command"`
	Args        []string `json:"args,omitempty" db:"-"`
	ArgsJSON    string   `json:"-" db:"args_json"`
	URL         string   `json:"url,omitempty" db:"url"`
	Env         map[string]string `json:"env,omitempty" db:"-"`
	EnvJSON     string   `json:"-" db:"env_json"`
	Enabled     bool     `json:"enabled" db:"enabled"`
	CreatedAt   string   `json:"created_at" db:"created_at"`
	UpdatedAt   string   `json:"updated_at" db:"updated_at"`
}

// CronJob is a scheduled agent task (Phase 2).
type CronJob struct {
	ID        string `json:"id" db:"id"`
	Name      string `json:"name" db:"name"`
	AgentKey  string `json:"agent_key" db:"agent_key"`
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
	ProviderCreds(ctx context.Context, name string) (apiBase, apiKey string, err error)
	UpdateProvider(ctx context.Context, id string, fields map[string]any) error
	DeleteProvider(ctx context.Context, id string) error

	// Agents
	CreateAgent(ctx context.Context, a *Agent) error
	ListAgents(ctx context.Context) ([]Agent, error)
	GetAgentByKey(ctx context.Context, key string) (*Agent, error)
	UpdateAgent(ctx context.Context, id string, fields map[string]any) error

	// Sessions + messages
	CreateSession(ctx context.Context, s *Session) error
	GetSessionByKey(ctx context.Context, key string) (*Session, error)
	ListSessions(ctx context.Context) ([]Session, error)
	UpdateSessionTitle(ctx context.Context, key, title string) error
	AppendMessage(ctx context.Context, sessionKey string, msg Message) (Message, error)
	ListMessages(ctx context.Context, sessionKey string) ([]Message, error)

	// Policy
	ToolEnabled(ctx context.Context, name string) bool
	SetToolEnabled(ctx context.Context, name string, enabled bool) error
	ListToolPolicy(ctx context.Context) ([]ToolPolicy, error)
	DisabledSkills(ctx context.Context) ([]string, error)
	SetSkillEnabled(ctx context.Context, slug string, enabled bool) error

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
}
