package local

import (
	"os"
	"testing"

	psnet "github.com/shirou/gopsutil/v4/net"
)

func TestFormatConnection(t *testing.T) {
	connection := formatConnection(psnet.ConnectionStat{
		Type:   1,
		Status: "LISTEN",
		Laddr:  psnet.Addr{IP: "127.0.0.1", Port: 8080},
	})
	if connection.Type != "tcp" || connection.Local != "127.0.0.1:8080" || connection.Remote != "-" || connection.Status != "LISTEN" {
		t.Fatalf("unexpected connection: %#v", connection)
	}
}

func TestAddress(t *testing.T) {
	if got := address("", 0); got != "-" {
		t.Fatalf("address = %q, want -", got)
	}
}

func TestInspectCurrentProcess(t *testing.T) {
	details, err := Inspect(int32(os.Getpid()))
	if err != nil {
		t.Fatal(err)
	}
	if details.PID != int32(os.Getpid()) || details.Name == "" {
		t.Fatalf("unexpected process details: %#v", details.Process)
	}
}
