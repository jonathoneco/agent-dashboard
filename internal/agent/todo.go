package agent

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// Todo represents a single task from a Claude todos JSON file.
type Todo struct {
	Subject    string `json:"subject"`
	Content    string `json:"content"`
	Status     string `json:"status"`     // "pending", "in_progress", "completed"
	ActiveForm string `json:"activeForm"` // present tense shown during in_progress
}

// TodoFile represents the JSON structure of a ~/.claude/todos/*.json file.
type TodoFile struct {
	Todos []Todo `json:"todos"`
}

// LoadTodos reads all ~/.claude/todos/*.json files and returns a combined
// slice of todos. Returns nil with no error if the directory is missing.
func LoadTodos() ([]Todo, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("load todos: %w", err)
	}

	dir := filepath.Join(home, ".claude", "todos")
	matches, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("load todos: %w", err)
	}
	if len(matches) == 0 {
		return nil, nil
	}

	var all []Todo
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			slog.Warn("skipping unreadable todo file", "path", path, "err", err)
			continue
		}

		var tf TodoFile
		if err := json.Unmarshal(data, &tf); err != nil {
			slog.Warn("skipping malformed todo file", "path", path, "err", err)
			continue
		}

		all = append(all, tf.Todos...)
	}

	return all, nil
}
