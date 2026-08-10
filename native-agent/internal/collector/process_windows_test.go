//go:build windows

package collector

import "testing"

func TestReadProcessesReturnsCurrentInventory(t *testing.T) {
	processes := readProcesses()
	if len(processes) == 0 {
		t.Fatal("expected at least one running process")
	}
	for _, process := range processes {
		if process.PID < 0 || process.Name == "" {
			t.Fatalf("invalid process entry: %+v", process)
		}
	}
}
