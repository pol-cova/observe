package local

import "testing"

func TestFormatRate(t *testing.T) {
	tests := map[float64]string{
		0:          "0.0 B/s",
		1024:       "1.0 KB/s",
		1048576:    "1.0 MB/s",
		1073741824: "1.0 GB/s",
	}
	for input, want := range tests {
		if got := FormatRate(input); got != want {
			t.Errorf("FormatRate(%v) = %q, want %q", input, got, want)
		}
	}
}

func TestRate(t *testing.T) {
	if got := rate(300, 100, 2); got != 100 {
		t.Fatalf("rate = %v, want 100", got)
	}
	if got := rate(100, 300, 2); got != 0 {
		t.Fatalf("rate reset = %v, want 0", got)
	}
}
