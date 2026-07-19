package docker

import "testing"

func TestParseStats(t *testing.T) {
	got := parse(cliContainer{ID: "abc", Name: "web", CPUPerc: "12.5%", MemPerc: "40%", NetIO: "1.0MB / 2.0KB", Ports: "0.0.0.0:80->80/tcp", Status: "Up 2 minutes"})
	if got.CPU != 12.5 || got.Memory != 40 || got.NetworkIn != 1048576 || got.NetworkOut != 2048 || got.Name != "web" {
		t.Fatalf("unexpected container: %#v", got)
	}
}

func TestPercentParsesDockerOutput(t *testing.T) {
	if got := percent("  8.25% "); got != 8.25 {
		t.Fatalf("percent() = %v, want 8.25", got)
	}
}

func TestBytesValueHandlesUnits(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"1.5GB", 1.5 * 1024 * 1024 * 1024},
		{"512KB", 512 * 1024},
		{"42", 42},
	}
	for _, test := range tests {
		if got := bytesValue(test.input); got != test.want {
			t.Fatalf("bytesValue(%q) = %v, want %v", test.input, got, test.want)
		}
	}
}
