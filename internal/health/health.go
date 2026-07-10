package health

import (
	"fmt"
	"github.com/pol-cova/observe/internal/metrics/local"
)

func Hints(s local.Snapshot) []string {
	var h []string
	if s.CPU >= 90 {
		h = append(h, "CPU saturated — reduce concurrency or profile the busiest process")
	}
	if s.Memory >= 85 {
		h = append(h, "Memory usage above 85% — check for leaks or lower worker counts")
	}
	if s.Disk >= 90 {
		h = append(h, "Disk almost full — free space before the test becomes unreliable")
	}
	if s.NetErrors > 0 {
		h = append(h, fmt.Sprintf("Network errors detected (%d total)", s.NetErrors))
	}
	if len(h) == 0 {
		h = append(h, "No immediate resource bottleneck detected")
	}
	return h
}
