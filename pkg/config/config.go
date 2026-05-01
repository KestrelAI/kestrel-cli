package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	ServerURL    string `json:"server_url"`
	SessionToken string `json:"session_token"`
	APIKey       string `json:"api_key,omitempty"`
	UserID       string `json:"user_id"`
	Email        string `json:"email"`
}

func Dir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".kestrel"
	}
	return filepath.Join(home, ".kestrel")
}

func Path() string {
	return filepath.Join(Dir(), "config.json")
}

func Load() (*Config, error) {
	data, err := os.ReadFile(Path())
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &c, nil
}

func (c *Config) Save() error {
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(Path(), data, 0o600)
}

func (c *Config) Clear() error {
	return os.Remove(Path())
}

func (c *Config) IsLoggedIn() bool {
	return c.ServerURL != "" && (c.SessionToken != "" || c.APIKey != "")
}
