package health

import (
	"strings"
	"testing"

	"github.com/pol-cova/observe/internal/metrics/local"
)

func TestHintsReturnsHealthyMessage(t *testing.T) {
	hints := Hints(local.Snapshot{CPU: 40, Memory: 50, Disk: 60})
	if len(hints) != 1 || hints[0] != HealthyMessage {
		t.Fatalf("unexpected healthy hints: %#v", hints)
	}
}

func TestHintsReportsResourcePressure(t *testing.T) {
	hints := Hints(local.Snapshot{CPU: 95, Memory: 90, Swap: 30, Disk: 95, IOWait: 25, NetErrors: 3})
	joined := strings.Join(hints, "\n")
	for _, phrase := range []string{"CPU saturated", "Memory usage", "Swap usage", "Disk almost full", "I/O wait", "Network errors"} {
		if !strings.Contains(joined, phrase) {
			t.Errorf("missing %q in hints: %s", phrase, joined)
		}
	}
}
