package prometheus

import (
	"testing"

	"github.com/pol-cova/observe/internal/config"
)

func TestMergePresetsUsesBuiltinsWhenEmpty(t *testing.T) {
	merged := MergePresets(nil)
	if len(merged) != len(Presets) {
		t.Fatalf("len(MergePresets(nil)) = %d, want %d", len(merged), len(Presets))
	}
}

func TestMergePresetsOverridesAndAddsCustom(t *testing.T) {
	custom := []config.PrometheusPreset{
		{Name: "5xx errors", Description: "custom 5xx", Query: "sum(rate(errors_total[1m]))"},
		{Name: "Queue depth", Description: "pending jobs", Query: "queue_depth"},
	}
	merged := MergePresets(custom)
	foundOverride, foundCustom := false, false
	for _, preset := range merged {
		if preset.Name == "5xx errors" {
			foundOverride = true
			if preset.Query != "sum(rate(errors_total[1m]))" {
				t.Fatalf("override query = %q", preset.Query)
			}
		}
		if preset.Name == "Queue depth" {
			foundCustom = true
		}
	}
	if !foundOverride || !foundCustom {
		t.Fatalf("merged presets missing override/custom: %#v", merged)
	}
}
