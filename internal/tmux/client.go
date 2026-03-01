package tmux

import (
	"fmt"
	"os/exec"
	"strings"
)

// ListPanes runs tmux list-panes -a and returns all panes across all sessions.
func ListPanes() ([]RawPane, error) {
	format := strings.Join([]string{
		"#{session_name}",
		"#{window_index}",
		"#{pane_index}",
		"#{pane_current_command}",
		"#{pane_title}",
		"#{pane_pid}",
		"#{pane_current_path}",
	}, "\t")

	out, err := exec.Command("tmux", "list-panes", "-a", "-F", format).Output()
	if err != nil {
		return nil, fmt.Errorf("tmux list-panes: %w", err)
	}

	return parsePanes(string(out))
}

// CapturePaneOutput captures the last N lines of output from a tmux pane.
func CapturePaneOutput(target string, lines int) (string, error) {
	out, err := exec.Command(
		"tmux", "capture-pane", "-p",
		"-t", target,
		"-S", fmt.Sprintf("-%d", lines),
	).Output()
	if err != nil {
		return "", fmt.Errorf("tmux capture-pane -t %s: %w", target, err)
	}

	return string(out), nil
}

// SwitchClient switches the current tmux client to the given target pane.
func SwitchClient(target string) error {
	if err := exec.Command("tmux", "switch-client", "-t", target).Run(); err != nil {
		return fmt.Errorf("tmux switch-client -t %s: %w", target, err)
	}
	return nil
}
