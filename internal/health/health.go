package health

import (
	"fmt"

	"github.com/pol-cova/observe/internal/config"
	"github.com/pol-cova/observe/internal/metrics/local"
)

type Thresholds = config.Thresholds

const HealthyMessage = "No immediate resource bottleneck detected"

func ApplyDefaults(thresholds Thresholds) Thresholds {
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
	return thresholds
}

func Hints(snapshot local.Snapshot) []string {
	return HintsWithThresholds([]local.Snapshot{snapshot}, Thresholds{})
}

func HintsWithThresholds(history []local.Snapshot, thresholds Thresholds) []string {
	if len(history) == 0 {
		return []string{HealthyMessage}
	}
	thresholds = ApplyDefaults(thresholds)
	s := history[len(history)-1]
	var hints []string
	if s.CPU >= thresholds.CPU {
		hints = append(hints, "CPU saturated — reduce concurrency or profile the busiest process")
	}
	if s.Memory >= thresholds.Memory {
		hints = append(hints, fmt.Sprintf("Memory usage above %.0f%% — check for leaks or lower worker counts", thresholds.Memory))
	}
	if s.Swap >= thresholds.Swap {
		hints = append(hints, fmt.Sprintf("Swap usage is %.0f%% — memory pressure may be affecting responsiveness", s.Swap))
	}
	if s.Disk >= thresholds.Disk {
		hints = append(hints, fmt.Sprintf("Disk almost full (above %.0f%%) — free space before it affects the system", thresholds.Disk))
	}
	if s.IOWait >= thresholds.IOWait {
		hints = append(hints, fmt.Sprintf("I/O wait is %.0f%% — storage may be the bottleneck", s.IOWait))
	}
	if s.NetErrors > 0 {
		hints = append(hints, fmt.Sprintf("Network errors detected (%d total)", s.NetErrors))
	}
	if len(history) >= 3 {
		first, last := history[0], history[len(history)-1]
		if last.Memory-first.Memory >= 10 {
			hints = append(hints, fmt.Sprintf("Memory is rising %.1f points across the sample window — check for growth or leaks", last.Memory-first.Memory))
		}
		if last.CPU-first.CPU >= 20 && last.CPU >= thresholds.CPU {
			hints = append(hints, "CPU pressure is rising with the trend — reduce concurrency before latency worsens")
		}
		if last.NetErrors > first.NetErrors {
			hints = append(hints, "Network errors are increasing across the sample window")
		}
	}
	if len(hints) == 0 {
		hints = append(hints, HealthyMessage)
	}
	return hints
}
