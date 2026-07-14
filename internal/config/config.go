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
	Host           string      `json:"host"`
	Port           int         `json:"port"`
	DBPath         string      `json:"db_path"`
	AuthToken      string      `json:"auth_token"`
	EncryptionKey  string      `json:"encryption_key"`
	WorkspaceDir   string      `json:"workspace_dir"`
	InitSkillsDir  string      `json:"init_skills_dir"`
	UserSkillsDir  string      `json:"user_skills_dir"`
	AllowedOrigins []string    `json:"allowed_origins"`
	MaxHistoryMsgs int         `json:"max_history_msgs"`
	// MaxConcurrentRuns caps in-flight Runner.Run calls globally; 0 = unlimited.
	MaxConcurrentRuns int `json:"max_concurrent_runs"`
	// ToolTimeoutSec wraps each tool Execute; 0 = 120s default.
	ToolTimeoutSec int         `json:"tool_timeout_sec"`
	Tools          ToolsConfig `json:"tools"`
	SkipAuth       bool        `json:"skip_auth"`
}

// ToolsConfig controls optional tools.
type ToolsConfig struct {
	ExecEnabled     bool `json:"exec_enabled"`
	BrowserEnabled  bool `json:"browser_enabled"`
	BrowserHeadless bool `json:"browser_headless"`
	// web search
	SearchProvider string `json:"search_provider"` // duckduckgo|brave|searxng; empty = disabled
	SearchBaseURL  string `json:"search_base_url"` // searxng base URL
	SearchAPIKey   string `json:"search_api_key"`  // brave (or similar)
}

// Default returns sensible local defaults.
func Default() Config {
	return Config{
		Host: "127.0.0.1", Port: 8000,
		DBPath:         "./data/swiflow.db",
		InitSkillsDir:  "", // empty = embedded builtins; set for local dev override
		UserSkillsDir:  "./data/user-skills",
		WorkspaceDir:   "./data/workspace",
		MaxHistoryMsgs: 100,
		Tools:          ToolsConfig{BrowserHeadless: true},
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
	if cfg.AuthToken == "" {
		return cfg, fmt.Errorf("auth_token is required (config or SWIFLOW_AUTH_TOKEN)")
	}
	if cfg.EncryptionKey == "" || len(cfg.EncryptionKey) < 16 {
		return cfg, fmt.Errorf("encryption_key is required and must be at least 16 chars")
	}
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
	if v := os.Getenv("SWIFLOW_DB"); v != "" {
		cfg.DBPath = v
	}
	if v := os.Getenv("SWIFLOW_AUTH_TOKEN"); v != "" {
		cfg.AuthToken = v
	}
	if v := os.Getenv("SWIFLOW_ENCRYPTION_KEY"); v != "" {
		cfg.EncryptionKey = v
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
}

// Addr returns host:port.
func (c Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
