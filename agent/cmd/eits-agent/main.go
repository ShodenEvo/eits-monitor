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

const version = "0.2.0"

type Config struct {
	ServerURL, EnrollmentToken, DeviceName, StateFile, HostRoot, ProcRoot string
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
type Metrics struct {
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
	PortResults   []PortResult `json:"port_results"`
}
type CPUStat struct{ Idle, Total uint64 }
type Client struct {
	cfg      Config
	identity Identity
	http     *http.Client
	lastCPU  CPUStat
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
		ProcRoot: env("EITS_PROC_ROOT", "/hostproc"), Interval: interval,
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
func readDisks(hostRoot, procRoot string) []DiskMetric {
	data, err := os.ReadFile(filepath.Join(procRoot, "mounts"))
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	result := []DiskMetric{}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		mountpoint, fs := strings.ReplaceAll(fields[1], `\040`, " "), fields[2]
		if seen[mountpoint] || strings.HasPrefix(mountpoint, "/proc") || strings.HasPrefix(mountpoint, "/sys") || strings.HasPrefix(mountpoint, "/dev") || strings.HasPrefix(mountpoint, "/run") {
			continue
		}
		seen[mountpoint] = true
		target := filepath.Join(hostRoot, strings.TrimPrefix(mountpoint, "/"))
		if mountpoint == "/" {
			target = hostRoot
		}
		var stat syscall.Statfs_t
		if err := syscall.Statfs(target, &stat); err != nil || stat.Blocks == 0 {
			continue
		}
		total := stat.Blocks * uint64(stat.Bsize)
		free := stat.Bavail * uint64(stat.Bsize)
		used := total - free
		percent := 100 * float64(used) / float64(total)
		result = append(result, DiskMetric{Mountpoint: mountpoint, Filesystem: fs, Total: total, Used: used, Free: free, Percent: percent})
	}
	return result
}
func (c *Client) collect(checks []PortCheck) Metrics {
	currentCPU, _ := readCPU(c.cfg.ProcRoot)
	cpuUsage := cpuPercent(c.lastCPU, currentCPU)
	c.lastCPU = currentCPU
	total, used, memPercent := readMemory(c.cfg.ProcRoot)
	sent, recv := readNetwork(c.cfg.ProcRoot)
	metrics := Metrics{RecordedAt: time.Now().UTC(), Hostname: hostname(), OS: runtime.GOOS, Architecture: runtime.GOARCH, AgentVersion: version, CPUPercent: cpuUsage, MemoryPercent: memPercent, MemoryTotal: total, MemoryUsed: used, UptimeSeconds: readUptime(c.cfg.ProcRoot), NetworkSent: sent, NetworkRecv: recv, Disks: readDisks(c.cfg.HostRoot, c.cfg.ProcRoot), PortResults: []PortResult{}}
	for _, check := range checks {
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
	for {
		agentCfg, err := client.getConfig()
		if err != nil {
			log.Printf("config error: %v", err)
		}
		metrics := client.collect(agentCfg.PortChecks)
		if err := client.send(metrics); err != nil {
			log.Printf("upload error: %v", err)
		} else {
			log.Printf("reported cpu=%.1f%% memory=%.1f%% disks=%d ports=%d", metrics.CPUPercent, metrics.MemoryPercent, len(metrics.Disks), len(metrics.PortResults))
		}
		time.Sleep(cfg.Interval)
	}
}
