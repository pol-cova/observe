package prometheus

import "testing"

func TestPresetByName(t *testing.T) {
	preset, ok := PresetByName("5xx errors")
	if !ok {
		t.Fatal("expected preset to exist")
	}
	if preset.Query == "" {
		t.Fatal("expected preset query")
	}
	if _, ok := PresetByName("missing"); ok {
		t.Fatal("unexpected preset match")
	}
}
