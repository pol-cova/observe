package docker

import "testing"

func TestParseStats(t *testing.T) {
	got := parse(cliContainer{ID: "abc", Name: "web", CPUPerc: "12.5%", MemPerc: "40%", NetIO: "1.0MB / 2.0KB", Ports: "0.0.0.0:80->80/tcp", Status: "Up 2 minutes"})
	if got.CPU != 12.5 || got.Memory != 40 || got.NetworkIn != 1048576 || got.NetworkOut != 2048 || got.Name != "web" {
		t.Fatalf("unexpected container: %#v", got)
	}
}
