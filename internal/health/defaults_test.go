package health

import "testing"

func TestApplyDefaults(t *testing.T) {
	defaults := ApplyDefaults(Thresholds{})
	if defaults.CPU != 90 || defaults.Memory != 85 || defaults.Swap != 25 || defaults.Disk != 90 || defaults.IOWait != 20 {
		t.Fatalf("unexpected defaults: %#v", defaults)
	}
}

func TestApplyDefaultsPreservesConfiguredValues(t *testing.T) {
	defaults := ApplyDefaults(Thresholds{CPU: 70, Memory: 60})
	if defaults.CPU != 70 || defaults.Memory != 60 || defaults.Swap != 25 {
		t.Fatalf("unexpected thresholds: %#v", defaults)
	}
}
