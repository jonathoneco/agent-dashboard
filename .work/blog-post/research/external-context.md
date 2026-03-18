# External Context for Agent Dashboard Blog Post

Researched 2026-03-18. Brief context bullets to help a reader understand why this project matters and where it fits.

---

1. **Bubbletea is the dominant Go TUI framework.** 40.7k GitHub stars, 18,000+ apps built with it, used by Microsoft Azure, AWS, CockroachDB, NVIDIA, and Ubuntu. Based on the Elm Architecture (Model-Update-View). Notable projects in the ecosystem include gh-dash (11k stars, GitHub workflow TUI), superfile (16.9k stars, file manager), and gama (GitHub Actions TUI). Agent Dashboard joins a well-established ecosystem of developer tools built on Charm's stack.

2. **Claude Code's agent teams are a new, experimental coordination layer.** Teams let a lead session spawn independent Claude Code instances as teammates, each with their own context window, communicating via a shared task list and mailbox system. Teams store config in `~/.claude/teams/{name}/config.json` and tasks in `~/.claude/tasks/{name}/`. Split-pane mode uses tmux natively. This is distinct from subagents, which run within a single session. Agent Dashboard monitors exactly this kind of multi-session setup from the outside.

3. **Claude Code subagents add another layer of invisible activity.** Beyond teams, Claude Code spawns built-in subagents (Explore, Plan, general-purpose) that run in isolated context windows within a single session. These don't appear as separate tmux panes but generate significant background work. A monitoring dashboard helps surface what's actually happening across all these layers.

4. **AI coding agents are proliferating with no standard observability story.** Claude Code, Cline (59k stars), Plandex (15k stars), Codex, and others all run as terminal processes. Each tracks its own token usage internally, but there's no unified way to monitor multiple agents running concurrently across a development environment. Agent Dashboard fills this gap for tmux-based workflows.

5. **AI agent observability is an emerging industry category.** LangSmith (from LangChain) positions itself around "see what your agent is really doing" with tracing, evaluation, and debugging. LangGraph offers native streaming for real-time agent reasoning visibility. The pattern is clear: as agents become more autonomous, developers need tools to understand what they're doing. Agent Dashboard applies this same principle to the terminal-native AI coding workflow.

6. **The tmux integration is a differentiator.** Most AI agent monitoring tools are web-based dashboards or cloud platforms (LangSmith, Arize, Datadog). Agent Dashboard takes the opposite approach: it's a terminal-native tool that monitors terminal-native agents, using tmux's own APIs (list-panes, capture-pane) as the data source. This matches the workflow of developers who run agents in tmux sessions rather than IDEs.

7. **Claude Code's team mode already uses tmux for split-pane display.** The official agent teams feature supports a `teammateMode: "tmux"` setting that gives each teammate its own tmux pane. Agent Dashboard complements this by providing a persistent birds-eye view across all sessions, not just within a single team. It also works with ad-hoc agent setups (multiple independent Claude Code sessions) that aren't formally organized as teams.

8. **The /proc/cmdline parsing reveals hidden metadata.** Agent Dashboard reads `--team-name`, `--agent-name`, and `--parent-session-id` from `/proc/<pid>/cmdline` to identify team relationships. This is a Linux-specific technique that surfaces information Claude Code doesn't expose through any UI. It's the kind of systems-level trick that makes terminal tooling powerful.

9. **Developer tool TUIs are having a moment.** The Charm ecosystem has grown significantly, with Bubbletea v2 recently released. Projects like gh-dash, lazygit, and k9s demonstrate that developers want rich terminal interfaces for complex workflows. Agent Dashboard extends this pattern to the new domain of AI agent management.

10. **No existing tool does this.** There is no comparable open-source project that provides a live TUI dashboard for monitoring AI coding agents across tmux sessions. The closest analogs are tmux session managers and system monitors, but none are aware of AI agent semantics (status detection from pane titles, todo parsing, team relationship mapping).
