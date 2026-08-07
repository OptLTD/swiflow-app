// Package config loads Swiflow configuration from a JSON file with environment
// overrides. Spec §6.1, §11.
package config

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds server configuration.
type Config struct {
	// HostAddress is the listen address ("host:port").
	HostAddress string `json:"host_address"`
	// DatabaseDSN locates the DB: "sqlite://./data/x.db" or "postgres://...".
	DatabaseDSN string `json:"database_dsn"`

	// Derived from HostAddress / DatabaseDSN after Load (not written to JSON).
	Host     string `json:"-"`
	Port     int    `json:"-"`
	DBDriver string `json:"-"` // sqlite | postgres
	DBPath   string `json:"-"` // sqlite file path
	DBDSN    string `json:"-"` // postgres connection string

	WorkspaceDir   string   `json:"workspace_dir"`
	InitSkillsDir  string   `json:"init_skills_dir"`
	UserSkillsDir  string   `json:"user_skills_dir"`
	LightAppsDir   string   `json:"light_apps_dir"`
	AllowedOrigins []string `json:"allowed_origins"`
	// EncryptionKey encrypts provider API keys at rest (AES-256-GCM via SHA-256 derive).
	EncryptionKey string `json:"encryption_key,omitempty"`
	// LocalMode skips HTTP auth and injects tid=default (Desktop). Not from JSON.
	LocalMode bool `json:"-"`

	Context ContextConfig `json:"context"`
	Tools   ToolsConfig   `json:"tools"`
}

// ContextConfig holds agent-loop / prompt budget settings.
type ContextConfig struct {
	MaxHistoryMsgs int `json:"max_history_msgs"`
	// MaxContextChars budgets in-memory LLM prompt size (rune/byte estimate).
	// Default 120000. 0 disables proactive fitting (overflow emergency compact remains).
	MaxContextChars int `json:"max_context_chars"`
	// MaxConcurrentRuns caps in-flight Runner.Run calls per tenant; 0 = unlimited.
	MaxConcurrentRuns int `json:"max_concurrent_runs"`
	// ToolTimeoutSec wraps each tool Execute; 0 = 120s default.
	ToolTimeoutSec int `json:"tool_timeout_sec"`
	// DisableThinking turns off model reasoning/thinking for chat completions
	// (GLM's `thinking:{type:disabled}`). Cuts latency at the cost of reasoning.
	DisableThinking bool `json:"disable_thinking"`
}

// ToolsConfig controls optional tools.
type ToolsConfig struct {
	ExecEnabled     bool   `json:"exec_enabled"`
	BrowserEnabled  bool   `json:"browser_enabled"`
	BrowserHeadless bool   `json:"browser_headless"`
	DocumentEnabled bool   `json:"document_enabled"`
	DocumentBaseURL string `json:"document_base_url"`
	DocumentAPIKey  string `json:"document_api_key"`
	DocumentModel   string `json:"document_model"`
	DocumentTimeout int    `json:"document_timeout"`
	// web search
	SearchProvider string `json:"search_provider"` // duckduckgo|brave|searxng|bing|google; empty = disabled
	SearchBaseURL  string `json:"search_base_url"` // searxng base URL
	SearchAPIKey   string `json:"search_api_key"`  // brave (or similar)
}

// Default returns sensible local defaults.
func Default() Config {
	return Config{
		HostAddress:   "127.0.0.1:8000",
		DatabaseDSN:   "sqlite://./data/swiflow.db",
		UserSkillsDir: "./data/user-skills",
		WorkspaceDir:  "./data/workspace",
		LightAppsDir:  "./data/light-apps",

		Context: ContextConfig{
			MaxHistoryMsgs:  100,
			MaxContextChars: 120_000,
			DisableThinking: true,
		},
		Tools: ToolsConfig{
			BrowserHeadless: true,
			DocumentEnabled: true,
			DocumentTimeout: 120,
			SearchProvider:  "duckduckgo",
		},
	}
}

// Load reads config from path, then applies env overrides (env wins).
func Load(path string) (Config, error) {
	cfg := Default()
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return cfg, fmt.Errorf("read config: %w", err)
			}
		} else if err := json.Unmarshal(data, &cfg); err != nil {
			return cfg, fmt.Errorf("parse config: %w", err)
		}
	}
	applyEnv(&cfg)
	if err := cfg.Normalize(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// UnmarshalJSON merges into the receiver so omitted keys (and partial
// "context" objects) keep Default() values instead of zeroing them.
func (c *Config) UnmarshalJSON(data []byte) error {
	keptCtx := c.Context
	keptTools := c.Tools
	var w struct {
		HostAddress    string          `json:"host_address"`
		DatabaseDSN    string          `json:"database_dsn"`
		WorkspaceDir   string          `json:"workspace_dir"`
		InitSkillsDir  string          `json:"init_skills_dir"`
		UserSkillsDir  string          `json:"user_skills_dir"`
		LightAppsDir   string          `json:"light_apps_dir"`
		AllowedOrigins []string        `json:"allowed_origins"`
		EncryptionKey  string          `json:"encryption_key"`
		Context        json.RawMessage `json:"context"`
		Tools          *ToolsConfig    `json:"tools"`
	}
	w.HostAddress = c.HostAddress
	w.DatabaseDSN = c.DatabaseDSN
	w.WorkspaceDir = c.WorkspaceDir
	w.InitSkillsDir = c.InitSkillsDir
	w.UserSkillsDir = c.UserSkillsDir
	w.LightAppsDir = c.LightAppsDir
	w.AllowedOrigins = c.AllowedOrigins
	w.EncryptionKey = c.EncryptionKey
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	c.HostAddress = w.HostAddress
	c.DatabaseDSN = w.DatabaseDSN
	c.WorkspaceDir = w.WorkspaceDir
	c.InitSkillsDir = w.InitSkillsDir
	c.UserSkillsDir = w.UserSkillsDir
	c.LightAppsDir = w.LightAppsDir
	if w.AllowedOrigins != nil {
		c.AllowedOrigins = w.AllowedOrigins
	}
	if w.EncryptionKey != "" {
		c.EncryptionKey = w.EncryptionKey
	}
	c.Context = keptCtx
	if len(w.Context) > 0 && string(w.Context) != "null" {
		var patch struct {
			MaxHistoryMsgs    *int  `json:"max_history_msgs"`
			MaxContextChars   *int  `json:"max_context_chars"`
			MaxConcurrentRuns *int  `json:"max_concurrent_runs"`
			ToolTimeoutSec    *int  `json:"tool_timeout_sec"`
			DisableThinking   *bool `json:"disable_thinking"`
		}
		if err := json.Unmarshal(w.Context, &patch); err != nil {
			return fmt.Errorf("context: %w", err)
		}
		if patch.MaxHistoryMsgs != nil {
			c.Context.MaxHistoryMsgs = *patch.MaxHistoryMsgs
		}
		if patch.MaxContextChars != nil {
			c.Context.MaxContextChars = *patch.MaxContextChars
		}
		if patch.MaxConcurrentRuns != nil {
			c.Context.MaxConcurrentRuns = *patch.MaxConcurrentRuns
		}
		if patch.ToolTimeoutSec != nil {
			c.Context.ToolTimeoutSec = *patch.ToolTimeoutSec
		}
		if patch.DisableThinking != nil {
			c.Context.DisableThinking = *patch.DisableThinking
		}
	}
	if w.Tools != nil {
		c.Tools = mergeTools(keptTools, *w.Tools, data)
	} else {
		c.Tools = keptTools
	}
	return nil
}

// mergeTools keeps defaults for tool fields omitted from JSON. ToolsConfig uses
// bools where false is meaningful, so we only overlay keys present in the raw
// "tools" object.
func mergeTools(base, overlay ToolsConfig, root []byte) ToolsConfig {
	var raw struct {
		Tools map[string]json.RawMessage `json:"tools"`
	}
	if err := json.Unmarshal(root, &raw); err != nil || raw.Tools == nil {
		return overlay
	}
	out := base
	if _, ok := raw.Tools["exec_enabled"]; ok {
		out.ExecEnabled = overlay.ExecEnabled
	}
	if _, ok := raw.Tools["browser_enabled"]; ok {
		out.BrowserEnabled = overlay.BrowserEnabled
	}
	if _, ok := raw.Tools["browser_headless"]; ok {
		out.BrowserHeadless = overlay.BrowserHeadless
	}
	if _, ok := raw.Tools["document_enabled"]; ok {
		out.DocumentEnabled = overlay.DocumentEnabled
	}
	if _, ok := raw.Tools["document_base_url"]; ok {
		out.DocumentBaseURL = overlay.DocumentBaseURL
	}
	if _, ok := raw.Tools["document_api_key"]; ok {
		out.DocumentAPIKey = overlay.DocumentAPIKey
	}
	if _, ok := raw.Tools["document_model"]; ok {
		out.DocumentModel = overlay.DocumentModel
	}
	if _, ok := raw.Tools["document_timeout"]; ok {
		out.DocumentTimeout = overlay.DocumentTimeout
	}
	if _, ok := raw.Tools["search_provider"]; ok {
		out.SearchProvider = overlay.SearchProvider
	}
	if _, ok := raw.Tools["search_base_url"]; ok {
		out.SearchBaseURL = overlay.SearchBaseURL
	}
	if _, ok := raw.Tools["search_api_key"]; ok {
		out.SearchAPIKey = overlay.SearchAPIKey
	}
	return out
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("SWIFLOW_HOST_ADDRESS"); v != "" {
		cfg.HostAddress = v
	}
	if v := os.Getenv("SWIFLOW_DATABASE_DSN"); v != "" {
		cfg.DatabaseDSN = v
	}
	if v := os.Getenv("SWIFLOW_WORKSPACE"); v != "" {
		cfg.WorkspaceDir = v
	}
	if v := os.Getenv("SWIFLOW_INIT_SKILLS"); v != "" {
		cfg.InitSkillsDir = v
	}
	if v := os.Getenv("SWIFLOW_USER_SKILLS"); v != "" {
		cfg.UserSkillsDir = v
	}
	if v := os.Getenv("SWIFLOW_EXEC"); v != "" {
		cfg.Tools.ExecEnabled = v == "1" || v == "true"
	}
	if v := os.Getenv("SWIFLOW_BROWSER"); v != "" {
		cfg.Tools.BrowserEnabled = v == "1" || v == "true"
	}
	if v := os.Getenv("SWIFLOW_DOCUMENT"); v != "" {
		cfg.Tools.DocumentEnabled = v == "1" || v == "true"
	}
	if v := os.Getenv("SWIFLOW_DOCUMENT_BASE_URL"); v != "" {
		cfg.Tools.DocumentBaseURL = v
	}
	if v := os.Getenv("SWIFLOW_DOCUMENT_API_KEY"); v != "" {
		cfg.Tools.DocumentAPIKey = v
	}
	if v := os.Getenv("SWIFLOW_DOCUMENT_MODEL"); v != "" {
		cfg.Tools.DocumentModel = v
	}
	if v := os.Getenv("SWIFLOW_DOCUMENT_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Tools.DocumentTimeout = n
		}
	}
	if v := os.Getenv("SWIFLOW_SEARCH_PROVIDER"); v != "" {
		cfg.Tools.SearchProvider = v
	}
	if v := os.Getenv("SWIFLOW_SEARCH_API_KEY"); v != "" {
		cfg.Tools.SearchAPIKey = v
	}
	if v := os.Getenv("SWIFLOW_SEARCH_BASE_URL"); v != "" {
		cfg.Tools.SearchBaseURL = v
	}
	if v := os.Getenv("SWIFLOW_ENCRYPTION_KEY"); v != "" {
		cfg.EncryptionKey = v
	}
	if v := os.Getenv("SWIFLOW_MAX_CONTEXT_CHARS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Context.MaxContextChars = n
		}
	}
	if v := os.Getenv("SWIFLOW_MAX_CONCURRENT_RUNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Context.MaxConcurrentRuns = n
		}
	}
	if v := os.Getenv("SWIFLOW_TOOL_TIMEOUT_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Context.ToolTimeoutSec = n
		}
	}
	if v := os.Getenv("SWIFLOW_DISABLE_THINKING"); v != "" {
		cfg.Context.DisableThinking = v == "1" || v == "true"
	}
}

// Normalize parses HostAddress and DatabaseDSN into Host/Port/DB* fields.
func (c *Config) Normalize() error {
	addr := strings.TrimSpace(c.HostAddress)
	if addr == "" {
		addr = "127.0.0.1:8000"
		c.HostAddress = addr
	}
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid host_address %q: %w", addr, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return fmt.Errorf("invalid host_address port in %q", addr)
	}
	c.Host = host
	c.Port = port

	dsn := strings.TrimSpace(c.DatabaseDSN)
	if dsn == "" {
		dsn = "sqlite://./data/swiflow.db"
		c.DatabaseDSN = dsn
	}
	driver, sqlitePath, pgDSN, err := ParseDatabaseDSN(dsn)
	if err != nil {
		return fmt.Errorf("invalid database_dsn: %w", err)
	}
	c.DBDriver = driver
	switch driver {
	case "sqlite":
		c.DBPath = sqlitePath
		c.DBDSN = ""
	case "postgres":
		c.DBDSN = pgDSN
		// Unused by postgres Open; keeps DataDir / mkdir heuristics working.
		if c.DBPath == "" {
			c.DBPath = "./data/swiflow.db"
		}
	}
	return nil
}

// ParseDatabaseDSN accepts:
//   - sqlite://./data/swiflow.db  /  sqlite:./data/swiflow.db
//   - postgres://...  /  postgresql://...
func ParseDatabaseDSN(dsn string) (driver, sqlitePath, pgDSN string, err error) {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return "", "", "", fmt.Errorf("empty dsn")
	}
	lower := strings.ToLower(dsn)
	switch {
	case strings.HasPrefix(lower, "postgres://"), strings.HasPrefix(lower, "postgresql://"):
		return "postgres", "", dsn, nil
	case strings.HasPrefix(lower, "sqlite://"):
		path := dsn[len("sqlite://"):]
		if path == "" {
			return "", "", "", fmt.Errorf("sqlite dsn missing path")
		}
		return "sqlite", path, "", nil
	case strings.HasPrefix(lower, "sqlite:"):
		path := strings.TrimPrefix(dsn[len("sqlite:"):], "//")
		if path == "" {
			return "", "", "", fmt.Errorf("sqlite dsn missing path")
		}
		return "sqlite", path, "", nil
	default:
		return "", "", "", fmt.Errorf("dsn must start with sqlite:// or postgres:// (got %q)", dsn)
	}
}

// Addr returns host:port.
func (c Config) Addr() string {
	if a := strings.TrimSpace(c.HostAddress); a != "" {
		return a
	}
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

// DataDir returns the persistent data directory (parent of the SQLite DB when
// using sqlite), e.g. %APPDATA%\Swiflow\data on Windows desktop. Falls back to
// the parent of workspace when workspace is …/data/workspace.
func (c Config) DataDir() string {
	if c.DBPath != "" && (c.DBDriver == "" || c.DBDriver == "sqlite" || c.DBDriver == "sqlite3") {
		return filepath.Dir(c.DBPath)
	}
	if c.WorkspaceDir != "" && filepath.Base(c.WorkspaceDir) == "workspace" {
		return filepath.Dir(c.WorkspaceDir)
	}
	if c.WorkspaceDir != "" {
		return c.WorkspaceDir
	}
	return "."
}

// TenantRoots holds per-tenant disk roots.
type TenantRoots struct {
	Workspace string
	Skills    string
	LightApps string
}

// RootsForTenant returns disk roots for tid.
// LocalMode (Desktop) keeps the configured single-root paths.
func (c Config) RootsForTenant(tid string) TenantRoots {
	if tid == "" {
		tid = "default"
	}
	if c.LocalMode {
		return TenantRoots{
			Skills:    c.UserSkillsDir,
			LightApps: c.LightAppsDir,
			Workspace: c.WorkspaceDir,
		}
	}
	base := filepath.Join(c.DataDir(), "tenants", tid)
	return TenantRoots{
		Skills:    filepath.Join(base, "skills"),
		LightApps: filepath.Join(base, "light-apps"),
		Workspace: filepath.Join(base, "workspace"),
	}
}
