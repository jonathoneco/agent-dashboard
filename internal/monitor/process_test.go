package monitor

import (
	"testing"
	"time"
)

func TestParseProcessTable(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantLen     int
		checkID     int
		wantCPU     float64
		wantMem     float64
		wantElapsed time.Duration
	}{
		{
			name: "basic",
			input: `  PID  PPID %CPU %MEM ELAPSED
  1     0  0.0  0.1    3600
 42     1  5.3  2.1     120
100    42  1.2  0.5      45`,
			wantLen:     3,
			checkID:     42,
			wantCPU:     5.3,
			wantMem:     2.1,
			wantElapsed: 120 * time.Second,
		},
		{
			name:    "header only",
			input:   `  PID  PPID %CPU %MEM ELAPSED`,
			wantLen: 0,
		},
		{
			name: "malformed lines skipped",
			input: `  PID  PPID %CPU %MEM ELAPSED
  abc  1  0.0  0.1  100
  10   1  3.0  1.0  500`,
			wantLen:     1,
			checkID:     10,
			wantCPU:     3.0,
			wantMem:     1.0,
			wantElapsed: 500 * time.Second,
		},
		{
			name: "short lines skipped",
			input: `  PID  PPID %CPU %MEM ELAPSED
  10   1  3.0
  20   1  3.0  1.0  200`,
			wantLen:     1,
			checkID:     20,
			wantCPU:     3.0,
			wantMem:     1.0,
			wantElapsed: 200 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table, err := parseProcessTable(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(table) != tt.wantLen {
				t.Fatalf("got %d entries, want %d", len(table), tt.wantLen)
			}
			if tt.checkID > 0 {
				info, ok := table[tt.checkID]
				if !ok {
					t.Fatalf("PID %d not found", tt.checkID)
				}
				if info.CPU != tt.wantCPU {
					t.Errorf("CPU = %f, want %f", info.CPU, tt.wantCPU)
				}
				if info.Mem != tt.wantMem {
					t.Errorf("Mem = %f, want %f", info.Mem, tt.wantMem)
				}
				if info.Elapsed != tt.wantElapsed {
					t.Errorf("Elapsed = %v, want %v", info.Elapsed, tt.wantElapsed)
				}
			}
		})
	}
}

func TestAggregateResources(t *testing.T) {
	table := map[int]ProcessInfo{
		1:   {PID: 1, PPID: 0, CPU: 1.0, Mem: 0.5},
		10:  {PID: 10, PPID: 1, CPU: 3.0, Mem: 2.0},
		20:  {PID: 20, PPID: 10, CPU: 2.0, Mem: 1.0},
		30:  {PID: 30, PPID: 10, CPU: 1.5, Mem: 0.5},
		100: {PID: 100, PPID: 1, CPU: 0.5, Mem: 0.2},
	}

	tests := []struct {
		name    string
		root    int
		wantCPU float64
		wantMem float64
	}{
		{
			name:    "subtree from PID 10",
			root:    10,
			wantCPU: 6.5, // 3.0 + 2.0 + 1.5
			wantMem: 3.5, // 2.0 + 1.0 + 0.5
		},
		{
			name:    "leaf node",
			root:    20,
			wantCPU: 2.0,
			wantMem: 1.0,
		},
		{
			name:    "missing PID",
			root:    999,
			wantCPU: 0.0,
			wantMem: 0.0,
		},
		{
			name:    "full tree from root",
			root:    1,
			wantCPU: 8.0,
			wantMem: 4.2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpu, mem := AggregateResources(tt.root, table)
			if cpu != tt.wantCPU {
				t.Errorf("CPU = %f, want %f", cpu, tt.wantCPU)
			}
			if mem != tt.wantMem {
				t.Errorf("Mem = %f, want %f", mem, tt.wantMem)
			}
		})
	}
}
