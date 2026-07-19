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
	if collectErr := collectCPUAndMemory(&result); collectErr != nil && err == nil {
		err = collectErr
	}
	collectLoadAndDisk(&result)
	now := time.Now()
	collectNetworkRates(c, now, &result)
	collectDiskIORates(c, now, &result)
	c.lastAt = now
	result.Processes = topProcesses()
	result.Ports = listeningPorts()
	return result, err
}

func collectCPUAndMemory(result *Snapshot) error {
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
	return err
}

func collectLoadAndDisk(result *Snapshot) {
	if values, err := load.Avg(); err == nil {
		result.Load1, result.Load5, result.Load15 = values.Load1, values.Load5, values.Load15
	}
	path := "/"
	if info, err := host.Info(); err == nil && info.OS == "windows" {
		path = "C:"
	}
	if values, err := disk.Usage(path); err == nil {
		result.Disk = values.UsedPercent
		result.DiskPath = path
	}
}

func collectNetworkRates(c *Collector, now time.Time, result *Snapshot) {
	values, err := net.IOCounters(false)
	if err != nil || len(values) == 0 {
		return
	}
	if !c.lastAt.IsZero() {
		seconds := now.Sub(c.lastAt).Seconds()
		result.NetIn = rate(values[0].BytesRecv, c.lastNet.BytesRecv, seconds)
		result.NetOut = rate(values[0].BytesSent, c.lastNet.BytesSent, seconds)
	}
	result.NetErrors = values[0].Errin + values[0].Errout
	c.lastNet = values[0]
}

func collectDiskIORates(c *Collector, now time.Time, result *Snapshot) {
	counters, err := disk.IOCounters()
	if err != nil {
		return
	}
	var total disk.IOCountersStat
	for _, counter := range counters {
		total.ReadBytes += counter.ReadBytes
		total.WriteBytes += counter.WriteBytes
	}
	if !c.lastAt.IsZero() {
		seconds := now.Sub(c.lastAt).Seconds()
		result.DiskRead = rate(total.ReadBytes, c.lastDiskIO.ReadBytes, seconds)
		result.DiskWrite = rate(total.WriteBytes, c.lastDiskIO.WriteBytes, seconds)
	}
	c.lastDiskIO = total
}

func listeningPorts() []uint32 {
	conns, err := net.Connections("tcp")
	if err != nil {
		return nil
	}
	seen := map[uint32]bool{}
	for _, conn := range conns {
		if conn.Status == "LISTEN" && conn.Laddr.Port > 0 {
			seen[conn.Laddr.Port] = true
		}
	}
	ports := make([]uint32, 0, len(seen))
	for port := range seen {
		ports = append(ports, port)
	}
	sort.Slice(ports, func(i, j int) bool { return ports[i] < ports[j] })
	return ports
}

func rate(current, previous uint64, seconds float64) float64 {
	if seconds <= 0 || current < previous {
		return 0
	}
	return float64(current-previous) / seconds
}

func topProcesses() []Process {
	processes, err := process.Processes()
	if err != nil {
		return nil
	}
	out := make([]Process, 0, len(processes))
	for _, handle := range processes {
		out = append(out, processSummary(handle))
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
