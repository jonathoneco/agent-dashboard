package pi

import (
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// sessionCache caches pi session metadata with file mtime tracking.
type sessionCache struct {
	sessions map[string]*SessionMeta // CWD → session
	mtimes   map[string]time.Time    // file path → last known mtime
	entries  map[string]*SessionMeta // file path → parsed session
}

var globalSessionCache = &sessionCache{
	mtimes:  make(map[string]time.Time),
	entries: make(map[string]*SessionMeta),
}

// LoadSessionsCached returns pi session metadata, only re-reading files whose
// mtime has changed.
func LoadSessionsCached() (map[string]*SessionMeta, error) {
	return globalSessionCache.load()
}

func (c *sessionCache) load() (map[string]*SessionMeta, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return c.sessions, err
	}

	baseDir := filepath.Join(home, ".pi", "agent", "sessions")
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return nil, nil
	}

	var matches []string
	err = filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".jsonl") {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return c.sessions, err
	}

	seen := make(map[string]bool, len(matches))
	changed := false

	for _, path := range matches {
		seen[path] = true

		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		if prev, ok := c.mtimes[path]; ok && info.ModTime().Equal(prev) {
			continue
		}

		c.mtimes[path] = info.ModTime()
		changed = true

		meta, err := ParseSessionMeta(path)
		if err != nil {
			slog.Debug("parsing pi session", "path", path, "error", err)
			delete(c.entries, path)
			continue
		}
		meta.SessionFile = path
		meta.LastUpdated = info.ModTime()
		c.entries[path] = meta
	}

	for path := range c.mtimes {
		if !seen[path] {
			delete(c.mtimes, path)
			delete(c.entries, path)
			changed = true
		}
	}

	if !changed && c.sessions != nil {
		return c.sessions, nil
	}

	sessions := make(map[string]*SessionMeta, len(c.entries))
	sessionMtime := make(map[string]time.Time, len(c.entries))
	for _, meta := range c.entries {
		prev, ok := sessionMtime[meta.CWD]
		if !ok || meta.LastUpdated.After(prev) {
			sessions[meta.CWD] = meta
			sessionMtime[meta.CWD] = meta.LastUpdated
		}
	}
	c.sessions = sessions
	return c.sessions, nil
}
