package metrics

import "testing"

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input uint64
		want  string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{1048576, "1.0 MiB"},
		{1073741824, "1.0 GiB"},
		{1099511627776, "1.0 TiB"},
		{5368709120, "5.0 GiB"},
	}

	for _, tt := range tests {
		got := FormatBytes(tt.input)
		if got != tt.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatRate(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0, "0 B/s"},
		{500, "500 B/s"},
		{1024, "1.0 KiB/s"},
		{1048576, "1.0 MiB/s"},
		{1073741824, "1.0 GiB/s"},
		{1536, "1.5 KiB/s"},
	}

	for _, tt := range tests {
		got := FormatRate(tt.input)
		if got != tt.want {
			t.Errorf("FormatRate(%f) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
