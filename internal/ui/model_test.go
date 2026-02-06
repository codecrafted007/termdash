package ui

import "testing"

func TestHistoryPushAndValues(t *testing.T) {
	h := NewHistory(5)

	// Push fewer than capacity.
	h.Push(1.0)
	h.Push(2.0)
	h.Push(3.0)

	vals := h.Values()
	if len(vals) != 3 {
		t.Fatalf("expected 3 values, got %d", len(vals))
	}
	if vals[0] != 1.0 || vals[1] != 2.0 || vals[2] != 3.0 {
		t.Errorf("unexpected values: %v", vals)
	}

	// Push beyond capacity — oldest should be evicted.
	h.Push(4.0)
	h.Push(5.0)
	h.Push(6.0)

	vals = h.Values()
	if len(vals) != 5 {
		t.Fatalf("expected 5 values, got %d", len(vals))
	}
	if vals[0] != 2.0 {
		t.Errorf("expected oldest value 2.0, got %f", vals[0])
	}
	if vals[4] != 6.0 {
		t.Errorf("expected newest value 6.0, got %f", vals[4])
	}
}

func TestHistoryValuesReturnsCopy(t *testing.T) {
	h := NewHistory(3)
	h.Push(10.0)

	vals := h.Values()
	vals[0] = 999.0

	original := h.Values()
	if original[0] != 10.0 {
		t.Errorf("Values() should return a copy, but mutation was reflected")
	}
}

func TestNewModel(t *testing.T) {
	m := New()
	if m.collector == nil {
		t.Error("expected non-nil collector")
	}
	if m.cpuHistory == nil || m.sendHistory == nil || m.recvHistory == nil {
		t.Error("expected non-nil history buffers")
	}
	if m.width != 80 || m.height != 24 {
		t.Error("expected default dimensions 80x24")
	}
}
