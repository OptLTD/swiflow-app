// Package config loads Swiflow configuration from a JSON file with environment
// overrides. Spec §6.1, §11.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

// Config holds server configuration.
type Config struct {
	Host string `json:"host"`
	Port int    `json:"port"`
	// DBDriver selects the persistence backend: "sqlite" (default) or "postgres".
	DBDriver string `json:"db_driver"`
	// DBPath is the SQLite file path (when DBDriver=sqlite).
	DBPath string `json:"db_path"`
	// DBDSN is the Postgres connection string (when DBDriver=postgres),
	// e.g. postgres://user:pass@localhost:5432/swiflow?sslmode=disable.
	DBDSN          string   `json:"db_dsn"`
	WorkspaceDir   string   `json:"workspace_dir"`
	InitSkillsDir  string   `json:"init_skills_dir"`
	UserSkillsDir  string   `json:"user_skills_dir"`
	AllowedOrigins []string `json:"allowed_origins"`
	MaxHistoryMsgs int      `json:"max_history_msgs"`
	// MaxConcurrentRuns caps in-flight Runner.Run calls globally; 0 = unlimited.
	MaxConcurrentRuns int `json:"max_concurrent_runs"`
	// ToolTimeoutSec wraps each tool Execute; 0 = 120s default.
	ToolTimeoutSec int `json:"tool_timeout_sec"`
	// DisableThinking turns off model reasoning/thinking for chat completions
	// (GLM's `thinking:{type:disabled}`). Cuts latency at the cost of reasoning.
	DisableThinking bool        `json:"disable_thinking"`
	Tools           ToolsConfig `json:"tools"`
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
		Host: "127.0.0.1", Port: 8000,
		DBDriver: "sqlite", DBPath: "./data/swiflow.db",
		UserSkillsDir: "./data/user-skills",
		WorkspaceDir:  "./data/workspace",

		MaxHistoryMsgs:  100,
		DisableThinking: true,
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
	return cfg, nil
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("SWIFLOW_HOST"); v != "" {
		cfg.Host = v
	}
	if v := os.Getenv("SWIFLOW_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Port = p
		}
	}
	if v := os.Getenv("SWIFLOW_DB_DRIVER"); v != "" {
		cfg.DBDriver = v
	}
	if v := os.Getenv("SWIFLOW_DB"); v != "" {
		cfg.DBPath = v
	}
	if v := os.Getenv("SWIFLOW_DB_DSN"); v != "" {
		cfg.DBDSN = v
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
	if v := os.Getenv("SWIFLOW_MAX_CONCURRENT_RUNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.MaxConcurrentRuns = n
		}
	}
	if v := os.Getenv("SWIFLOW_TOOL_TIMEOUT_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.ToolTimeoutSec = n
		}
	}
	if v := os.Getenv("SWIFLOW_DISABLE_THINKING"); v != "" {
		cfg.DisableThinking = v == "1" || v == "true"
	}
}

// Addr returns host:port.
func (c Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
