package health

import (
	"fmt"

	"github.com/pol-cova/observe/internal/metrics/local"
)

func Hints(s local.Snapshot) []string {
	return HintsWithThresholds([]local.Snapshot{s}, Thresholds{})
}

type Thresholds struct{ CPU, Memory, Swap, Disk, IOWait float64 }

func HintsWithHistory(history []local.Snapshot) []string {
	return HintsWithThresholds(history, Thresholds{})
}

func HintsWithThresholds(history []local.Snapshot, thresholds Thresholds) []string {
	if len(history) == 0 {
		return []string{"No immediate resource bottleneck detected"}
	}
	s := history[len(history)-1]
	if thresholds.CPU <= 0 {
		thresholds.CPU = 90
	}
	if thresholds.Memory <= 0 {
		thresholds.Memory = 85
	}
	if thresholds.Swap <= 0 {
		thresholds.Swap = 25
	}
	if thresholds.Disk <= 0 {
		thresholds.Disk = 90
	}
	if thresholds.IOWait <= 0 {
		thresholds.IOWait = 20
	}
	var h []string
	if s.CPU >= thresholds.CPU {
		h = append(h, "CPU saturated — reduce concurrency or profile the busiest process")
	}
	if s.Memory >= thresholds.Memory {
		h = append(h, fmt.Sprintf("Memory usage above %.0f%% — check for leaks or lower worker counts", thresholds.Memory))
	}
	if s.Swap >= thresholds.Swap {
		h = append(h, fmt.Sprintf("Swap usage is %.0f%% — memory pressure may be affecting responsiveness", s.Swap))
	}
	if s.Disk >= thresholds.Disk {
		h = append(h, fmt.Sprintf("Disk almost full (above %.0f%%) — free space before it affects the system", thresholds.Disk))
	}
	if s.IOWait >= thresholds.IOWait {
		h = append(h, fmt.Sprintf("I/O wait is %.0f%% — storage may be the bottleneck", s.IOWait))
	}
	if s.NetErrors > 0 {
		h = append(h, fmt.Sprintf("Network errors detected (%d total)", s.NetErrors))
	}
	if len(history) >= 3 {
		first, last := history[0], history[len(history)-1]
		if last.Memory-first.Memory >= 10 {
			h = append(h, fmt.Sprintf("Memory is rising %.1f points across the sample window — check for growth or leaks", last.Memory-first.Memory))
		}
		if last.CPU-first.CPU >= 20 && last.CPU >= thresholds.CPU {
			h = append(h, "CPU pressure is rising with the trend — reduce concurrency before latency worsens")
		}
		if last.NetErrors > first.NetErrors {
			h = append(h, "Network errors are increasing across the sample window")
		}
	}
	if len(h) == 0 {
		h = append(h, "No immediate resource bottleneck detected")
	}
	return h
}
