package assistant

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pol-cova/observe/internal/health"
	"github.com/pol-cova/observe/internal/metrics/local"
)

func Answer(parts []string, history []local.Snapshot, thresholds health.Thresholds) string {
	if len(history) == 0 {
		return "No samples collected yet.\n"
	}
	latest := history[len(history)-1]
	q := strings.ToLower(strings.Join(parts, " "))

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", titleFor(q))
	fmt.Fprintf(&b, "CPU %.1f%%, memory %.1f%%, disk %.1f%%.\n", latest.CPU, latest.Memory, latest.Disk)
	if len(history) > 1 {
		first := history[0]
		fmt.Fprintf(&b, "Trend (%d samples): CPU %+.1f, memory %+.1f.\n",
			len(history), latest.CPU-first.CPU, latest.Memory-first.Memory)
	}
	for _, process := range topProcesses(latest.Processes, 3) {
		fmt.Fprintf(&b, "  • %s (pid %d) — %.1f%% CPU\n", process.Name, process.PID, process.CPU)
	}
	for _, hint := range health.HintsWithThresholds(history, thresholds) {
		fmt.Fprintf(&b, "  • %s\n", hint)
	}
	fmt.Fprintf(&b, "\n%s\n", adviceFor(q, latest))
	return b.String()
}

func titleFor(q string) string {
	switch {
	case strings.Contains(q, "prometheus") || strings.Contains(q, "5xx"):
		return "Prometheus diagnosis"
	case strings.Contains(q, "latency"):
		return "Latency diagnosis"
	case strings.Contains(q, "memory") || strings.Contains(q, "ram"):
		return "Memory diagnosis"
	case strings.Contains(q, "disk"):
		return "Disk diagnosis"
	case strings.Contains(q, "network"):
		return "Network diagnosis"
	case strings.Contains(q, "cpu"):
		return "CPU diagnosis"
	default:
		return "Current system health"
	}
}

func adviceFor(q string, s local.Snapshot) string {
	switch {
	case strings.Contains(q, "prometheus") || strings.Contains(q, "5xx"):
		return "For 5xx errors, use:\n\n  sum(rate(http_requests_total{status=~\"5..\"}[1m]))"
	case strings.Contains(q, "cpu") && s.CPU >= 90:
		return "This machine appears CPU-bound. Profile the top process or reduce concurrency."
	case strings.Contains(q, "memory") && (s.Memory >= 85 || s.Swap >= 25):
		return "Memory pressure is the strongest signal. Check process growth and swapping."
	case strings.Contains(q, "disk") && s.Disk >= 90:
		return "Disk capacity is critically high; free space before trusting results."
	default:
		return "No clear local bottleneck in these samples. Compare metrics over time or run with --prometheus."
	}
}

func topProcesses(processes []local.Process, limit int) []local.Process {
	if limit <= 0 || len(processes) == 0 {
		return nil
	}
	ranked := append([]local.Process(nil), processes...)
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].CPU > ranked[j].CPU })
	if len(ranked) > limit {
		return ranked[:limit]
	}
	return ranked
}
