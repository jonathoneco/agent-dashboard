package tmux

import (
	"os"
	"os/exec"
	"strings"
	"time"
)

// DrainStdin reads and discards any pending stdin bytes to prevent
// Bubble Tea's DA1 terminal capability response from leaking into
// the target pane after a switch-client.
func DrainStdin() {
	_ = os.Stdin.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
	buf := make([]byte, 256)
	for {
		_, err := os.Stdin.Read(buf)
		if err != nil {
			break
		}
	}
	_ = os.Stdin.SetReadDeadline(time.Time{})
}

// CleanDA1 removes DA1 terminal response artifacts ([?6c etc.) from
// the specified pane. Runs a brief polling loop in the foreground.
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

func sendKeys(target string, keys string) error {
	return exec.Command("tmux", "send-keys", "-t", target, keys).Run()
}
