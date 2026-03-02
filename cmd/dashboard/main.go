package main

import (
	"fmt"
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jonco/agent-dashboard/internal/config"
	"github.com/jonco/agent-dashboard/internal/tmux"
	"github.com/jonco/agent-dashboard/internal/tui"
)

func main() {
	cfg := config.Load()

	logFile, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening log file: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()
	slog.SetDefault(slog.New(slog.NewTextHandler(logFile, nil)))

	for {
		p := tea.NewProgram(tui.New(cfg), tea.WithAltScreen())
		m, err := p.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		// If we switched to a pane, drain stdin and wait for return.
		// When the user detaches or returns, restart the dashboard.
		final, ok := m.(tui.Model)
		if !ok || final.SwitchedTo == "" {
			break
		}

		tmux.DrainStdin()
	}
}
