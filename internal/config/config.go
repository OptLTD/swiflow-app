// Package config loads Mira configuration from a JSON file with environment
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
	WebDistDir     string      `json:"web_dist_dir"`
	Tools          ToolsConfig `json:"tools"`
}

// ToolsConfig controls optional tools.
type ToolsConfig struct {
	ExecEnabled       bool   `json:"exec_enabled"`
	WebSearchProvider string `json:"web_search_provider"`
}

// Default returns sensible local defaults.
func Default() Config {
	return Config{
		Host: "127.0.0.1", Port: 18800,
		DBPath:        "./data/mira.db",
		InitSkillsDir: "./skills",
		UserSkillsDir: "./data/skills",
		WorkspaceDir:  "./data/workspace",
		Tools:         ToolsConfig{},
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
		return cfg, fmt.Errorf("auth_token is required (config or MIRA_AUTH_TOKEN)")
	}
	if cfg.EncryptionKey == "" || len(cfg.EncryptionKey) < 16 {
		return cfg, fmt.Errorf("encryption_key is required and must be at least 16 chars")
	}
	return cfg, nil
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("MIRA_HOST"); v != "" {
		cfg.Host = v
	}
	if v := os.Getenv("MIRA_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Port = p
		}
	}
	if v := os.Getenv("MIRA_DB"); v != "" {
		cfg.DBPath = v
	}
	if v := os.Getenv("MIRA_AUTH_TOKEN"); v != "" {
		cfg.AuthToken = v
	}
	if v := os.Getenv("MIRA_ENCRYPTION_KEY"); v != "" {
		cfg.EncryptionKey = v
	}
	if v := os.Getenv("MIRA_WORKSPACE"); v != "" {
		cfg.WorkspaceDir = v
	}
	if v := os.Getenv("MIRA_INIT_SKILLS"); v != "" {
		cfg.InitSkillsDir = v
	}
	if v := os.Getenv("MIRA_USER_SKILLS"); v != "" {
		cfg.UserSkillsDir = v
	}
	if v := os.Getenv("MIRA_EXEC"); v != "" {
		cfg.Tools.ExecEnabled = v == "1" || v == "true"
	}
}

// Addr returns host:port.
func (c Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
