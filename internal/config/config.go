package config

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds user-configurable settings for the dashboard.
type Config struct {
	PollInterval time.Duration `yaml:"poll_interval"`
	CaptureLines int           `yaml:"capture_lines"`
	StatusLines  int           `yaml:"status_lines"`
	LogFile      string        `yaml:"log_file"`
}

// defaults returns the default configuration matching previous hardcoded values.
func defaults() Config {
	return Config{
		PollInterval: 2 * time.Second,
		CaptureLines: 20,
		StatusLines:  5,
		LogFile:      filepath.Join(os.TempDir(), "agent-dashboard.log"),
	}
}

// Load reads configuration from ~/.config/agent-dashboard/config.yaml.
// Returns defaults if the file is missing or unreadable.
func Load() *Config {
	cfg := defaults()

	home, err := os.UserHomeDir()
	if err != nil {
		return &cfg
	}

	data, err := os.ReadFile(filepath.Join(home, ".config", "agent-dashboard", "config.yaml"))
	if err != nil {
		return &cfg
	}

	_ = yaml.Unmarshal(data, &cfg)
	return &cfg
}
