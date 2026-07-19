package assistant

import (
	"strings"
	"testing"

	"github.com/pol-cova/observe/internal/health"
	"github.com/pol-cova/observe/internal/metrics/local"
)

func TestAnswerIncludesTrendAndHints(t *testing.T) {
	history := []local.Snapshot{
		{CPU: 40, Memory: 50, Disk: 60, Processes: []local.Process{{PID: 1, Name: "worker", CPU: 12}}},
		{CPU: 55, Memory: 62, Disk: 61, Processes: []local.Process{{PID: 1, Name: "worker", CPU: 18}}},
	}
	answer := Answer([]string{"cpu"}, history, health.Thresholds{})
	for _, phrase := range []string{"Trend (2 samples)", "worker", "No immediate resource bottleneck detected"} {
		if !strings.Contains(answer, phrase) {
			t.Fatalf("answer missing %q:\n%s", phrase, answer)
		}
	}
}

func TestAnswerCPUAdviceWhenSaturated(t *testing.T) {
	answer := Answer([]string{"cpu"}, []local.Snapshot{{CPU: 95, Memory: 40, Disk: 50}}, health.Thresholds{})
	if !strings.Contains(answer, "CPU-bound") {
		t.Fatalf("expected CPU-bound advice: %s", answer)
	}
}

func TestAnswerUsesThresholdHints(t *testing.T) {
	answer := Answer([]string{"health"}, []local.Snapshot{{CPU: 96, Memory: 40, Disk: 50}}, health.Thresholds{CPU: 95})
	if !strings.Contains(answer, "CPU saturated") {
		t.Fatalf("expected threshold hint: %s", answer)
	}
}
