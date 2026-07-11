package local

import (
	"fmt"
	"sort"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

type Process struct {
	PID    int32   `json:"pid"`
	Name   string  `json:"name"`
	CPU    float64 `json:"cpu_percent"`
	Memory float64 `json:"memory_percent"`
}
type Snapshot struct {
	CPU       float64   `json:"cpu_percent"`
	IOWait    float64   `json:"io_wait_percent"`
	Memory    float64   `json:"memory_percent"`
	Swap      float64   `json:"swap_percent"`
	Disk      float64   `json:"disk_percent"`
	Load1     float64   `json:"load_1"`
	Load5     float64   `json:"load_5"`
	Load15    float64   `json:"load_15"`
	DiskPath  string    `json:"disk_path"`
	NetIn     float64   `json:"network_in_bytes_per_second"`
	NetOut    float64   `json:"network_out_bytes_per_second"`
	DiskRead  float64   `json:"disk_read_bytes_per_second"`
	DiskWrite float64   `json:"disk_write_bytes_per_second"`
	NetErrors uint64    `json:"network_errors"`
	Processes []Process `json:"processes"`
	Ports     []uint32  `json:"listening_ports"`
	At        time.Time `json:"sampled_at"`
}
type Collector struct {
	lastNet    net.IOCountersStat
	lastDiskIO disk.IOCountersStat
	lastAt     time.Time
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
	if times, e := cpu.Times(false); e == nil && len(times) > 0 {
		total := times[0].Total()
		if total > 0 {
			result.IOWait = times[0].Iowait / total * 100
		}
	}
	if values, e := mem.VirtualMemory(); e == nil {
		result.Memory = values.UsedPercent
	} else if err == nil {
		err = e
	}
	if values, e := mem.SwapMemory(); e == nil {
		result.Swap = values.UsedPercent
	}
	if values, e := load.Avg(); e == nil {
		result.Load1, result.Load5, result.Load15 = values.Load1, values.Load5, values.Load15
	}
	path := "/"
	if info, e := host.Info(); e == nil && info.OS == "windows" {
		path = "C:"
	}
	if values, e := disk.Usage(path); e == nil {
		result.Disk = values.UsedPercent
		result.DiskPath = path
	}
	now := time.Now()
	if values, e := net.IOCounters(false); e == nil && len(values) > 0 {
		if !c.lastAt.IsZero() {
			secs := now.Sub(c.lastAt).Seconds()
			result.NetIn = rate(values[0].BytesRecv, c.lastNet.BytesRecv, secs)
			result.NetOut = rate(values[0].BytesSent, c.lastNet.BytesSent, secs)
		}
		result.NetErrors = values[0].Errin + values[0].Errout
		c.lastNet = values[0]
	}
	if counters, e := disk.IOCounters(); e == nil {
		var total disk.IOCountersStat
		for _, counter := range counters {
			total.ReadBytes += counter.ReadBytes
			total.WriteBytes += counter.WriteBytes
		}
		if !c.lastAt.IsZero() {
			secs := now.Sub(c.lastAt).Seconds()
			result.DiskRead = rate(total.ReadBytes, c.lastDiskIO.ReadBytes, secs)
			result.DiskWrite = rate(total.WriteBytes, c.lastDiskIO.WriteBytes, secs)
		}
		c.lastDiskIO = total
	}
	c.lastAt = now
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

func rate(current, previous uint64, seconds float64) float64 {
	if seconds <= 0 || current < previous {
		return 0
	}
	return float64(current-previous) / seconds
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
