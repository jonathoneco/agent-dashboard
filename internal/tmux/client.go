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

// NewWindowDetached creates a new detached tmux window and returns pane target.
func NewWindowDetached(session, cwd, name string) (string, error) {
	out, err := exec.Command(
		"tmux", "new-window", "-d", "-P",
		"-t", session,
		"-c", cwd,
		"-n", name,
		"-F", "#{session_name}:#{window_index}.0",
	).Output()
	if err != nil {
		return "", fmt.Errorf("tmux new-window -t %s: %w", session, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// SendLiteral writes text to a pane as literal keystrokes.
func SendLiteral(target, text string) error {
	if err := exec.Command("tmux", "send-keys", "-t", target, "-l", text).Run(); err != nil {
		return fmt.Errorf("tmux send-keys -l -t %s: %w", target, err)
	}
	return nil
}

// SendEnter sends an Enter key to the target pane.
func SendEnter(target string) error {
	if err := exec.Command("tmux", "send-keys", "-t", target, "Enter").Run(); err != nil {
		return fmt.Errorf("tmux send-keys Enter -t %s: %w", target, err)
	}
	return nil
}
