package agent

import (
	"os"
	"path/filepath"
	"time"
)

// teamCache caches team configs with file modification time tracking.
type teamCache struct {
	configs map[string]TeamConfig
	mtimes  map[string]time.Time // path → last known mtime
}

var globalTeamCache = &teamCache{
	mtimes: make(map[string]time.Time),
}

// LoadTeamConfigsCached returns team configs, only re-reading files that
// have changed since the last call. On first call, reads everything.
func LoadTeamConfigsCached() (map[string]TeamConfig, error) {
	return globalTeamCache.load()
}

func (c *teamCache) load() (map[string]TeamConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return c.configs, err
	}

	pattern := filepath.Join(home, ".claude", "teams", "*", "config.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return c.configs, err
	}

	// Track which files still exist.
	seen := make(map[string]bool, len(matches))
	changed := false

	for _, path := range matches {
		seen[path] = true

		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		if prev, ok := c.mtimes[path]; ok && info.ModTime().Equal(prev) {
			continue // unchanged
		}

		c.mtimes[path] = info.ModTime()
		changed = true
	}

	// Remove entries for deleted files.
	for path := range c.mtimes {
		if !seen[path] {
			delete(c.mtimes, path)
			changed = true
		}
	}

	if !changed && c.configs != nil {
		return c.configs, nil
	}

	// Re-read all files (simpler than incremental merge).
	configs, err := LoadTeamConfigs()
	if err != nil {
		return c.configs, err
	}
	c.configs = configs
	return c.configs, nil
}
