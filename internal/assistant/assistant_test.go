package assistant

import (
	"strings"
	"testing"

	"github.com/pol-cova/observe/internal/config"
	"github.com/pol-cova/observe/internal/metrics/local"
)

func TestAnswerUsesHealthHints(t *testing.T) {
	answer := Answer([]string{"cpu"}, local.Snapshot{CPU: 95, Memory: 40, Disk: 10}, config.Thresholds{})
	if !strings.Contains(answer, "CPU saturated") {
		t.Fatalf("expected CPU hint, got %q", answer)
	}
}

func TestAnswerHealthyIncludesLatencyTip(t *testing.T) {
	answer := Answer([]string{"status"}, local.Snapshot{CPU: 10, Memory: 20, Disk: 30}, config.Thresholds{})
	if !strings.Contains(answer, "No clear local resource bottleneck") {
		t.Fatalf("expected healthy guidance, got %q", answer)
	}
}

func TestAnswerUsesPrometheusPreset(t *testing.T) {
	answer := Answer([]string{"prometheus", "5xx"}, local.Snapshot{}, config.Thresholds{})
	if !strings.Contains(answer, `status=~"5.."`) {
		t.Fatalf("expected prometheus preset query, got %q", answer)
	}
}

func TestAnswerRespectsConfigThresholds(t *testing.T) {
	answer := Answer([]string{"cpu"}, local.Snapshot{CPU: 85, Memory: 40, Disk: 10}, config.Thresholds{CPU: 80})
	if !strings.Contains(answer, "CPU saturated") {
		t.Fatalf("expected CPU hint at 85 with threshold 80, got %q", answer)
	}
}
