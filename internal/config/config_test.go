package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "observe.yaml")
	if err := os.WriteFile(path, []byte("refresh_interval: 750\nthresholds:\n  cpu: 80\npanels: [processes, docker]\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RefreshInterval != 750 || cfg.Thresholds.CPU != 80 || len(cfg.Panels) != 2 {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}

func TestMissingConfigUsesDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RefreshInterval != 0 {
		t.Fatalf("missing config should preserve zero-value defaults: %#v", cfg)
	}
}
