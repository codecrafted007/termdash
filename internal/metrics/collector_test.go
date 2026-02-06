package metrics

import "testing"

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
