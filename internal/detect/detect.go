package detect

import (
	"fmt"
	"net"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/paulcontreras/observe/internal/metrics/local"
)

type Report struct {
	Ports                          []uint32
	Docker, Prometheus, Kubernetes bool
}

func Scan() (Report, error) {
	c := local.New()
	s, err := c.Collect()
	if err != nil {
		return Report{}, err
	}
	r := Report{Ports: s.Ports}
	r.Docker = available("docker")
	r.Kubernetes = available("kubectl")
	r.Prometheus = reachable("127.0.0.1:9090")
	return r, nil
}
func available(binary string) bool { _, err := exec.LookPath(binary); return err == nil }
func reachable(addr string) bool {
	c, e := net.DialTimeout("tcp", addr, 500*time.Millisecond)
	if e == nil {
		c.Close()
		return true
	}
	return false
}
func (r Report) String() string {
	var b strings.Builder
	b.WriteString("observe setup scan\n\n")
	if len(r.Ports) == 0 {
		b.WriteString("No listening TCP ports found.\n")
	} else {
		ports := append([]uint32(nil), r.Ports...)
		sort.Slice(ports, func(i, j int) bool { return ports[i] < ports[j] })
		fmt.Fprintf(&b, "Listening TCP ports: %v\n", ports)
	}
	fmt.Fprintf(&b, "Docker: %s\nKubernetes CLI: %s\nPrometheus on localhost:9090: %s\n", yesNo(r.Docker), yesNo(r.Kubernetes), yesNo(r.Prometheus))
	b.WriteString("\nStart the local dashboard with: observe\n")
	if r.Prometheus {
		b.WriteString("Prometheus detected: observe --prometheus http://localhost:9090\n")
	} else {
		b.WriteString("For app metrics, expose a Prometheus /metrics endpoint and run Prometheus, then pass --prometheus.\n")
	}
	return b.String()
}
func yesNo(v bool) string {
	if v {
		return "detected"
	}
	return "not found"
}
