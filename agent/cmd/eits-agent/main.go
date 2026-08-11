package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const version = "0.4.0-alpha.1"

type Config struct {
	ServerURL, EnrollmentToken, DeviceName, StateFile, HostRoot, ProcRoot, LocalCheckHost string
	Interval                                                              time.Duration
}
type Identity struct {
	AgentID     string `json:"agent_id"`
	AgentSecret string `json:"agent_secret"`
}
type PortCheck struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	TimeoutSeconds int    `json:"timeout_seconds"`
	Protocol       string `json:"protocol"`
	UDPPayload     string `json:"udp_payload"`
	ExpectResponse bool   `json:"expect_response"`
}
type AgentConfig struct {
	Revision   int         `json:"revision"`
	PortChecks []PortCheck `json:"port_checks"`
}
type DiskMetric struct {
	Mountpoint string  `json:"mountpoint"`
	Filesystem string  `json:"filesystem"`
	Total      uint64  `json:"total"`
	Used       uint64  `json:"used"`
	Free       uint64  `json:"free"`
	Percent    float64 `json:"percent"`
}
type PortResult struct {
	CheckID   int     `json:"check_id"`
	IsUp      bool    `json:"is_up"`
	LatencyMS float64 `json:"latency_ms"`
	Error     string  `json:"error"`
}
type ProcessInfo struct {
	PID         int    `json:"pid"`
	Name        string `json:"name"`
	MemoryBytes uint64 `json:"memory_bytes"`
}
type Metrics struct {
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
	PortResults   []PortResult  `json:"port_results"`
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
type CPUStat struct{ Idle, Total uint64 }
type Client struct {
	cfg           Config
	identity      Identity
	http          *http.Client
	lastCPU       CPUStat
	lastInventory time.Time
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
func loadConfig() Config {
	interval, err := time.ParseDuration(env("EITS_INTERVAL", "30s"))
	if err != nil {
		log.Fatal(err)
	}
	return Config{
		ServerURL:       strings.TrimRight(env("EITS_SERVER_URL", "http://localhost:8000"), "/"),
		EnrollmentToken: os.Getenv("EITS_ENROLLMENT_TOKEN"), DeviceName: env("EITS_DEVICE_NAME", "eits-agent"),
		StateFile: env("EITS_STATE_FILE", "/data/agent.json"), HostRoot: env("EITS_HOST_ROOT", "/hostfs"),
		ProcRoot: env("EITS_PROC_ROOT", "/hostproc"), LocalCheckHost: os.Getenv("EITS_LOCAL_CHECK_HOST"), Interval: interval,
	}
}
func readIdentity(path string) (Identity, error) {
	var identity Identity
	data, err := os.ReadFile(path)
	if err != nil {
		return identity, err
	}
	return identity, json.Unmarshal(data, &identity)
}
func saveIdentity(path string, identity Identity) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(identity, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
func hostname() string { value, _ := os.Hostname(); return value }
func postJSON(client *http.Client, url string, payload any, headers map[string]string, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		if len(detail) > 0 {
			return fmt.Errorf("server returned %s: %s", resp.Status, strings.TrimSpace(string(detail)))
		}
		return fmt.Errorf("server returned %s", resp.Status)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}
func enroll(cfg Config, client *http.Client) (Identity, error) {
	var identity Identity
	if cfg.EnrollmentToken == "" {
		return identity, errors.New("EITS_ENROLLMENT_TOKEN is required for first enrollment")
	}
	payload := map[string]any{"enrollment_token": cfg.EnrollmentToken, "name": cfg.DeviceName, "hostname": hostname(), "os": runtime.GOOS, "architecture": runtime.GOARCH, "agent_version": version}
	err := postJSON(client, cfg.ServerURL+"/api/agent/enroll", payload, nil, &identity)
	return identity, err
}
func (c *Client) headers() map[string]string {
	return map[string]string{"X-Agent-ID": c.identity.AgentID, "X-Agent-Secret": c.identity.AgentSecret}
}
func (c *Client) getConfig() (AgentConfig, error) {
	var cfg AgentConfig
	req, _ := http.NewRequest(http.MethodGet, c.cfg.ServerURL+"/api/agent/config", nil)
	for key, value := range c.headers() {
		req.Header.Set(key, value)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return cfg, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return cfg, fmt.Errorf("config returned %s", resp.Status)
	}
	return cfg, json.NewDecoder(resp.Body).Decode(&cfg)
}
func checkPort(check PortCheck) PortResult {
	started := time.Now()
	protocol := strings.ToLower(strings.TrimSpace(check.Protocol))
	if protocol == "" {
		protocol = "tcp"
	}
	address := net.JoinHostPort(check.Host, strconv.Itoa(check.Port))
	timeout := time.Duration(check.TimeoutSeconds) * time.Second
	result := PortResult{CheckID: check.ID}

	if protocol == "udp" {
		conn, err := net.DialTimeout("udp", address, timeout)
		if err != nil {
			result.Error = err.Error()
			result.LatencyMS = float64(time.Since(started).Microseconds()) / 1000
			return result
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(timeout))
		payload := []byte(check.UDPPayload)
		if len(payload) == 0 {
			payload = []byte{0}
		}
		if _, err = conn.Write(payload); err != nil {
			result.Error = err.Error()
			result.LatencyMS = float64(time.Since(started).Microseconds()) / 1000
			return result
		}
		buffer := make([]byte, 2048)
		_, err = conn.Read(buffer)
		result.LatencyMS = float64(time.Since(started).Microseconds()) / 1000
		if err == nil {
			result.IsUp = true
			return result
		}
		if networkError, ok := err.(net.Error); ok && networkError.Timeout() && !check.ExpectResponse {
			// UDP has no handshake. A timeout after a successful send is treated as
			// reachable unless the check explicitly requires an application response.
			result.IsUp = true
			return result
		}
		result.Error = err.Error()
		return result
	}

	conn, err := net.DialTimeout("tcp", address, timeout)
	result.LatencyMS = float64(time.Since(started).Microseconds()) / 1000
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.IsUp = true
	_ = conn.Close()
	return result
}
func textFile(path string) string {
	data, _ := os.ReadFile(path)
	return strings.TrimSpace(string(data))
}
func hostPath(root, path string) string { return filepath.Join(root, strings.TrimPrefix(path, "/")) }
func collectInventory(cfg Config) Inventory {
	inv := Inventory{CollectedAt: time.Now().UTC(), DeviceType: "physical", GPUs: []GPUInfo{}}
	inv.Manufacturer = textFile(hostPath(cfg.HostRoot, "/sys/class/dmi/id/sys_vendor"))
	inv.Model = textFile(hostPath(cfg.HostRoot, "/sys/class/dmi/id/product_name"))
	inv.SerialNumber = textFile(hostPath(cfg.HostRoot, "/sys/class/dmi/id/product_serial"))
	inv.BIOSVersion = textFile(hostPath(cfg.HostRoot, "/sys/class/dmi/id/bios_version"))
	inv.KernelVersion = textFile(filepath.Join(cfg.ProcRoot, "sys/kernel/osrelease"))
	data, _ := os.ReadFile(hostPath(cfg.HostRoot, "/etc/os-release"))
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			v := strings.Trim(strings.TrimSpace(parts[1]), "\"")
			if parts[0] == "PRETTY_NAME" {
				inv.OSName = v
			}
			if parts[0] == "VERSION_ID" {
				inv.OSVersion = v
			}
		}
	}
	cpu, _ := os.ReadFile(filepath.Join(cfg.ProcRoot, "cpuinfo"))
	physical := map[string]bool{}
	for _, line := range strings.Split(string(cpu), "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		k, v := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		if inv.CPUModel == "" && (k == "model name" || k == "Hardware") {
			inv.CPUModel = v
		}
		if inv.CPUVendor == "" && (k == "vendor_id" || k == "CPU implementer") {
			inv.CPUVendor = v
		}
		if k == "physical id" {
			physical[v] = true
		}
	}
	inv.CPULogicalProcessors = runtime.NumCPU()
	inv.CPUPhysicalCores = inv.CPULogicalProcessors
	inv.TotalMemoryBytes, _, _ = readMemory(cfg.ProcRoot)
	lower := strings.ToLower(inv.Manufacturer + " " + inv.Model)
	if strings.Contains(lower, "virtual") || strings.Contains(lower, "kvm") || strings.Contains(lower, "vmware") {
		inv.DeviceType = "virtual"
	}
	apt, _ := os.ReadFile(hostPath(cfg.HostRoot, "/var/log/apt/history.log"))
	lines := strings.Split(strings.TrimSpace(string(apt)), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.HasPrefix(lines[i], "End-Date:") {
			inv.LastOSUpdate = strings.TrimSpace(strings.TrimPrefix(lines[i], "End-Date:"))
			break
		}
	}
	return inv
}
func (c *Client) sendInventory(force bool) error {
	if !force && time.Since(c.lastInventory) < 24*time.Hour {
		return nil
	}
	inv := collectInventory(c.cfg)
	if err := postJSON(c.http, c.cfg.ServerURL+"/api/agent/inventory", inv, c.headers(), nil); err != nil {
		return err
	}
	c.lastInventory = time.Now()
	log.Printf("inventory reported: %s %s, %s", inv.Manufacturer, inv.Model, inv.CPUModel)
	return nil
}

func readCPU(procRoot string) (CPUStat, error) {
	file, err := os.Open(filepath.Join(procRoot, "stat"))
	if err != nil {
		return CPUStat{}, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return CPUStat{}, errors.New("missing cpu line")
	}
	fields := strings.Fields(scanner.Text())
	if len(fields) < 8 || fields[0] != "cpu" {
		return CPUStat{}, errors.New("invalid cpu line")
	}
	var values []uint64
	for _, field := range fields[1:] {
		value, _ := strconv.ParseUint(field, 10, 64)
		values = append(values, value)
	}
	var total uint64
	for _, value := range values {
		total += value
	}
	return CPUStat{Idle: values[3] + values[4], Total: total}, nil
}
func cpuPercent(previous, current CPUStat) float64 {
	if previous.Total == 0 || current.Total <= previous.Total {
		return 0
	}
	totalDelta, idleDelta := current.Total-previous.Total, current.Idle-previous.Idle
	return 100 * float64(totalDelta-idleDelta) / float64(totalDelta)
}
func readMemory(procRoot string) (total, used uint64, percent float64) {
	data, err := os.ReadFile(filepath.Join(procRoot, "meminfo"))
	if err != nil {
		return
	}
	values := map[string]uint64{}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			v, _ := strconv.ParseUint(fields[1], 10, 64)
			values[strings.TrimSuffix(fields[0], ":")] = v * 1024
		}
	}
	total = values["MemTotal"]
	available := values["MemAvailable"]
	if total > available {
		used = total - available
	}
	if total > 0 {
		percent = 100 * float64(used) / float64(total)
	}
	return
}
func readUptime(procRoot string) uint64 {
	data, _ := os.ReadFile(filepath.Join(procRoot, "uptime"))
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0
	}
	value, _ := strconv.ParseFloat(fields[0], 64)
	return uint64(value)
}
func readNetwork(procRoot string) (sent, recv uint64) {
	data, err := os.ReadFile(filepath.Join(procRoot, "net/dev"))
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.Contains(line, ":") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		fields := strings.Fields(parts[1])
		if len(fields) < 9 {
			continue
		}
		r, _ := strconv.ParseUint(fields[0], 10, 64)
		s, _ := strconv.ParseUint(fields[8], 10, 64)
		recv += r
		sent += s
	}
	return
}
func readProcesses(procRoot string) []ProcessInfo {
	entries, err := os.ReadDir(procRoot)
	if err != nil {
		return []ProcessInfo{}
	}
	result := []ProcessInfo{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		base := filepath.Join(procRoot, entry.Name())
		nameData, _ := os.ReadFile(filepath.Join(base, "comm"))
		name := strings.TrimSpace(string(nameData))
		if name == "" {
			continue
		}
		var memory uint64
		statusData, _ := os.ReadFile(filepath.Join(base, "status"))
		for _, line := range strings.Split(string(statusData), "\n") {
			if strings.HasPrefix(line, "VmRSS:") {
				fields := strings.Fields(line)
				if len(fields) > 1 {
					value, _ := strconv.ParseUint(fields[1], 10, 64)
					memory = value * 1024
				}
				break
			}
		}
		result = append(result, ProcessInfo{PID: pid, Name: name, MemoryBytes: memory})
	}
	return result
}

func readDisks(hostRoot, procRoot string) []DiskMetric {
	// With host PID access, PID 1 exposes the host mount namespace. Reading the
	// proc root's own mounts from inside Docker would instead include container
	// bind mounts such as /etc/hostname and omit host filesystems such as ZFS.
	data, err := os.ReadFile(filepath.Join(procRoot, "1", "mounts"))
	if err != nil {
		data, err = os.ReadFile(filepath.Join(procRoot, "mounts"))
		if err != nil {
			return nil
		}
	}
	virtualFilesystems := map[string]bool{
		"autofs": true, "bpf": true, "cgroup": true, "cgroup2": true,
		"configfs": true, "debugfs": true, "devpts": true, "devtmpfs": true,
		"efivarfs": true, "fusectl": true, "hugetlbfs": true, "mqueue": true,
		"nsfs": true, "overlay": true, "proc": true, "pstore": true,
		"ramfs": true, "securityfs": true, "squashfs": true, "sysfs": true,
		"tmpfs": true, "tracefs": true,
	}
	seen := map[string]bool{}
	result := []DiskMetric{}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		mountpoint, fs := unescapeMount(fields[1]), fields[2]
		if seen[mountpoint] || virtualFilesystems[fs] || strings.HasPrefix(mountpoint, "/proc") || strings.HasPrefix(mountpoint, "/sys") || strings.HasPrefix(mountpoint, "/dev") || strings.HasPrefix(mountpoint, "/run") {
			continue
		}
		target := filepath.Join(hostRoot, strings.TrimPrefix(mountpoint, "/"))
		if mountpoint == "/" {
			target = hostRoot
		}
		info, err := os.Stat(target)
		if err != nil || !info.IsDir() {
			continue
		}
		var stat syscall.Statfs_t
		if err := syscall.Statfs(target, &stat); err != nil || stat.Blocks == 0 {
			continue
		}
		seen[mountpoint] = true
		total := stat.Blocks * uint64(stat.Bsize)
		free := stat.Bavail * uint64(stat.Bsize)
		used := total - free
		percent := 100 * float64(used) / float64(total)
		result = append(result, DiskMetric{Mountpoint: mountpoint, Filesystem: fs, Total: total, Used: used, Free: free, Percent: percent})
	}
	return result
}

func unescapeMount(value string) string {
	replacer := strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`)
	return replacer.Replace(value)
}
func (c *Client) collect(checks []PortCheck) Metrics {
	currentCPU, _ := readCPU(c.cfg.ProcRoot)
	cpuUsage := cpuPercent(c.lastCPU, currentCPU)
	c.lastCPU = currentCPU
	total, used, memPercent := readMemory(c.cfg.ProcRoot)
	sent, recv := readNetwork(c.cfg.ProcRoot)
	metrics := Metrics{RecordedAt: time.Now().UTC(), Hostname: hostname(), OS: runtime.GOOS, Architecture: runtime.GOARCH, AgentVersion: version, CPUPercent: cpuUsage, MemoryPercent: memPercent, MemoryTotal: total, MemoryUsed: used, UptimeSeconds: readUptime(c.cfg.ProcRoot), NetworkSent: sent, NetworkRecv: recv, Disks: readDisks(c.cfg.HostRoot, c.cfg.ProcRoot), PortResults: []PortResult{}, Processes: readProcesses(c.cfg.ProcRoot)}
	for _, check := range checks {
		if c.cfg.LocalCheckHost != "" {
			check.Host = c.cfg.LocalCheckHost
		}
		metrics.PortResults = append(metrics.PortResults, checkPort(check))
	}
	return metrics
}
func (c *Client) send(metrics Metrics) error {
	return postJSON(c.http, c.cfg.ServerURL+"/api/agent/metrics", metrics, c.headers(), nil)
}
func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	cfg := loadConfig()
	httpClient := &http.Client{Timeout: 20 * time.Second}
	identity, err := readIdentity(cfg.StateFile)
	if err != nil {
		log.Printf("No identity found; enrolling %s", cfg.DeviceName)
		identity, err = enroll(cfg, httpClient)
		if err != nil {
			log.Fatal(err)
		}
		if err = saveIdentity(cfg.StateFile, identity); err != nil {
			log.Fatal(err)
		}
	}
	client := &Client{cfg: cfg, identity: identity, http: httpClient}
	client.lastCPU, _ = readCPU(cfg.ProcRoot)
	log.Printf("EITS agent %s started as %s", version, identity.AgentID)
	if err := client.sendInventory(true); err != nil {
		log.Printf("inventory error: %v", err)
	}
	for {
		if err := client.sendInventory(false); err != nil {
			log.Printf("inventory error: %v", err)
		}
		agentCfg, err := client.getConfig()
		if err != nil {
			log.Printf("config error: %v", err)
		}
		metrics := client.collect(agentCfg.PortChecks)
		if err := client.send(metrics); err != nil {
			log.Printf("upload error: %v", err)
		} else {
			log.Printf("reported cpu=%.1f%% memory=%.1f%% disks=%d ports=%d processes=%d", metrics.CPUPercent, metrics.MemoryPercent, len(metrics.Disks), len(metrics.PortResults), len(metrics.Processes))
		}
		time.Sleep(cfg.Interval)
	}
}
