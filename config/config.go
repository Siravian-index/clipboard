package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const defaultMaxEntries = 50

type Config struct {
	MaxEntries      int  `json:"max_entries"`
	KeepWindowOpen  bool `json:"keep_window_open"`
}

func Default() *Config {
	return &Config{
		MaxEntries:     defaultMaxEntries,
		KeepWindowOpen: true,
	}
}

func configPath() (string, error) {
	dir := filepath.Join(os.Getenv("HOME"), ".config", "clipboard-manager")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return Default(), err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Default(), nil
	}
	if err != nil {
		return Default(), err
	}

	cfg := Default()
	if err := json.Unmarshal(data, cfg); err != nil {
		return Default(), err
	}
	return cfg, nil
}

func (c *Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
