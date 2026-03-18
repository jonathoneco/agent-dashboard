---
title: "Building a Dashboard for My AI Agents"
description: "I run 10+ AI coding agents simultaneously across tmux sessions. I built a Go TUI to monitor them all — and learned a lot about how they work under the hood."
pubDate: 2026-03-18
draft: true
tags: [go, bubbletea, tui, claude-code, ai-agents, tmux]
---

I run a lot of AI coding agents at the same time. On any given day I have five or six Claude Code sessions going across different projects, a couple of Codex instances, and a few team-mode runs where a lead agent has spawned three or four teammates. That's ten to fifteen agents spread across tmux sessions.

The problem: I have no idea what any of them are doing. To check if an agent is stuck waiting for input, burning tokens on a wrong approach, or finished and ready for review, I have to manually switch to each tmux pane. With fifteen agents, that's exhausting. I needed a single screen that shows me everything.

So I built one.

## How Agents Appear in tmux

The first question is discovery: how do you find all running AI agents on a machine? The answer is `tmux list-panes -a`, which returns every pane across every session with metadata like the running command, PID, and working directory.

For Claude Code, the pane's command is just `claude`. Easy. But team-mode subagents — the teammates that a lead session spawns into split panes — show up with their *version number* as the command name. Not `claude`, not `agent`, just something like `1.0.52`:

```go
var semverRe = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
```

That was an unexpected find. I'm guessing it's the Node.js process reporting the package version, but regardless, a regex on semver strings is how you detect team subagents.

Codex has a different problem. It runs as `node .../bin/codex`, so tmux reports the command as `node`. The fix is to read `/proc/<pid>/cmdline` and check whether any argument's basename is a known agent binary:

```go
func detectAgentInCmdline(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return ""
	}
	args := splitCmdline(data)
	for _, arg := range args {
		base := filepath.Base(arg)
		if agentBinaries[base] {
			return base
		}
	}
	return ""
}
```

So agent detection works in three layers: direct command match (`claude`, `codex`), semver match for team subprocesses, and `/proc` inspection for wrapper commands like `node`. None of this is documented anywhere — Claude Code doesn't expose which processes are agents through any UI. You have to go find them.

## Inferring What They're Doing

Finding agents is the easy part. Figuring out their *status* is where things get interesting.

Claude Code communicates its state through the tmux pane title. When it's actively running a tool, the title contains braille characters — those animated dot patterns you see in the terminal. When it's idle, the title has a sparkle character (`✳`). This is the first layer of status detection:

```go
func ParseStatus(title string) AgentStatus {
	if strings.ContainsRune(title, '✳') {
		return StatusIdle
	}
	for _, r := range title {
		if r >= 0x2800 && r <= 0x28FF {
			return StatusActive
		}
	}
	return StatusUnknown
}
```

Scanning a Unicode range to detect a spinner animation. Not what I expected to be doing when I started this project.

But two states (active/idle) aren't enough. I want to know *what* an agent is doing — running a tool call, waiting for my input, in plan mode, standing by after finishing. So there's a second layer that parses the actual pane output, checking the last five non-empty lines from the bottom up:

```go
func ParseOutputStatus(output string, titleStatus AgentStatus) (AgentStatus, string) {
	lines := lastNonEmptyLines(output, 5)
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		if strings.Contains(line, "Standing by for questions") {
			return StatusStandby, "Standing by"
		}
		if strings.Contains(line, "⏸ plan mode") {
			return StatusPlanMode, "Plan mode"
		}
		if m := toolCallRe.FindStringSubmatch(line); m != nil {
			return StatusWorking, "Running " + m[1] + "..."
		}
		// ...
	}
}
```

That `toolCallRe` matches patterns like `● Edit(` or `● Bash(` — the tool call indicators that Claude Code renders. The spinner regex catches the animated status lines. Between title parsing and output parsing, the dashboard resolves seven distinct states.

The catch? Claude Code's prompt character `❯` is *always visible* in the pane. It's a persistent input area, not a signal that the agent is waiting. My first version treated every `❯` as "awaiting input," which meant agents showed as waiting even while actively running tools. The fix: only treat the prompt as a waiting signal when the pane *title* confirms idle status. When the title shows braille (active), skip the prompt so the tool calls and spinners above it get matched instead.

There's a similar problem with UI chrome. Claude Code renders box-drawing separator lines (`───`) and a permission mode status bar that contains `⏵⏵`. Both were consuming slots in the 5-line analysis window and triggering false positives. The solution was a filter:

```go
func isUIChrome(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	if strings.Contains(trimmed, "(shift+tab to cycle)") {
		return true
	}
	for _, r := range trimmed {
		if r != '─' && r != '━' && r != '═' {
			return false
		}
	}
	return true
}
```

Every one of these edge cases was discovered through real usage, not anticipated upfront. Status detection is an adversarial parsing problem — the thing you're parsing wasn't designed to be parsed.

## The Bugs You Only Find in Production

My favorite bug involves terminal escape sequences. When you exit a Bubbletea TUI, it sends a DA1 (Device Attributes) request to the terminal. If the response arrives *after* the dashboard has already switched you to a different tmux pane, it leaks into that pane as garbage characters: `[?6c`.

The fix sends backspaces to the target pane to clean it up:

```go
func CleanDA1(pane string) {
	for range 10 {
		time.Sleep(50 * time.Millisecond)
		out, err := CapturePaneOutput(pane, 5)
		if err != nil {
			return
		}
		if strings.Contains(out, "[?") {
			_ = sendKeys(pane, "BSpace BSpace BSpace BSpace")
			return
		}
	}
}
```

Polling a remote tmux pane for terminal escape artifacts and sending backspace keys to erase them. This is the kind of fix you'd never write in a design doc.

Another good one: team lead detection. When Claude Code spawns a team, the lead agent and its teammates end up in the same tmux window. To figure out which agent is the lead, I look for the unassigned Claude agent in the same window as the team members. Simple enough — until `tmux swap-window` puts two unrelated Claude agents in the same window. Now there are two candidates for lead.

The tiebreaker: PIDs. The parent agent must have started *before* its spawned teammates, so its PID is lower than the minimum member PID. Among the candidates, pick the one with the highest PID that's still below the members:

```go
bestPID := 0
for _, j := range windowCandidates {
	pid := g.Agents[j].PID
	if pid > 0 && pid < minMemberPID && pid > bestPID {
		bestPID = pid
		leadIdx = j
	}
}
```

Process creation order as a proxy for parent-child relationships. It works because team members are spawned shortly after the lead, so there's a reliable PID ordering.

## Making It Fast Enough to Be Invisible

The dashboard polls every two seconds. Each cycle rebuilds the entire agent list from scratch — re-querying tmux, re-reading `/proc`, re-parsing output. The key constraint: the cursor can't jump around.

The solution is cursor stability by identity: save the selected agent's `PaneTarget` (a stable identifier like `myproject:0.1`), rebuild the list, then search for that target to restore position. If the agent vanished, clamp to bounds.

Resource monitoring uses a single `ps` call per cycle, then a BFS walk from each agent's root PID to aggregate CPU and memory across the entire process subtree:

```go
func AggregateResources(rootPID int, table map[int]ProcessInfo) (cpu, mem float64) {
	children := make(map[int][]int)
	for pid, info := range table {
		children[info.PPID] = append(children[info.PPID], pid)
	}
	queue := []int{rootPID}
	visited := make(map[int]bool)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if visited[current] {
			continue
		}
		visited[current] = true
		if info, ok := table[current]; ok {
			cpu += info.CPU
			mem += info.Mem
		}
		queue = append(queue, children[current]...)
	}
	return cpu, mem
}
```

This matters because an agent's real resource usage is spread across many children — language servers, build tools, subprocesses. Reporting just the root PID's usage would be meaningless.

Team configs and Codex session files use mtime-based cache invalidation. Glob for the files, stat each one, only re-read what changed. This keeps the poll cycle under 50ms even with a dozen agents running.

## What I Learned

There's no standard observability story for AI coding agents. Tools like LangSmith and LangGraph offer cloud dashboards for agent tracing, but nothing targets terminal-native workflows. If you're running agents in tmux — and power users often are — you're on your own.

Building this dashboard forced me to reverse-engineer how Claude Code and Codex actually work at the process level. Pane titles as status signals. Semver strings as process identifiers. `/proc/cmdline` as a metadata API that the tools never intended to expose. The information is there if you know where to look.

The whole thing is about 3,800 lines of Go, built in a day with refinements over the following two weeks. Every commit after day one fixes a real bug found through actual usage. That's the thing about building tools for your own workflow — you're also the QA team, and you find bugs fast.

The project is [on GitHub](https://github.com/jonathoneco/agent-dashboard) if you want to try it. It requires tmux and Linux (for `/proc`), which narrows the audience, but if you're the kind of person running ten agents at once, you're probably already in tmux.
