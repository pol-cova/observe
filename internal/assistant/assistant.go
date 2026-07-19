package assistant

import (
	"fmt"
	"strings"

	"github.com/pol-cova/observe/internal/config"
	"github.com/pol-cova/observe/internal/health"
	"github.com/pol-cova/observe/internal/metrics/local"
	"github.com/pol-cova/observe/internal/prometheus"
)

func Answer(parts []string, snapshot local.Snapshot, thresholds config.Thresholds) string {
	question := strings.ToLower(strings.Join(parts, " "))
	var answer strings.Builder
	switch {
	case strings.Contains(question, "prometheus") || strings.Contains(question, "5xx") || strings.Contains(question, "query"):
		if preset, ok := prometheus.PresetByName("5xx errors"); ok {
			fmt.Fprintf(&answer, "For 5xx errors, use:\n\n  %s\n\n", preset.Query)
		} else {
			answer.WriteString("Prometheus query help\n\n")
		}
	case strings.Contains(question, "latency"):
		answer.WriteString("Latency diagnosis\n\n")
	case strings.Contains(question, "cpu"):
		answer.WriteString("CPU diagnosis\n\n")
	default:
		answer.WriteString("Current system health\n\n")
	}

	fmt.Fprintf(&answer, "CPU is %.1f%%, memory is %.1f%%, and disk is %.1f%%.\n", snapshot.CPU, snapshot.Memory, snapshot.Disk)

	hints := health.HintsWithThresholds([]local.Snapshot{snapshot}, thresholds)
	hasIssue := false
	for _, hint := range hints {
		if hint == health.HealthyMessage {
			continue
		}
		hasIssue = true
		answer.WriteString("\n")
		answer.WriteString(hint)
		answer.WriteByte('\n')
	}
	if !hasIssue {
		answer.WriteString("\nNo clear local resource bottleneck is present in this single sample. For latency, compare p95 with CPU and error rate over time.\n")
	}
	return answer.String()
}
