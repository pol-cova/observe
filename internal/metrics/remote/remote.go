package remote

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/pol-cova/observe/internal/metrics/local"
)

// Collector uses the existing observe binary on the remote host, so SSH mode
// requires no daemon or agent and returns the same snapshot schema as local mode.
type Collector struct{ Target string }

func New(target string) *Collector { return &Collector{Target: target} }

func (c *Collector) Collect() (local.Snapshot, error) {
	var output bytes.Buffer
	cmd := exec.Command("ssh", c.Target, "observe", "snapshot", "--output", "-")
	cmd.Stdout, cmd.Stderr = &output, &output
	if err := cmd.Run(); err != nil {
		return local.Snapshot{}, fmt.Errorf("ssh %s: %w", c.Target, err)
	}
	var report struct {
		Metrics local.Snapshot `json:"metrics"`
	}
	if err := json.Unmarshal(output.Bytes(), &report); err != nil {
		return local.Snapshot{}, fmt.Errorf("decode remote snapshot: %w", err)
	}
	return report.Metrics, nil
}
