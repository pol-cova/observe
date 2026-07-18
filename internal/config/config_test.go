package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "observe.yaml")
	if err := os.WriteFile(path, []byte("refresh_interval: 750\ntheme: light\nthresholds:\n  cpu: 80\npanels: [overview, docker]\nprometheus_presets:\n  - name: Queue depth\n    query: queue_depth\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RefreshInterval != 750 || cfg.Thresholds.CPU != 80 || len(cfg.Panels) != 2 || cfg.Theme != "light" {
		t.Fatalf("unexpected config: %#v", cfg)
	}
	if len(cfg.Prometheus) != 1 || cfg.Prometheus[0].Name != "Queue depth" {
		t.Fatalf("unexpected prometheus presets: %#v", cfg.Prometheus)
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
