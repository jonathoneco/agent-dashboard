package codex

import (
	"bufio"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// sessionCache caches Codex session metadata with file mtime tracking.
type sessionCache struct {
	sessions map[string]*SessionMeta // CWD → session
	mtimes   map[string]time.Time    // file path → last known mtime
	entries  map[string]*SessionMeta // file path → parsed session
}

var globalSessionCache = &sessionCache{
	mtimes:  make(map[string]time.Time),
	entries: make(map[string]*SessionMeta),
}

// LoadSessionsCached returns Codex session metadata, only re-reading files
// whose mtime has changed. Scans the last 7 days of session directories.
func LoadSessionsCached() (map[string]*SessionMeta, error) {
	return globalSessionCache.load()
}

func (c *sessionCache) load() (map[string]*SessionMeta, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return c.sessions, err
	}

	baseDir := filepath.Join(home, ".codex", "sessions")
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return nil, nil
	}

	// Glob JSONL files from the last 7 days.
	var matches []string
	now := time.Now()
	for i := 0; i < 7; i++ {
		d := now.AddDate(0, 0, -i)
		pattern := filepath.Join(baseDir, d.Format("2006"), d.Format("01"), d.Format("02"), "rollout-*.jsonl")
		m, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		matches = append(matches, m...)
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

		meta, err := readFirstLine(path)
		if err != nil {
			slog.Debug("parsing codex session", "path", path, "error", err)
			delete(c.entries, path)
			continue
		}
		meta.SessionFile = path
		meta.LastUpdated = info.ModTime()
		c.entries[path] = meta
	}

	// Remove entries for deleted files.
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

	// Rebuild CWD → session index using most recently updated session file.
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

// readFirstLine reads and parses only the first line of a JSONL file.
func readFirstLine(path string) (*SessionMeta, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// Session files can have large first lines (base_instructions).
	scanner.Buffer(make([]byte, 0, 64*1024), 256*1024)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		return nil, nil
	}

	meta, err := ParseSessionMeta(scanner.Bytes())
	if err != nil {
		return nil, err
	}
	return &meta, nil
}
