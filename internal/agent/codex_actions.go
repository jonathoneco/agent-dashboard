package agent

import (
	"fmt"
	"time"

	"github.com/jonco/agent-dashboard/internal/tmux"
)

const expertPrompt = "Become the codebase expert for this repository. Start by mapping architecture, key workflows, and likely risk areas, then report findings."

// SpawnCodexExpert opens a new tmux window in the same project and starts Codex
// with a focused prompt to build an expert teammate.
func SpawnCodexExpert(base Agent) (string, error) {
	if base.Session == "" || base.CWD == "" {
		return "", fmt.Errorf("missing session or cwd for codex spawn")
	}

	windowName := fmt.Sprintf("codex-expert-%d", time.Now().Unix()%1000)
	target, err := tmux.NewWindowDetached(base.Session, base.CWD, windowName)
	if err != nil {
		return "", err
	}

	cmd := fmt.Sprintf("codex %q", expertPrompt)
	if err := tmux.SendLiteral(target, cmd); err != nil {
		return "", err
	}
	if err := tmux.SendEnter(target); err != nil {
		return "", err
	}
	return target, nil
}
