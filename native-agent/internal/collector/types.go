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

type ProcessInfo struct {
	PID         int    `json:"pid"`
	Name        string `json:"name"`
	MemoryBytes uint64 `json:"memory_bytes"`
}

type Snapshot struct {
	RecordedAt    time.Time     `json:"recorded_at"`
	Hostname      string        `json:"hostname"`
	OS            string        `json:"os"`
	Architecture  string        `json:"architecture"`
	AgentVersion  string        `json:"agent_version"`
	CPUPercent    float64       `json:"cpu_percent"`
	MemoryPercent float64       `json:"memory_percent"`
	MemoryTotal   uint64        `json:"memory_total"`
	MemoryUsed    uint64        `json:"memory_used"`
	UptimeSeconds uint64        `json:"uptime_seconds"`
	NetworkSent   uint64        `json:"network_sent"`
	NetworkRecv   uint64        `json:"network_recv"`
	Disks         []DiskMetric  `json:"disks"`
	Processes     []ProcessInfo `json:"processes"`
}

type GPUInfo struct {
	Vendor        string `json:"vendor"`
	Model         string `json:"model"`
	MemoryBytes   uint64 `json:"memory_bytes"`
	DriverVersion string `json:"driver_version"`
}

type Inventory struct {
	CollectedAt          time.Time `json:"collected_at"`
	Manufacturer         string    `json:"manufacturer"`
	Model                string    `json:"model"`
	SerialNumber         string    `json:"serial_number"`
	DeviceType           string    `json:"device_type"`
	OSName               string    `json:"os_name"`
	OSVersion            string    `json:"os_version"`
	OSBuild              string    `json:"os_build"`
	KernelVersion        string    `json:"kernel_version"`
	LastOSUpdate         string    `json:"last_os_update"`
	CPUVendor            string    `json:"cpu_vendor"`
	CPUModel             string    `json:"cpu_model"`
	CPUPhysicalCores     int       `json:"cpu_physical_cores"`
	CPULogicalProcessors int       `json:"cpu_logical_processors"`
	TotalMemoryBytes     uint64    `json:"total_memory_bytes"`
	BIOSVersion          string    `json:"bios_version"`
	GPUs                 []GPUInfo `json:"gpus"`
}
