# Agent Dashboard: Development History Research

Research compiled from beads issues, git history, source code, CLAUDE.md,
Serena memories, and project artifacts. No Claude Code conversation logs
were found in `~/.claude/projects/` for this project (the directory
`-home-jonco-src-agent-dashboard` does not exist there), so this
reconstruction is drawn entirely from the commit record, issue tracker,
and code.

---

## Timeline Overview

The entire project was built in a remarkably compressed timeframe:

| Date | Phase | Key Commits |
|------|-------|-------------|
| 2026-03-01 | Day 1: bootstrap through phase 4 | 7 commits (6ac1dc0 through 3cd2287) |
| 2026-03-02 | Bug fix: reconnect loop removal | 2 commits (d187ca7, d8650c1) |
| 2026-03-05 | Codex support, team links, README | 1 commit (e90e5ae) |
| 2026-03-11 | Screenshot added to README | 1 commit (352d380) |
| 2026-03-13 | Bug fix: team lead selection after swap-window | 1 commit (824dd1b) |
| 2026-03-17 | Status detection refinements | 2 commits (bf62cfc, 1493d66) |

The core application -- from empty repo to a fully featured TUI with
polling, navigation, filtering, team enrichment, resource monitoring,
help overlay, and more -- was built in a single day (March 1).

---

## Phase 1: Bootstrap and Initial Build (March 1, morning)

### The 10-Issue Plan

The project started with a carefully sequenced dependency chain of 10
beads issues, created all at once:

1. `agent-dashboard-kg4` -- Scaffold project (go mod init, Bubbletea skeleton)
2. `agent-dashboard-5oz` -- Implement tmux client (ListPanes, CapturePaneOutput, SwitchClient)
3. `agent-dashboard-zzh` -- Implement agent collector (detect agents, enrich from /proc, group by session)
4. `agent-dashboard-7bp` -- Build basic TUI with polling (model/update/view cycle, cursor stability)
5. `agent-dashboard-3yl` -- Add navigation and pane switching (j/k, Enter to jump)
6. `agent-dashboard-2iz` -- Add detail panel (viewport, captured output)
7. `agent-dashboard-y11` -- Add filter search (/ key, textinput)
8. `agent-dashboard-0r0` -- Add team enrichment (read ~/.claude/teams/)
9. `agent-dashboard-w4d` -- Add todo display (parse ~/.claude/todos/)
10. `agent-dashboard-3l1` -- Add tmux keybinding to dotfiles

Dependencies were wired so each issue blocked the next, creating a strict
build order. Issues were closed in pairs as each layer was completed:
- 5oz + zzh closed together: "Implemented and tested, 39 tests passing"
- 7bp + 3yl closed together: "polling, cursor stability, j/k nav, enter/switch-client, q/quit all working"
- 2iz + y11 closed together: "Already implemented in update.go (updateFilter, rebuildItems) and view.go"
- 0r0 + w4d closed together: "Team enrichment via team.go + LoadTeamConfigs/EnrichWithTeams; todo display via todo.go"

### Architectural Decision: No bubbles/list

From CLAUDE.md:
> **No `bubbles/list`** -- agent list is hierarchical (session headers ->
> agents). Custom cursor-over-flat-slice is simpler. ~50 lines.

The `listItem` struct in model.go embodies this: a flat slice where each
entry is either a group header, a regular agent, or a team member. Headers
are rendered but cursor skips them. This was much simpler than trying to
bend `bubbles/list` to handle hierarchical data with mixed item types.

### Architectural Decision: Shell out to tmux CLI

> **Shell out to tmux CLI** -- `exec.Command("tmux", ...)` not a Go
> library. tmux CLI is stable, zero deps.

Rather than using a Go tmux library (which would add complexity and
version coupling), the project shells out to the tmux CLI for everything:
listing panes, capturing output, switching clients, creating windows, and
sending keys.

### The Agent Detection Insight

Agent detection works by inspecting `pane_current_command` from tmux. The
initial detection was straightforward for `claude` and `codex`, but the
project discovered that Claude Code's team subprocesses appear with a
semver version string (e.g. "1.2.3") as their command name:

```go
var semverRe = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
```

This was a key finding -- team members spawned by Claude Code don't show
up as "claude" but as their version number.

### First Commit: The Big Bang

The first real code commit (`f4e4229`) was a "full feature set"
implementation: tmux client, agent collector, full TUI with cursor
stability, navigation, filtering, team enrichment, and todo display. This
was followed immediately by the enrichment phases.

---

## Phase 2: Enrichment and Polish (March 1, afternoon)

### Rich Status Inference (commit 155d15e)

This was a significant "aha" moment. The initial status detection was
binary: pane title contains braille dots = active, contains eight-spoked
asterisk = idle. But this missed crucial nuance.

The output-based status enrichment layer was added, parsing the last 5
lines of pane output to detect 7 distinct states:

| Pattern | Detected State | Detail |
|---------|---------------|--------|
| `Standing by for questions` | Standby | Agent finished but waiting |
| `pause plan mode` | Plan mode | Agent reviewing, not executing |
| `bullet toolname(` | Working | "Running Edit...", "Running Bash..." |
| `star spinner text` | Working | Whatever the spinner says |
| `prompt char` | Waiting | Awaiting user input |

The status icon system maps these to Unicode symbols with semantic colors:
green circle = active/working, yellow ring = idle, cyan target = waiting,
magenta diamond = plan mode.

### Display Name Computation

Another UX insight: raw agent names are often unhelpful. The
`computeDisplayName` function implements a priority chain:
1. Meaningful `--agent-name` from cmdline (e.g. "research-assistant")
2. Basename of working directory (e.g. "agent-dashboard")
3. Raw command name as last resort

The `isGenericName` helper filters out names that are just the binary name
or a semver string -- these get overridden by the CWD basename.

### Config File (commit 852c59a)

Replaced hardcoded constants with a YAML config at
`~/.config/agent-dashboard/config.yaml`. The `Load()` function uses a
clean pattern: build defaults, attempt to read file, unmarshal over
defaults. Missing file silently uses defaults.

### BFS Resource Monitoring (commit 852c59a)

Single `ps -eo pid,ppid,%cpu,%mem,etimes` call per poll, then BFS walk
from each agent's PID to aggregate CPU and memory for the entire process
subtree. This is important because an agent's resource usage is spread
across many child processes (the main process, language servers, tool
executors, etc.).

```go
func AggregateResources(rootPID int, table map[int]ProcessInfo) (cpu, mem float64) {
    children := make(map[int][]int)
    for pid, info := range table {
        children[info.PPID] = append(children[info.PPID], pid)
    }
    queue := []int{rootPID}
    // BFS walk...
}
```

### DA1 Terminal Escape Cleanup (commit 9d8ab13)

This was a gnarly bug. When the dashboard exits (via Enter to jump to an
agent pane), Bubble Tea's terminal capability query (DA1) response
sometimes leaks into the target pane, appearing as `[?6c]` garbage
characters.

The fix is a two-part cleanup:
1. `DrainStdin()` -- set a 50ms read deadline and consume any pending
   bytes before switching
2. `CleanDA1()` -- poll the target pane for 500ms after the switch,
   looking for `[?` in the output, and send backspace keys to erase it

```go
func CleanDA1(pane string) {
    for range 10 {
        time.Sleep(50 * time.Millisecond)
        out, err := CapturePaneOutput(pane, 5)
        if err != nil { return }
        if strings.Contains(out, "[?") {
            _ = sendKeys(pane, "BSpace BSpace BSpace BSpace")
            return
        }
    }
}
```

This is the kind of terminal compatibility issue that only surfaces in
real usage -- sending backspace keys to a remote pane to clean up
artifacts from a terminal escape sequence response.

---

## Phase 3-4: Advanced Features (March 1, evening)

### Reconnect Loop -- Added and Then Removed

Issue `agent-dashboard-ctb` added a reconnect loop: after the user
switches to an agent pane, the dashboard would restart rather than exit.
This was committed as part of `3cd2287` on March 1.

The very next day (commit `d8650c1`), this was reverted:

> The reconnect loop prevented the dashboard from closing when switching
> to an agent pane. Since the dashboard runs in a dedicated tmux session
> with a toggle keybind (Ctrl+d), it should exit cleanly on switch so
> the pane closes.

This pivot reveals the design tension: the dashboard runs in its own tmux
session with a toggle keybind. If it reconnects after switching, the tmux
session stays alive but the pane is confused. The clean solution was to
let it die -- the toggle keybind re-launches it when needed.

### Team Config Caching

Both team configs and Codex sessions use mtime-based cache invalidation.
The pattern is consistent across `internal/agent/cache.go` and
`internal/codex/cache.go`:

1. Glob for config files
2. Stat each file and compare mtime to cached value
3. Only re-read files whose mtime changed
4. Remove entries for deleted files
5. Rebuild derived indices only when something changed

This keeps the 2-second poll cycle fast even with many team configs.

### Status-Based Filtering

Filter mode gained `:status` prefix syntax:
- `:idle` -- show only idle agents
- `:active` -- show only active agents
- `:working frontend` -- compound filter: working status AND "frontend" text match

---

## Phase 5: Codex Support (March 2-5)

### The Node Wrapper Problem (commit d187ca7)

Codex runs as `node .../bin/codex` in tmux, so `pane_current_command`
shows "node" rather than "codex". The fix inspects `/proc/<pid>/cmdline`
for wrapper commands:

```go
var wrapperCommands = map[string]bool{
    "node":   true,
    "python": true,
}
```

When a pane runs "node", the dashboard reads the full cmdline and checks
if any argument's basename matches a known agent binary. This generalizes
nicely for future agent types.

### Codex Session JSONL Parsing

Codex stores session metadata in JSONL files under `~/.codex/sessions/`.
The `Source` field is polymorphic -- it can be a string (`"cli"`) or an
object (`{"subagent": {"thread_spawn": {...}}}`). The parser handles both:

```go
if s[0] == '"' {
    var src string
    json.Unmarshal(p.Source, &src)
    meta.Source = src
} else {
    meta.Source = "subagent"
    var sub jsonSubagentSource
    json.Unmarshal(p.Source, &sub)
    // extract parent thread, nickname, role...
}
```

### Codex Status Detection

Codex doesn't set tmux pane titles (no braille spinners or asterisks), so
status is derived entirely from output patterns:

- Interrupt pattern: `bullet text (Ns bullet esc to interrupt)` = working
- Empty prompt: `>` alone = awaiting input
- Status bar: `model-name default dot N% left` = idle
- Session file recency: if the session JSONL was written within 8 seconds,
  the agent is probably still active even if the output looks idle

The 8-second recency threshold is a pragmatic heuristic -- Codex panes
often end with static UI lines, so file modification time provides a
secondary activity signal.

### Team Lead Linking

The `LinkTeamLeads` algorithm (in `team_link.go`) solves a surprisingly
tricky problem: given a flat list of agents in a tmux session, figure out
which "regular" Claude agent is the team lead for each set of team
members.

The approach:
1. Collect all agents with `--team-name` set (these are members)
2. Find which tmux window(s) the members occupy
3. Look for a Claude agent (without a team name) in the same window
4. If multiple candidates, use PID tiebreaking (parent started before children)
5. Fall back to first unassigned Claude agent if no window match

---

## Phase 6: Real-World Bug Fixes (March 13-17)

### The swap-window Bug (commit 824dd1b)

> When multiple agents share a window with team members (e.g. after
> tmux swap-window), pick the agent with the highest PID still lower
> than the minimum member PID, since the parent must have started
> before spawning team members.

After `tmux swap-window`, two unrelated Claude agents can end up in the
same window as a team's members. The original algorithm would pick the
wrong one as lead. The fix uses process creation order: the real parent
must have a PID lower than its spawned team members but higher than any
other agent that was swapped into the window later.

The test case captures the exact scenario:
```go
{
    name: "PID tiebreak when two agents in same window as members",
    groups: []SessionGroup{{
        Session: "myproj",
        Agents: []Agent{
            // Agent at PID 1000 is the real parent (started before members)
            // Agent at PID 3000 was swapped into this window later
            {PaneTarget: "myproj:1.0", AgentType: AgentTypeClaude, PID: 3000, TeamName: ""},
            {PaneTarget: "myproj:1.1", AgentType: AgentTypeClaude, PID: 1000, TeamName: ""},
            {PaneTarget: "myproj:1.2", AgentType: AgentTypeClaude, PID: 2000, TeamName: "team-alpha"},
            {PaneTarget: "myproj:1.3", AgentType: AgentTypeClaude, PID: 2001, TeamName: "team-alpha"},
        },
    }},
    wantLeads: map[string]bool{"myproj:1.1": true},
}
```

### UI Chrome Pollution (commit bf62cfc)

The status detection was misreading Claude Code's persistent UI elements:

> The `>>` status bar and `---` separator lines were consuming slots in
> the 5-line analysis window and matching as StatusWorking. The prompt
> is always visible in Claude Code's pane regardless of activity state.

Two fixes:
1. Added `isUIChrome()` to filter out separator lines (composed entirely
   of box-drawing characters) and the permission mode bar ("shift+tab to
   cycle")
2. The prompt character is only treated as "waiting" when the pane title
   confirms idle status. When the title shows braille (active), the prompt
   is skipped so tool calls and spinners above it get matched instead.

This is documented in the Serena memory file
(`status-detection-pipeline.md`) -- a 253-line reference document
capturing the complete status detection and rendering pipeline.

### No Auto-Select on Launch (issue agent-dashboard-jdt)

A small but important UX decision: `cursor starts at -1, no auto-select
on first data load`. The dashboard opens with no agent highlighted until
the user intentionally navigates. This prevents accidental pane switches
and lets the user survey the landscape first.

---

## Key Design Patterns

### Cursor Stability Across Polls

Every 2 seconds, the entire agent list is rebuilt from scratch (tmux is
re-polled, processes re-inspected). The cursor must not jump around.
Solution: save the selected agent's `PaneTarget` (a stable identifier
like `myproject:0.1`), rebuild the list, then search for that target to
restore the cursor position. If the agent vanished, clamp to bounds.

### Flat List with Typed Items

The `listItem` struct handles three item types in one flat slice:
- `isHeader: true` -- session group header (rendered, not selectable)
- `isTeamMember: true` -- team member (rendered with tree connectors, not selectable)
- Neither -- regular agent (selectable, gets a jump number)

This avoids nested data structures while supporting the hierarchical
display.

### Polling Debounce

The `collecting` flag prevents poll commands from stacking if collection
exceeds the 2-second interval:
```go
case tickMsg:
    if m.collecting {
        return m, tickCmd(m.cfg.PollInterval) // skip, just re-tick
    }
    m.collecting = true
    return m, tea.Batch(collectCmd(...), tickCmd(...))
```

### Process Tree Inspection via /proc

The project reads several /proc entries:
- `/proc/<pid>/cmdline` -- extract `--team-name`, `--agent-name`, `--parent-session-id`
- `/proc/<pid>/task/<pid>/children` -- find child processes for semver team subprocesses
- `/proc/<pid>/stat` -- fallback child detection via ppid field
- `ps -eo pid,ppid,%cpu,%mem,etimes` -- process table for BFS resource aggregation

---

## Dead Ends and Pivots

1. **Reconnect loop** (added March 1, removed March 2): The idea of
   restarting after a pane switch conflicted with the tmux session model.
   Quick pivot to clean exit.

2. **Todo JSON format**: The initial implementation expected a wrapped
   object format for todo JSON files. This was changed to parse raw arrays
   instead (commit 155d15e). The format mismatch was discovered during
   real testing.

3. **Accept edits as status signal**: The `>> accept edits` pattern was
   initially treated as StatusWorking. Commit bf62cfc removed this because
   it matched the permanent permission mode status bar, causing false
   positives.

4. **Prompt as universal waiting signal**: The prompt character was
   initially always treated as StatusWaiting, but it's always visible in
   Claude Code's pane. The fix gates it on idle title status (commit
   bf62cfc).

---

## Interesting Technical Details

### Braille Dot Detection for Activity

Claude Code uses braille characters (U+2800-U+28FF) as a spinner
animation in the pane title. The dashboard detects these with a simple
range check:
```go
for _, r := range title {
    if r >= 0x2800 && r <= 0x28FF {
        return StatusActive
    }
}
```

### Codex Expert Spawning

The dashboard can spawn a Codex "expert teammate" from the TUI (press
`e`). It creates a new tmux window in the same session and sends a
hardcoded prompt:
```go
const expertPrompt = "Become the codebase expert for this repository.
Start by mapping architecture, key workflows, and likely risk areas,
then report findings."
```

### Session File Scanner Buffer Sizing

Codex session JSONL files can have very large first lines (base
instructions). The scanner uses a custom buffer:
```go
scanner.Buffer(make([]byte, 0, 64*1024), 256*1024)
```

### Process Tree Resolution for Team Members

Team subprocesses show up with a semver version as their command. Their
cmdline flags live on a child process, not the pane's PID. The
`resolveAgentPID` function walks `/proc/<pid>/task/<pid>/children` to
find the real agent binary, falling back to a full /proc scan if the
children file is unavailable.

---

## Project Velocity

The beads issue tracker shows the entire initial feature set (10 issues
with dependency chains) was created, claimed, implemented, tested, and
closed within a few hours on March 1. A second wave of 10+ enhancement
issues was created in the afternoon and completed by evening.

The most telling close reasons:
- `agent-dashboard-5oz`: "Implemented and tested, 39 tests passing"
- `agent-dashboard-kg4`: "Project scaffolded: cmd/dashboard/main.go, internal/tui/{model,update,view}.go, bubbletea+lipgloss+bubbles deps added, builds clean"
- `agent-dashboard-7bp`: "Implemented in model.go, update.go, view.go, styles.go, keys.go, commands.go -- polling, cursor stability, j/k nav, enter/switch-client, q/quit all working"

The project went from zero to daily-driver tool in a single day, with
subsequent refinement driven by real-world usage bugs discovered over the
following two weeks.

---

## Sources

- **Beads issues**: `/home/jonco/src/agent-dashboard/.beads/issues.jsonl` (26 issues)
- **Git log**: 13 commits from 2026-03-01 to 2026-03-17
- **CLAUDE.md**: `/home/jonco/src/agent-dashboard/CLAUDE.md` (architecture docs)
- **Serena memory**: `/home/jonco/src/agent-dashboard/.serena/memories/agent-dashboard/status-detection-pipeline.md`
- **Source code**: 28 Go files across cmd/, internal/tui/, internal/agent/, internal/tmux/, internal/monitor/, internal/codex/, internal/config/
- **No conversation logs found**: `~/.claude/projects/` does not contain an agent-dashboard project directory
