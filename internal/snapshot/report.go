package snapshot

import (
	"encoding/json"
	"io"
	"runtime"
	"time"

	"github.com/pol-cova/observe/internal/health"
	"github.com/pol-cova/observe/internal/metrics/local"
	"github.com/shirou/gopsutil/v4/host"
)

type Report struct {
	GeneratedAt time.Time      `json:"generated_at"`
	Machine     Machine        `json:"machine"`
	Metrics     local.Snapshot `json:"metrics"`
	Hints       []string       `json:"hints"`
}

type Machine struct {
	Hostname     string `json:"hostname"`
	OS           string `json:"os"`
	Platform     string `json:"platform"`
	Architecture string `json:"architecture"`
	Uptime       uint64 `json:"uptime_seconds"`
}

func New(metrics local.Snapshot) Report {
	info, _ := host.Info()
	machine := Machine{Architecture: runtime.GOARCH}
	if info != nil {
		machine.Hostname = info.Hostname
		machine.OS = info.OS
		machine.Platform = info.Platform
		machine.Uptime = info.Uptime
	}
	return Report{
		GeneratedAt: time.Now(),
		Machine:     machine,
		Metrics:     metrics,
		Hints:       health.Hints(metrics),
	}
}

func Write(writer io.Writer, report Report) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
