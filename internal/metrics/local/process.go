package local

import (
	"fmt"
	"sort"

	psnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

// ProcessDetails contains the system information useful when investigating a busy process.
type ProcessDetails struct {
	Process
	Executable  string
	Command     string
	ParentPID   int32
	Children    []Process
	OpenFiles   []string
	Connections []Connection
}

type Connection struct {
	Type   string
	Local  string
	Remote string
	Status string
}

func Inspect(pid int32) (ProcessDetails, error) {
	processHandle, err := process.NewProcess(pid)
	if err != nil {
		return ProcessDetails{}, err
	}

	details := ProcessDetails{Process: processSummary(processHandle)}
	details.Executable, _ = processHandle.Exe()
	details.Command, _ = processHandle.Cmdline()
	if parent, err := processHandle.Parent(); err == nil && parent != nil {
		details.ParentPID = parent.Pid
	}
	details.Children = childProcesses(processHandle)
	details.OpenFiles = openFiles(processHandle)
	details.Connections = connections(processHandle)
	return details, nil
}

func processSummary(handle *process.Process) Process {
	name, _ := handle.Name()
	cpu, _ := handle.CPUPercent()
	memory, _ := handle.MemoryPercent()
	return Process{PID: handle.Pid, Name: name, CPU: cpu, Memory: float64(memory)}
}

func childProcesses(handle *process.Process) []Process {
	children, err := handle.Children()
	if err != nil {
		return nil
	}
	result := make([]Process, 0, len(children))
	for _, child := range children {
		result = append(result, processSummary(child))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CPU > result[j].CPU })
	return result
}

func openFiles(handle *process.Process) []string {
	files, err := handle.OpenFiles()
	if err != nil {
		return nil
	}
	result := make([]string, 0, len(files))
	for _, file := range files {
		result = append(result, file.Path)
	}
	sort.Strings(result)
	return result
}

func connections(handle *process.Process) []Connection {
	stats, err := handle.Connections()
	if err != nil {
		return nil
	}
	result := make([]Connection, 0, len(stats))
	for _, stat := range stats {
		result = append(result, formatConnection(stat))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Local < result[j].Local })
	return result
}

func formatConnection(stat psnet.ConnectionStat) Connection {
	return Connection{
		Type:   connectionType(stat.Type),
		Local:  address(stat.Laddr.IP, stat.Laddr.Port),
		Remote: address(stat.Raddr.IP, stat.Raddr.Port),
		Status: stat.Status,
	}
}

func connectionType(value uint32) string {
	switch value {
	case 1:
		return "tcp"
	case 2:
		return "udp"
	default:
		return fmt.Sprintf("socket/%d", value)
	}
}

func address(ip string, port uint32) string {
	if ip == "" && port == 0 {
		return "-"
	}
	return fmt.Sprintf("%s:%d", ip, port)
}
