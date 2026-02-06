package metrics

import (
	"os"
	"runtime"
	"testing"
)

func TestCollectorCollect(t *testing.T) {
	c := NewCollector()

	snap, err := c.Collect()
	if err != nil {
		t.Fatalf("first Collect() error: %v", err)
	}

	if snap.Hostname == "" {
		t.Error("expected non-empty hostname")
	}
	if snap.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
	if snap.Memory.TotalRAM == 0 {
		t.Error("expected non-zero total RAM")
	}
	if len(snap.CPU.PerCore) == 0 {
		t.Error("expected at least one CPU core")
	}

	// First call may have zero network rates (no previous data).
	if snap.Network.SendRate != 0 || snap.Network.RecvRate != 0 {
		t.Error("expected zero rates on first collection")
	}

	// Second call should have valid rates (possibly still zero if no traffic).
	snap2, err := c.Collect()
	if err != nil {
		t.Fatalf("second Collect() error: %v", err)
	}
	if snap2.Network.SendRate < 0 || snap2.Network.RecvRate < 0 {
		t.Error("expected non-negative network rates")
	}
}

func TestProcessCollectorReturnsAllProcesses(t *testing.T) {
	pc := NewProcessCollector()

	procs, err := pc.CollectAll()
	if err != nil {
		t.Fatalf("CollectAll() error: %v", err)
	}

	// On any OS there should be more than 25 processes running.
	// This verifies the old topN=25 truncation is gone.
	minExpected := 10
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		minExpected = 25
	}
	if len(procs) < minExpected {
		t.Errorf("expected at least %d processes, got %d", minExpected, len(procs))
	}

	// Our own process must be in the list.
	myPID := int32(os.Getpid())
	found := false
	for _, p := range procs {
		if p.PID == myPID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("own PID %d not found in process list", myPID)
	}
}

func TestProcessCollectorCPUNonNegative(t *testing.T) {
	pc := NewProcessCollector()

	// First collection establishes the CPU baseline (values will be 0).
	_, err := pc.CollectAll()
	if err != nil {
		t.Fatalf("first CollectAll() error: %v", err)
	}

	// Second collection should produce non-negative CPU% values.
	procs, err := pc.CollectAll()
	if err != nil {
		t.Fatalf("second CollectAll() error: %v", err)
	}

	for _, p := range procs {
		if p.CPUPercent < 0 {
			t.Errorf("process %d (%s) has negative CPU%%: %f", p.PID, p.Name, p.CPUPercent)
		}
	}
}
