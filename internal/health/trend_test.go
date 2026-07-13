package health

import (
	"github.com/pol-cova/observe/internal/metrics/local"
	"strings"
	"testing"
)

func TestHintsDetectsTrends(t *testing.T) {
	hints := HintsWithHistory([]local.Snapshot{{Memory: 40, CPU: 30}, {Memory: 45, CPU: 50}, {Memory: 55, CPU: 95}})
	joined := strings.Join(hints, "\n")
	if !strings.Contains(joined, "Memory is rising") || !strings.Contains(joined, "CPU pressure is rising") {
		t.Fatalf("missing trend hints: %s", joined)
	}
}
