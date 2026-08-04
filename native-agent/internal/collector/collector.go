package collector

import (
	"os"
	"runtime"
	"time"
)

type Collector struct{ previous cpuTimes }

func New() *Collector { return &Collector{} }

func (c *Collector) Collect(version string) (Snapshot, error) {
	host, _ := os.Hostname()
	current, err := readCPUTimes()
	if err != nil {
		return Snapshot{}, err
	}
	cpu := calculateCPU(c.previous, current)
	c.previous = current
	total, used, memPercent, err := readMemory()
	if err != nil {
		return Snapshot{}, err
	}
	disks, err := readDisks()
	if err != nil {
		return Snapshot{}, err
	}
	uptime, err := readUptime()
	if err != nil {
		return Snapshot{}, err
	}
	sent, recv := readNetwork()
	return Snapshot{
		RecordedAt: time.Now().UTC(), Hostname: host, OS: runtime.GOOS,
		Architecture: runtime.GOARCH, AgentVersion: version, CPUPercent: cpu,
		MemoryPercent: memPercent, MemoryTotal: total, MemoryUsed: used,
		UptimeSeconds: uptime, NetworkSent: sent, NetworkRecv: recv,
		Disks: disks,
	}, nil
}
