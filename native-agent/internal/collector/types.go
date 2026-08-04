package collector

import "time"

type DiskMetric struct {
	Mountpoint string  `json:"mountpoint"`
	Filesystem string  `json:"filesystem"`
	Total      uint64  `json:"total"`
	Used       uint64  `json:"used"`
	Free       uint64  `json:"free"`
	Percent    float64 `json:"percent"`
}

type Snapshot struct {
	RecordedAt    time.Time    `json:"recorded_at"`
	Hostname      string       `json:"hostname"`
	OS            string       `json:"os"`
	Architecture  string       `json:"architecture"`
	AgentVersion  string       `json:"agent_version"`
	CPUPercent    float64      `json:"cpu_percent"`
	MemoryPercent float64      `json:"memory_percent"`
	MemoryTotal   uint64       `json:"memory_total"`
	MemoryUsed    uint64       `json:"memory_used"`
	UptimeSeconds uint64       `json:"uptime_seconds"`
	NetworkSent   uint64       `json:"network_sent"`
	NetworkRecv   uint64       `json:"network_recv"`
	Disks         []DiskMetric `json:"disks"`
}
