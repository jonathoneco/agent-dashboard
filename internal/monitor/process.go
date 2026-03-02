package monitor

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ProcessInfo holds resource usage for a single process.
type ProcessInfo struct {
	PID     int
	PPID    int
	CPU     float64
	Mem     float64
	Elapsed time.Duration // elapsed time since process start
}

// GetProcessTable runs a single ps call and returns a map of PID → ProcessInfo.
func GetProcessTable() (map[int]ProcessInfo, error) {
	out, err := exec.Command("ps", "-eo", "pid,ppid,%cpu,%mem,etimes").Output()
	if err != nil {
		return nil, fmt.Errorf("get process table: %w", err)
	}
	return parseProcessTable(string(out))
}

// parseProcessTable parses ps output into a process map.
func parseProcessTable(output string) (map[int]ProcessInfo, error) {
	table := make(map[int]ProcessInfo)
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines[1:] { // skip header
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		ppid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		cpu, _ := strconv.ParseFloat(fields[2], 64)
		mem, _ := strconv.ParseFloat(fields[3], 64)
		etimes, _ := strconv.Atoi(fields[4])

		table[pid] = ProcessInfo{
			PID:     pid,
			PPID:    ppid,
			CPU:     cpu,
			Mem:     mem,
			Elapsed: time.Duration(etimes) * time.Second,
		}
	}

	return table, nil
}

// AggregateResources walks the process tree from rootPID via BFS and sums
// CPU and memory usage for the entire subtree.
func AggregateResources(rootPID int, table map[int]ProcessInfo) (cpu, mem float64) {
	// Build parent → children index.
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
