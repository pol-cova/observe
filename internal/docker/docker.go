package docker

import (
	"bufio"
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
)

type Container struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	CPU        float64 `json:"cpu_percent"`
	Memory     float64 `json:"memory_percent"`
	NetworkIn  float64 `json:"network_in_bytes"`
	NetworkOut float64 `json:"network_out_bytes"`
	Ports      string  `json:"ports"`
	Status     string  `json:"status"`
}

type cliContainer struct{ ID, Name, CPUPerc, MemPerc, NetIO, Ports, Status string }
type cliMetadata struct{ ID, Name, Ports, Status string }

func Collect() ([]Container, error) {
	metadata := map[string]cliMetadata{}
	if out, err := exec.Command("docker", "ps", "-a", "--format", "{{json .}}").Output(); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			var item cliMetadata
			if json.Unmarshal([]byte(line), &item) == nil {
				metadata[item.ID] = item
			}
		}
	}
	cmd := exec.Command("docker", "stats", "--no-stream", "--all", "--format", "{{json .}}")
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	var result []Container
	s := bufio.NewScanner(out)
	for s.Scan() {
		var item cliContainer
		if json.Unmarshal(s.Bytes(), &item) != nil {
			continue
		}
		if meta, ok := metadata[item.ID]; ok {
			item.Name, item.Ports, item.Status = meta.Name, meta.Ports, meta.Status
		}
		result = append(result, parse(item))
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}
	return result, s.Err()
}

func parse(item cliContainer) Container {
	return Container{ID: item.ID, Name: item.Name, CPU: percent(item.CPUPerc), Memory: percent(item.MemPerc), NetworkIn: ioValue(item.NetIO, 0), NetworkOut: ioValue(item.NetIO, 1), Ports: item.Ports, Status: item.Status}
}
func percent(v string) float64 {
	v = strings.TrimSuffix(strings.TrimSpace(v), "%")
	n, _ := strconv.ParseFloat(v, 64)
	return n
}
func ioValue(v string, index int) float64 {
	parts := strings.Split(v, "/")
	if len(parts) != 2 {
		return 0
	}
	return bytesValue(strings.TrimSpace(parts[index]))
}
func bytesValue(v string) float64 {
	fields := strings.Fields(v)
	if len(fields) == 0 {
		return 0
	}
	value := fields[0]
	unit := ""
	if i := strings.IndexFunc(value, func(r rune) bool { return (r < '0' || r > '9') && r != '.' }); i >= 0 {
		value, unit = value[:i], value[i:]
	}
	if len(fields) > 1 {
		unit = fields[1]
	}
	n, _ := strconv.ParseFloat(value, 64)
	switch strings.ToUpper(unit) {
	case "KB", "KIB":
		return n * 1024
	case "MB", "MIB":
		return n * 1024 * 1024
	case "GB", "GIB":
		return n * 1024 * 1024 * 1024
	}
	return n
}
