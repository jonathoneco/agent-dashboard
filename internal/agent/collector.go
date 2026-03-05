package agent

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jonco/agent-dashboard/internal/codex"
	"github.com/jonco/agent-dashboard/internal/monitor"
	"github.com/jonco/agent-dashboard/internal/tmux"
)

// semverRe matches version strings like "1.2.3" (team subprocesses).
var semverRe = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// agentBinaries are command names that directly indicate an agent process.
var agentBinaries = map[string]bool{
	"claude": true,
	"codex":  true,
}

// wrapperCommands are interpreters that may host an agent binary (e.g. node
// running codex). When a pane runs one of these, we inspect /proc/<pid>/cmdline
// to check if the actual script is an agent binary.
var wrapperCommands = map[string]bool{
	"node":   true,
	"python": true,
}

// isAgentCommand returns true if the pane command indicates an agent process.
// For wrapper commands like "node", it inspects /proc/<pid>/cmdline to check
// if the process is running an agent binary (e.g. node .../bin/codex).
func isAgentCommand(cmd string, pid int) bool {
	if agentBinaries[cmd] {
		return true
	}
	if semverRe.MatchString(cmd) {
		return true
	}
	if wrapperCommands[cmd] {
		return detectAgentInCmdline(pid) != ""
	}
	return false
}

// detectAgentInCmdline reads /proc/<pid>/cmdline and returns the agent binary
// name if found (e.g. "codex", "claude"), or "" if not an agent process.
func detectAgentInCmdline(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return ""
	}
	args := splitCmdline(data)
	for _, arg := range args {
		// Check the basename of each arg for known agent binaries.
		base := filepath.Base(arg)
		if agentBinaries[base] {
			return base
		}
	}
	return ""
}

// Collect discovers all agent panes via tmux and returns them grouped by session.
// statusLines controls how many pane output lines to capture for status inference.
func Collect(statusLines int) ([]SessionGroup, error) {
	panes, err := tmux.ListPanes()
	if err != nil {
		return nil, fmt.Errorf("collect agents: %w", err)
	}

	grouped := make(map[string][]Agent)

	for _, p := range panes {
		if !isAgentCommand(p.Command, p.PID) {
			continue
		}

		teamName, agentName := readCmdlineArgs(resolveAgentPID(p.PID, p.Command))

		// Resolve the actual agent command for wrapper processes
		// (e.g. "node" running codex → command becomes "codex").
		cmd := p.Command
		if wrapperCommands[cmd] {
			if detected := detectAgentInCmdline(p.PID); detected != "" {
				cmd = detected
			}
		}

		agentType := AgentTypeUnknown
		switch cmd {
		case "codex":
			agentType = AgentTypeCodex
		case "claude":
			agentType = AgentTypeClaude
		default:
			if semverRe.MatchString(cmd) {
				agentType = AgentTypeClaude
			}
		}

		name := agentName
		if name == "" {
			name = cmd
		}

		a := Agent{
			Name:       name,
			Session:    p.Session,
			PaneTarget: p.Target(),
			Command:    cmd,
			Status:     tmux.ParseStatus(p.Title),
			CWD:        p.CWD,
			PID:        p.PID,
			TeamName:   teamName,
			AgentType:  agentType,
		}

		grouped[p.Session] = append(grouped[p.Session], a)
	}

	groups := make([]SessionGroup, 0, len(grouped))
	for session, agents := range grouped {
		sort.Slice(agents, func(i, j int) bool {
			return agents[i].PaneTarget < agents[j].PaneTarget
		})
		groups = append(groups, SessionGroup{
			Session: session,
			Agents:  agents,
		})
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Session < groups[j].Session
	})

	teams, err := LoadTeamConfigsCached()
	if err != nil {
		slog.Warn("loading team configs", "error", err)
	} else {
		EnrichWithTeams(groups, teams)
	}

	codexSessions, codexErr := codex.LoadSessionsCached()
	if codexErr != nil {
		slog.Debug("loading codex sessions", "error", codexErr)
	}

	// Fetch process table once for resource monitoring.
	procTable, procErr := monitor.GetProcessTable()
	if procErr != nil {
		slog.Debug("process table unavailable", "error", procErr)
	}

	enrichAgents(groups, statusLines, procTable, codexSessions)
	LinkTeamLeads(groups)

	return groups, nil
}

// enrichAgents captures pane output and computes display name, richer status,
// and resource usage for each agent.
func enrichAgents(groups []SessionGroup, statusLines int, procTable map[int]monitor.ProcessInfo, codexSessions map[string]*codex.SessionMeta) {
	for i := range groups {
		for j := range groups[i].Agents {
			a := &groups[i].Agents[j]
			a.DisplayName = computeDisplayName(a)

			output, err := tmux.CapturePaneOutput(a.PaneTarget, statusLines)
			if err != nil {
				slog.Debug("capture for status enrichment", "target", a.PaneTarget, "error", err)
				continue
			}
			if a.AgentType == AgentTypeCodex {
				a.Status, a.StatusDetail = ParseCodexOutputStatus(output, a.Status)
				if session := codex.FindSession(a.CWD, codexSessions); session != nil {
					a.ModelProvider = session.ModelProvider
					a.CLIVersion = session.CLIVersion
					a.GitBranch = session.GitBranch
					a.SessionSource = session.Source
					a.ParentThread = session.ParentThreadID
					if session.AgentRole != "" {
						a.AgentRole = session.AgentRole
					}
					if session.AgentNickname != "" && isGenericName(a.Name) {
						a.Name = session.AgentNickname
						a.DisplayName = computeDisplayName(a)
					}
					// Codex panes often end with static UI lines. Use session write
					// recency to keep status responsive and avoid sticky "working".
					if !session.LastUpdated.IsZero() {
						recent := time.Since(session.LastUpdated) < 8*time.Second
						if recent && (a.Status == tmux.StatusUnknown || a.Status == tmux.StatusIdle) {
							a.Status = tmux.StatusWorking
							a.StatusDetail = "Active"
						} else if !recent && (a.Status == tmux.StatusUnknown || a.Status == tmux.StatusIdle || a.StatusDetail == "Processing command...") {
							a.Status = tmux.StatusIdle
							a.StatusDetail = "Idle"
						}
					}
				}
			} else {
				a.Status, a.StatusDetail = ParseOutputStatus(output, a.Status)
			}

			if procTable != nil {
				a.CPU, a.Memory = monitor.AggregateResources(a.PID, procTable)
				if info, ok := procTable[a.PID]; ok {
					a.Uptime = info.Elapsed
				}
			}
		}
	}
}

// computeDisplayName returns a human-friendly name for the agent.
// Priority: meaningful --agent-name > basename of CWD > raw command.
func computeDisplayName(a *Agent) string {
	if a.Name != "" && !isGenericName(a.Name) {
		return a.Name
	}
	if a.CWD != "" {
		base := filepath.Base(a.CWD)
		if base != "." && base != "/" {
			return base
		}
	}
	return a.Command
}

// resolveAgentPID returns the PID whose cmdline should be inspected for agent
// flags. For semver commands (team subprocesses) the pane PID is the parent
// shell; the actual agent binary is a direct child.
func resolveAgentPID(pid int, cmd string) int {
	if !semverRe.MatchString(cmd) {
		return pid
	}
	// /proc/<pid>/task/<pid>/children is a flat file with space-separated child PIDs.
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/task/%d/children", pid, pid))
	if err != nil {
		return findChildByPPID(pid)
	}
	fields := strings.Fields(strings.TrimSpace(string(data)))
	for _, f := range fields {
		var childPID int
		if _, err := fmt.Sscanf(f, "%d", &childPID); err == nil {
			return childPID
		}
	}
	return pid
}

// findChildByPPID finds a child process by scanning /proc entries.
func findChildByPPID(ppid int) int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return ppid
	}
	target := fmt.Sprintf("%d", ppid)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// Only numeric directories.
		var childPID int
		if _, err := fmt.Sscanf(e.Name(), "%d", &childPID); err != nil {
			continue
		}
		statData, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", childPID))
		if err != nil {
			continue
		}
		// Format: pid (comm) state ppid ...
		// Find the ppid field after the closing paren.
		s := string(statData)
		closeParen := strings.LastIndex(s, ")")
		if closeParen < 0 || closeParen+2 >= len(s) {
			continue
		}
		fields := strings.Fields(s[closeParen+2:])
		if len(fields) >= 2 && fields[1] == target {
			return childPID
		}
	}
	return ppid
}

// readCmdlineArgs reads /proc/<pid>/cmdline and extracts --team-name and
// --agent-name flag values. Returns empty strings on any error.
func readCmdlineArgs(pid int) (teamName, agentName string) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return "", ""
	}
	return parseCmdlineArgs(data)
}

// parseCmdlineArgs parses a null-byte-separated cmdline blob and extracts
// --team-name and --agent-name flag values.
func parseCmdlineArgs(data []byte) (teamName, agentName string) {
	args := splitCmdline(data)

	for i := 0; i < len(args)-1; i++ {
		switch args[i] {
		case "--team-name":
			teamName = args[i+1]
		case "--agent-name":
			agentName = args[i+1]
		}
	}

	return teamName, agentName
}

// CaptureOutput captures the last N lines of a pane's output.
func CaptureOutput(target string, lines int) (string, error) {
	return tmux.CapturePaneOutput(target, lines)
}

func isGenericName(name string) bool {
	if agentBinaries[name] {
		return true
	}
	return semverRe.MatchString(name)
}

// splitCmdline splits null-byte-separated /proc/pid/cmdline data into
// individual arguments, discarding empty trailing entries.
func splitCmdline(data []byte) []string {
	// /proc/pid/cmdline is null-terminated; trim trailing null(s).
	data = bytes.TrimRight(data, "\x00")
	if len(data) == 0 {
		return nil
	}

	parts := strings.Split(string(data), "\x00")
	return parts
}
