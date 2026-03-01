# Agent Dashboard

Persistent live-updating TUI for monitoring Claude Code and Codex agent instances across tmux sessions. Built with Go + Bubbletea.

## Architecture

- **Go + Bubbletea** TUI framework with lipgloss styling
- Runs in dedicated tmux session `dashboard`
- Polls tmux panes every 2s to discover agents
- Groups agents by project (tmux session name)
- Detail panel shows metadata, todos, and captured pane output

## Project Structure

```
cmd/dashboard/main.go          # tea.NewProgram entry point
internal/
  tui/
    model.go                   # model struct, Init
    update.go                  # Update handlers
    view.go                    # View rendering
    keys.go                    # key.Binding definitions
    styles.go                  # lipgloss styles
    commands.go                # tea.Cmd factories (tick, capture, switch)
  agent/
    agent.go                   # Agent, AgentStatus, SessionGroup types
    collector.go               # Collect() — orchestrates data gathering
    collector_test.go
    filter.go                  # Substring match on name/session/path
    todo.go                    # Todo JSON parsing
  tmux/
    client.go                  # ListPanes, CapturePaneOutput, SwitchClient
    parse.go                   # parsePanes, ParseStatus
    parse_test.go
```

## Data Sources

| Source | Data |
|---|---|
| `tmux list-panes -a` | Session, window, pane, command, title, pid, cwd |
| `/proc/<pid>/cmdline` | `--team-name`, `--agent-name`, `--parent-session-id` |
| `~/.claude/teams/*/config.json` | Team name, member roles, model |
| `~/.claude/todos/*.json` | Task content, status, activeForm |
| `tmux capture-pane` | Last 20 lines of selected agent output |

## Agent Detection

Filter `pane_current_command` matching `claude`, `codex`, or semver pattern `X.Y.Z` (team subprocesses).

Status from pane title: `✳` = idle, braille dots = active.

## Key Bindings (TUI)

| Key | Action |
|-----|--------|
| `j/↓` | Move down |
| `k/↑` | Move up |
| `Enter` | Jump to agent pane |
| `/` | Filter mode |
| `Esc` | Clear filter |
| `r` | Force refresh |
| `q` | Quit |

## Dependencies

| Package | Purpose |
|---|---|
| `charmbracelet/bubbletea` | TUI framework |
| `charmbracelet/lipgloss` | Styling, layout |
| `charmbracelet/bubbles/viewport` | Scrollable detail output |
| `charmbracelet/bubbles/textinput` | Filter input |
| `charmbracelet/bubbles/key` | Key binding definitions |

## Design Decisions

- **No `bubbles/list`** — agent list is hierarchical (session headers → agents). Custom cursor-over-flat-slice is simpler. ~50 lines.
- **`viewport.Model` for detail panel** — capture-pane output needs scrolling.
- **Shell out to tmux CLI** — `exec.Command("tmux", ...)` not a Go library. tmux CLI is stable, zero deps.
- **Cursor stability across polls** — restore cursor by matching `PaneTarget` (stable identifier). If agent vanished, clamp to bounds.
- **Capture only selected agent** — not all agents. Fires on cursor move and each poll cycle.
- **Polling debounce** — `collecting` flag prevents stacking polls if collection exceeds interval.

## Agent Workflow Rules

| After editing | Run |
|---|---|
| Go files | `go build ./...` and `go vet ./...` |
| Go tests | `go test ./...` |
| Any change | `gofmt` (auto via hook) |

## Build & Run

```bash
go build -o agent-dashboard ./cmd/dashboard
ln -sf ~/src/agent-dashboard/agent-dashboard ~/.local/bin/agent-dashboard
tmux new-session -d -s dashboard 'agent-dashboard'
```

## tmux Integration

Toggle keybind in dotfiles (`config/tmux/config/keybindings.conf`):
```tmux
bind-key C-d if-shell '[ "#{session_name}" = "dashboard" ]' 'switch-client -l' 'switch-client -t dashboard'
```
