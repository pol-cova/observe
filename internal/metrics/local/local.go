package local

import (
	"fmt"
	"sort"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

type Process struct {
	PID         int32
	Name        string
	CPU, Memory float64
}
type Snapshot struct {
	CPU, Memory, Disk float64
	DiskPath          string
	NetIn, NetOut     float64
	NetErrors         uint64
	Processes         []Process
	Ports             []uint32
	At                time.Time
}
type Collector struct {
	last   net.IOCountersStat
	lastAt time.Time
}

func New() *Collector { return &Collector{} }

func (c *Collector) Collect() (Snapshot, error) {
	result := Snapshot{At: time.Now()}
	var err error
	if values, e := cpu.Percent(0, false); e == nil && len(values) > 0 {
		result.CPU = values[0]
	} else {
		err = e
	}
	if values, e := mem.VirtualMemory(); e == nil {
		result.Memory = values.UsedPercent
	} else if err == nil {
		err = e
	}
	path := "/"
	if info, e := host.Info(); e == nil && info.OS == "windows" {
		path = "C:"
	}
	if values, e := disk.Usage(path); e == nil {
		result.Disk = values.UsedPercent
		result.DiskPath = path
	}
	if values, e := net.IOCounters(false); e == nil && len(values) > 0 {
		now := time.Now()
		if !c.lastAt.IsZero() {
			secs := now.Sub(c.lastAt).Seconds()
			result.NetIn = float64(values[0].BytesRecv-c.last.BytesRecv) / secs
			result.NetOut = float64(values[0].BytesSent-c.last.BytesSent) / secs
		}
		result.NetErrors = values[0].Errin + values[0].Errout
		c.last = values[0]
		c.lastAt = now
	}
	result.Processes = topProcesses()
	if conns, e := net.Connections("tcp"); e == nil {
		seen := map[uint32]bool{}
		for _, conn := range conns {
			if conn.Status == "LISTEN" && conn.Laddr.Port > 0 {
				seen[conn.Laddr.Port] = true
			}
		}
		for p := range seen {
			result.Ports = append(result.Ports, p)
		}
		sort.Slice(result.Ports, func(i, j int) bool { return result.Ports[i] < result.Ports[j] })
	}
	return result, err
}
func topProcesses() []Process {
	ps, err := process.Processes()
	if err != nil {
		return nil
	}
	out := make([]Process, 0, len(ps))
	for _, p := range ps {
		n, _ := p.Name()
		cpu, _ := p.CPUPercent()
		m, _ := p.MemoryPercent()
		out = append(out, Process{p.Pid, n, cpu, float64(m)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CPU > out[j].CPU })
	if len(out) > 6 {
		out = out[:6]
	}
	return out
}
func FormatRate(bytes float64) string {
	units := []string{"B/s", "KB/s", "MB/s", "GB/s"}
	i := 0
	for bytes >= 1024 && i < len(units)-1 {
		bytes /= 1024
		i++
	}
	return fmt.Sprintf("%.1f %s", bytes, units[i])
}
