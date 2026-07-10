package assistant

import (
	"fmt"
	"github.com/pol-cova/observe/internal/metrics/local"
	"strings"
)

func Answer(parts []string, s local.Snapshot) string {
	q := strings.ToLower(strings.Join(parts, " "))
	var b strings.Builder
	switch {
	case strings.Contains(q, "prometheus") || strings.Contains(q, "5xx") || strings.Contains(q, "query"):
		b.WriteString("For 5xx errors, use:\n\n  sum(rate(http_requests_total{status=~\"5..\"}[1m]))\n\n")
	case strings.Contains(q, "latency"):
		b.WriteString("Latency diagnosis\n\n")
	case strings.Contains(q, "cpu"):
		b.WriteString("CPU diagnosis\n\n")
	default:
		b.WriteString("Current server health\n\n")
	}
	fmt.Fprintf(&b, "CPU is %.1f%%, memory is %.1f%%, and disk is %.1f%%.\n", s.CPU, s.Memory, s.Disk)
	if s.CPU >= 90 {
		b.WriteString("\nThis machine appears CPU-bound. Profile the top process or reduce test concurrency.\n")
	} else if s.Memory >= 85 {
		b.WriteString("\nMemory pressure is the strongest signal. Check process growth and swapping.\n")
	} else if s.Disk >= 90 {
		b.WriteString("\nDisk capacity is critically high; free space before trusting results.\n")
	} else {
		b.WriteString("\nNo clear local resource bottleneck is present in this single sample. For latency, compare p95 with CPU and error rate over time.\n")
	}
	return b.String()
}
