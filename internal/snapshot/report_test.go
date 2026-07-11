package snapshot

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/pol-cova/observe/internal/metrics/local"
)

func TestWriteProducesReadableJSON(t *testing.T) {
	report := Report{GeneratedAt: time.Unix(1, 0), Machine: Machine{Hostname: "host"}, Metrics: local.Snapshot{CPU: 42}, Hints: []string{"all clear"}}
	var output bytes.Buffer
	if err := Write(&output, report); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"\"generated_at\"", "\"hostname\": \"host\"", "\"cpu_percent\": 42", "all clear"} {
		if !strings.Contains(output.String(), value) {
			t.Errorf("snapshot does not include %q: %s", value, output.String())
		}
	}
}
