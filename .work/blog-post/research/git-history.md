# Agent Dashboard: Project History and Technical Deep Dive

## Overview

Agent Dashboard is a persistent live-updating TUI for monitoring Claude Code and Codex agent instances across tmux sessions. Built in Go with the Bubbletea framework, the entire project was developed over approximately 18 days (March 1 - March 17, 2026) in 13 commits. The codebase is ~3,800 lines of Go across 29 files — compact but feature-rich.

The project lives at `/home/jonco/src/agent-dashboard`.

---

## Chronological Development Narrative

### Day 1: March 1 — Bootstrap to Feature-Complete in a Single Day

The most striking aspect of this project is its velocity. On March 1, 2026, eight of the thirteen total commits landed — taking the project from zero to a polished, multi-feature TUI in a single day.

#### Commit 1: `6ac1dc0` — infra: bootstrap agent-dashboard repo

The project started with a full architectural plan before any code. The bootstrap commit included:
- `go.mod` with Bubbletea/lipgloss/bubbles dependencies
- `CLAUDE.md` with detailed architecture docs
- A `Makefile` with build, install, test, lint, clean targets
- A 10-step implementation plan tracked as beads issues with a dependency chain

The beads issue graph reveals deliberate planning. Ten issues were created with dependency relationships forming a DAG:

```
kg4 (scaffold) → 5oz (tmux client) → zzh (collector) → 7bp (basic TUI)
                                          ↘ 0r0 (team enrichment)
                                   7bp → 3yl (navigation) → 2iz (detail panel) → w4d (todo display)
                                   7bp → y11 (filter search)
```

This dependency chain ensured the work could proceed bottom-up: data layer first, then presentation.

#### Commit 2: `f4e4229` — feat: implement agent dashboard TUI with full feature set

The largest single commit delivered the complete foundational architecture:

- **`internal/tmux/`** — Shell out to `tmux list-panes -a` with a 7-field tab-delimited format string to discover all panes across all sessions. Parse pane titles for status: `✳` (sparkle) means idle, braille dots (Unicode range U+2800-U+28FF) mean active. This is how Claude Code signals its state.
- **`internal/agent/`** — `Collect()` orchestrator that calls tmux, filters for agent commands, reads `/proc/<pid>/cmdline` for `--team-name` and `--agent-name` flags, groups by session, enriches with team configs from `~/.claude/teams/*/config.json`, and loads todos from `~/.claude/todos/*.json`.
- **`internal/tui/`** — Bubbletea model with cursor-over-flat-slice pattern (no `bubbles/list` — the list is hierarchical with session headers interleaved with agents). Viewport for scrollable detail panel. Filter via textinput.

Agent detection used a three-pronged approach:
1. Direct match: `pane_current_command` is `claude` or `codex`
2. Semver match: version strings like `1.2.3` (team subprocess agents report their version as the command name)
3. Wrapper detection: `node` or `python` commands inspected via `/proc/<pid>/cmdline` to find agent binaries in the script path

**Design decision — cursor stability across polls**: Since the agent list rebuilds every 2 seconds, the cursor position is saved by `PaneTarget` (e.g., `myproject:0.1`) which is a stable identifier. After each poll, the cursor is restored to the same agent even if the list order changed or agents appeared/disappeared.

This commit closed five beads issues at once: scaffold (kg4), tmux client (5oz), collector (zzh), basic TUI (7bp), navigation (3yl).

#### Commit 3: `155d15e` — feat: add rich status inference, display names, jump keys, and todo fixes

The status system was dramatically expanded from 2 states to 7:

| Status | Meaning | Detection |
|--------|---------|-----------|
| `idle` | Agent waiting | `✳` in pane title |
| `active` | Agent running (title-based) | Braille dots in pane title |
| `working` | Agent running (output-based) | Tool call `●` or spinner `✽` patterns |
| `waiting` | Awaiting human input | `❯` prompt visible when title confirms idle |
| `plan_mode` | In plan mode | `⏸ plan mode` in output |
| `standby` | Standing by for questions | `Standing by for questions` in output |
| `unknown` | Can't determine | Fallback |

The status parsing examines the last 5 non-empty lines of pane output (bottom-up) using regex patterns:
- `toolCallRe`: `● (\w+)\(` — matches tool call invocations like `● Edit(`
- `spinnerRe`: `[✽✻] (.+)` — matches Bubbletea spinner lines

**Key insight**: The `❯` prompt is always visible in Claude Code's pane (it's a persistent input area). The solution: only treat `❯` as "waiting for input" when the pane title confirms idle (`✳`). When actively working (braille title), skip it so spinners and tool calls above it match instead.

Also added:
- Display name computation: meaningful `--agent-name` > CWD basename > raw command
- Jump keys (1-9, 0 for 10th) for instant agent switching
- Todo JSON parsing switched from wrapped object to raw array format
- `slog` file logging to `/tmp/agent-dashboard.log`

#### Commit 4: `852c59a` — feat: add YAML config file and BFS resource monitoring

Two features, both ported from an earlier prototype called `claude-dashboard`:

**YAML config** (`~/.config/agent-dashboard/config.yaml`): Replaced all hardcoded constants with configurable values — `poll_interval`, `capture_lines`, `status_lines`, `log_file`. Missing file gracefully falls back to defaults.

**Resource monitoring**: A single `ps -eo pid,ppid,%cpu,%mem,etimes` call per poll cycle, then BFS (breadth-first search) process tree walk to aggregate CPU% and memory% per agent's entire subprocess tree. The BFS builds a parent-to-children index from the flat process table, then walks from the agent's root PID outward. This correctly accounts for agents that spawn many child processes (LSP servers, build tools, etc.).

#### Commit 5: `9d8ab13` — feat: add scroll indicators, help overlay, DA1 cleanup, and row truncation

Four phase-2 features in one commit:

- **Scroll indicators**: `▲ N more` / `▼ N more` when the agent list exceeds visible terminal height
- **Help overlay** (`?` key): Full-screen overlay showing keybindings and status icon legend, modeled after k9s/htop/lazygit
- **DA1 terminal escape cleanup**: After Bubbletea exits (pane switch), a terminal capability query response `[?6c` can leak into the target pane. Solution: `DrainStdin()` reads and discards pending stdin bytes with a 50ms deadline, then `CleanDA1()` polls the target pane for artifacts and sends backspaces to clean them up.
- **Row truncation**: Ellipsis-based truncation for narrow terminals

#### Commit 6: `3cd2287` — feat: add reconnect loop, status filter, uptime, confirm prompts, team cache

Five phase-3/phase-4 features:

- **Status-based filtering**: `:idle`, `:active`, `:working` prefix syntax in the filter. Compound filters like `:idle frontend` combine status + text match.
- **Uptime display**: Elapsed time per agent from `ps etimes`, formatted as `Xh Ym` or `Xd Yh`
- **Confirmation prompt pattern**: Modal y/n infrastructure for destructive actions
- **Team config cache**: mtime-based invalidation — only re-reads `~/.claude/teams/*/config.json` files whose modification time has changed
- **Reconnect loop**: Dashboard restarts after pane switch instead of exiting (later removed)

#### Commit 7: `d8650c1` — fix: remove reconnect loop, restore exit-on-switch behavior

A quick reversal — the reconnect loop prevented the dashboard from closing when switching to an agent pane. Since the dashboard runs in a dedicated tmux session with a toggle keybind (`Ctrl+d`), it should exit cleanly so the pane closes. The toggle keybind handles re-entry.

### Day 2: March 2 — Codex Support

#### Commit 8: `d187ca7` — feat: detect Codex agents running as node wrapper processes

Codex runs as `node .../bin/codex` in tmux, appearing as command "node" rather than "codex". The solution: `isAgentCommand()` now inspects `/proc/<pid>/cmdline` for wrapper commands (node, python) to detect agent binaries in the script path. The detection system was generalized with `agentBinaries` and `wrapperCommands` maps so future agent types can be added by editing a data structure rather than code.

### Day 5: March 5 — Codex Deep Integration

#### Commit 9: `e90e5ae` — feat: add Codex support, team links, README, and TUI enhancements

The most feature-packed commit after the initial build. Major additions:

**Codex session parsing** (`internal/codex/`): Reads JSONL session files from `~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl`. Parses the first line of each file (a `session_meta` JSON object) to extract model provider, CLI version, git branch, agent nickname/role, and parent thread ID for subagents. The `source` field is polymorphic — either a string `"cli"` or an object `{"subagent": {"thread_spawn": {...}}}`.

**Codex output status parsing**: Separate parser for Codex TUI patterns:
- `codexInterruptRe`: `^[*·]\s*(.*?)\s*\(.*(?:esc|ctrl\+c)\s+to interrupt\)` — active action with interrupt hint
- `codexStatusBarRe`: `^\s*([\w.-]+)\s+\w+\s*·\s*(\d+)%\s*left` — model name and context budget
- `codexPromptRe`: `^›\s*(.*)$` — Codex's prompt (empty = awaiting input, has text = processing)

**Session file mtime-based freshness**: Codex panes often end with static UI lines, making output-based status detection unreliable. The solution: use the session file's modification time as a recency signal. If the file was modified within the last 8 seconds, treat the agent as active; if not, treat it as idle. This avoids "sticky working" status.

**Team link resolution** (`internal/agent/team_link.go`): `LinkTeamLeads()` identifies team leads by finding the `AgentTypeClaude` process in the same tmux window as the team members. Members are rendered as indented tree items with `├`/`└` connectors.

**Codex expert spawning**: `SpawnCodexExpert()` opens a new tmux window in the same project and starts Codex with a focused prompt: _"Become the codebase expert for this repository..."_. Triggered by the `e` key.

**README.md**: User-facing documentation with install instructions, usage guide, and architecture explanation.

### Day 11: March 11 — Screenshot

#### Commit 10: `352d380` — docs: add screenshot to README

Added a screenshot showing the dashboard in action.

### Day 13: March 13 — Bug Fix: Team Lead Selection

#### Commit 11: `824dd1b` — fix: use PID tiebreaking for team lead selection after swap-window

A subtle bug: when multiple agents share a window with team members (e.g., after `tmux swap-window`), the wrong agent could be selected as team lead. The fix: among window candidates, prefer the agent with the highest PID that's still lower than the minimum member PID, since the parent must have started before spawning team members.

### Day 17: March 17 — Status Detection Refinement

#### Commit 12: `bf62cfc` — fix: filter UI chrome from status detection, gate prompt on idle title

The `⏵⏵` status bar and `───` separator lines in Claude Code's UI were consuming slots in the 5-line analysis window and falsely matching as `StatusWorking`. The `❯` prompt was matching regardless of actual state.

Fixes:
- Added `isUIChrome()` to skip separator lines (all `─`/`━`/`═` characters) and the permission mode bar (`(shift+tab to cycle)`)
- Removed the `⏵⏵ accept edits` pattern that matched the permanent status bar
- `❯` prompt now only triggers `StatusWaiting` when the pane title confirms idle (`✳`)

#### Commit 13: `1493d66` — feat: parse --parent-session-id from agent cmdline

Extended `/proc/<pid>/cmdline` parsing to extract the `--parent-session-id` flag alongside `--team-name` and `--agent-name`.

---

## Beads Issue Tracking: The Complete Picture

23 beads issues were created and closed across the project's lifetime. They reveal a structured, phased approach:

### Prerequisites (P0)
- **agent-dashboard-b6v**: Commit in-progress work — a "gate" issue that blocked all phase-1/2 work until the initial feature set was committed (closed as `155d15e`)

### Phase 1: Foundation (P0)
- **agent-dashboard-qgj**: Resource monitoring via BFS process tree
- **agent-dashboard-dvx**: YAML configuration file

### Phase 2: UX Polish (P1)
- **agent-dashboard-hzm**: DA1 terminal escape cleanup (bug)
- **agent-dashboard-j01**: Help overlay
- **agent-dashboard-bu4**: Scroll indicators
- **agent-dashboard-r5r**: Flexible column layout

### Phase 3: Optimization (P2)
- **agent-dashboard-ctb**: Reconnect loop on attach
- **agent-dashboard-kqm**: Async/cached collection pipeline

### Phase 4: Features (P2)
- **agent-dashboard-efh**: Confirmation prompt pattern
- **agent-dashboard-jya**: Uptime display
- **agent-dashboard-bzy**: Status-based filtering

### Bug Fixes
- **agent-dashboard-dr3**: Team members attached to wrong parent after swap-window
- **agent-dashboard-jdt**: No agent selected until intentional navigation (cursor starts at -1)

### Dotfiles Integration
- **agent-dashboard-3l1**: tmux keybinding (`C-d` toggle) added to dotfiles repo

---

## Architecture and Technical Details

### Package Structure (6 internal packages + main)

```
cmd/dashboard/main.go           29 LOC — tea.NewProgram entry point
internal/
  config/config.go              46 LOC — YAML config with sensible defaults
  monitor/process.go            88 LOC — ps + BFS resource aggregation
  codex/session.go             125 LOC — JSONL session metadata parsing
  codex/cache.go               132 LOC — mtime-based session cache
  tmux/client.go                80 LOC — 6 tmux CLI wrappers
  tmux/parse.go                 87 LOC — pane parsing + status detection
  tmux/cleanup.go               43 LOC — DA1 escape cleanup
  agent/agent.go                69 LOC — Agent/SessionGroup types
  agent/collector.go           347 LOC — main orchestrator
  agent/status.go              153 LOC — output-based status inference
  agent/team_link.go           114 LOC — team lead detection
  agent/team.go                 85 LOC — team config loading
  agent/cache.go                76 LOC — mtime-based team cache
  agent/filter.go               62 LOC — substring + status filtering
  agent/todo.go                 52 LOC — Claude todo JSON parsing
  agent/codex_actions.go        33 LOC — Codex expert spawning
  tui/model.go                  69 LOC — Bubbletea model definition
  tui/update.go                430 LOC — message handling, cursor logic
  tui/view.go                  383 LOC — rendering (list, detail, help)
  tui/keys.go                   64 LOC — key bindings
  tui/styles.go                 59 LOC — lipgloss styling
  tui/commands.go               40 LOC — tea.Cmd factories
```

Total: ~2,780 LOC application code + ~1,040 LOC tests = ~3,820 LOC

### Data Flow (per poll cycle)

1. `tmux list-panes -a` with 7-field format string
2. Filter panes where command is `claude`, `codex`, semver, or wrapper running an agent
3. Read `/proc/<pid>/cmdline` for `--team-name`, `--agent-name`, `--parent-session-id`
4. For semver commands (team subprocesses), resolve to the actual child PID via `/proc/<pid>/task/<pid>/children`
5. Group agents by tmux session name
6. Enrich with team configs from `~/.claude/teams/*/config.json` (mtime-cached)
7. Enrich with Codex session metadata from `~/.codex/sessions/` JSONL files (mtime-cached)
8. Capture last N lines of each agent's pane output
9. Parse output for status signals (7 distinct states)
10. Aggregate CPU/memory via single `ps` call + BFS process tree walk
11. Link team leads to members by window co-location + PID ordering
12. Rebuild flat item list, restore cursor position by PaneTarget

### Interesting Implementation Details

**Semver as command name**: Claude Code team subprocesses report their version number (e.g., `1.2.3`) as `pane_current_command` in tmux. This is detected via `semverRe = regexp.MustCompile(`^\d+\.\d+\.\d+$`)`.

**PID resolution for team members**: For semver commands, the pane PID is the parent shell, not the actual agent. Resolution goes through `/proc/<pid>/task/<pid>/children` first (a flat file of space-separated child PIDs), falling back to scanning all `/proc` entries matching the PPID.

**The gg/G vim pattern**: The `pendingG` field on the model implements vim-style `gg` (go to top). First `g` sets `pendingG = true`; second `g` within the same update jumps to the first agent.

**No bubbles/list**: The agent list is hierarchical — session headers interleaved with agents, team members nested under leads. Using `bubbles/list` would have required flattening this hierarchy and losing the visual structure. Instead, a custom ~50-line cursor-over-flat-slice pattern handles navigation, skipping headers and team member rows.

**Cursor starts at -1**: Issue `agent-dashboard-jdt` documents the decision to not auto-select an agent on first data load. The cursor starts at -1 (no selection), requiring intentional navigation.

**DA1 cleanup**: Bubbletea sends a DA1 (Device Attributes request) terminal escape sequence on startup. If the response arrives after `switch-client`, it appears as garbage text `[?6c` in the target pane. The cleanup pattern: drain stdin with a 50ms deadline, then poll the target pane for `[?` artifacts and send backspaces.

**Codex session file polymorphic source**: The `source` field in Codex JSONL can be either a JSON string (`"cli"`) or an object (`{"subagent": {...}}`). Handled by checking if `json.RawMessage` starts with `"` vs `{`.

**mtime-based cache invalidation**: Both team configs and Codex sessions use file modification time to skip re-parsing unchanged files. This keeps the 2-second poll cycle lightweight.

---

## The Story Arc

This project tells the story of building observability tooling for AI agent workflows. The developer runs multiple Claude Code and Codex instances simultaneously across tmux sessions — working on different projects, spawning team subagents, running in parallel. Without a dashboard, tracking which agents are active, waiting for input, or stuck requires manually switching between tmux panes.

The solution is deeply integrated with the specific tooling:
- Reads `/proc` filesystem for process introspection (Linux-specific)
- Parses Claude Code's pane title conventions (sparkle = idle, braille = active)
- Understands Claude Code's `--team-name` and `--agent-name` CLI flags
- Reads Codex's JSONL session format for metadata
- Uses tmux as both the discovery mechanism and the navigation target

The project went from zero to production-ready in a single day (8 commits on March 1), with subsequent commits handling edge cases that only emerge with real-world multi-agent usage — wrong team lead assignment after `swap-window`, UI chrome interfering with status detection, the persistent prompt appearing as "waiting" during active work.

Every commit addresses a real, observed problem. There are no speculative features. The beads issue tracker shows disciplined progression through a dependency graph, and the close reasons document concrete outcomes rather than aspirational descriptions.
