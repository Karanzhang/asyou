package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config stores desktop application settings.
type Config struct {
	ServerURL string `json:"server_url"`
	Email     string `json:"email,omitempty"`
	Token     string `json:"token,omitempty"`
}

// configPath returns the path to the config file.
func configPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "asyou", "config.json")
}

// LoadConfig reads the config file or returns defaults.
func LoadConfig() *Config {
	cfg := &Config{ServerURL: "http://localhost:8080"}
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}
	json.Unmarshal(data, cfg)
	return cfg
}

// Save persists the config to disk.
func (c *Config) Save() error {
	path := configPath()
	os.MkdirAll(filepath.Dir(path), 0755)
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
