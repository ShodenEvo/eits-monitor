//go:build linux

package collector

import (
	"bufio"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func readCPUTimes() (cpuTimes, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return cpuTimes{}, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	if !s.Scan() {
		return cpuTimes{}, errors.New("missing /proc/stat cpu line")
	}
	fields := strings.Fields(s.Text())
	if len(fields) < 8 || fields[0] != "cpu" {
		return cpuTimes{}, errors.New("invalid /proc/stat cpu line")
	}
	vals := make([]uint64, 0, len(fields)-1)
	var total uint64
	for _, field := range fields[1:] {
		v, _ := strconv.ParseUint(field, 10, 64)
		vals = append(vals, v)
		total += v
	}
	return cpuTimes{idle: vals[3] + vals[4], total: total}, nil
}

func readMemory() (uint64, uint64, float64, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, 0, err
	}
	values := map[string]uint64{}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			v, _ := strconv.ParseUint(fields[1], 10, 64)
			values[strings.TrimSuffix(fields[0], ":")] = v * 1024
		}
	}
	total := values["MemTotal"]
	available := values["MemAvailable"]
	if available == 0 {
		available = values["MemFree"] + values["Buffers"] + values["Cached"]
	}
	used := total - available
	percent := 0.0
	if total > 0 {
		percent = 100 * float64(used) / float64(total)
	}
	return total, used, percent, nil
}

func readUptime() (uint64, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0, errors.New("invalid /proc/uptime")
	}
	seconds, err := strconv.ParseFloat(fields[0], 64)
	return uint64(seconds), err
}

func readDisks() ([]DiskMetric, error) {
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	ignored := map[string]bool{"proc": true, "sysfs": true, "tmpfs": true, "devtmpfs": true, "devpts": true, "cgroup": true, "cgroup2": true, "overlay": true, "squashfs": true, "securityfs": true, "pstore": true, "debugfs": true, "tracefs": true, "configfs": true, "fusectl": true, "mqueue": true, "hugetlbfs": true, "rpc_pipefs": true, "autofs": true, "binfmt_misc": true}
	disks := []DiskMetric{}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		mount := strings.ReplaceAll(fields[1], `\040`, " ")
		fs := fields[2]
		if ignored[fs] || seen[mount] || strings.HasPrefix(mount, "/snap/") {
			continue
		}
		seen[mount] = true
		var st syscall.Statfs_t
		if err := syscall.Statfs(mount, &st); err != nil {
			continue
		}
		total := st.Blocks * uint64(st.Bsize)
		free := st.Bavail * uint64(st.Bsize)
		if total == 0 {
			continue
		}
		used := total - free
		disks = append(disks, DiskMetric{Mountpoint: filepath.Clean(mount), Filesystem: fs, Total: total, Used: used, Free: free, Percent: 100 * float64(used) / float64(total)})
	}
	return disks, nil
}

func readNetwork() (uint64, uint64) {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return 0, 0
	}
	var recv, sent uint64
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.Contains(line, ":") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		iface := strings.TrimSpace(parts[0])
		if iface == "lo" {
			continue
		}
		fields := strings.Fields(parts[1])
		if len(fields) < 9 {
			continue
		}
		r, _ := strconv.ParseUint(fields[0], 10, 64)
		s, _ := strconv.ParseUint(fields[8], 10, 64)
		recv += r
		sent += s
	}
	return sent, recv
}

func readText(path string) string {
	data, _ := os.ReadFile(path)
	return strings.TrimSpace(string(data))
}
func osRelease() map[string]string {
	result := map[string]string{}
	data, _ := os.ReadFile("/etc/os-release")
	for _, line := range strings.Split(string(data), "\n") {
		if p := strings.Index(line, "="); p > 0 {
			result[line[:p]] = strings.Trim(strings.TrimSpace(line[p+1:]), "\"")
		}
	}
	return result
}
func readInventory() (Inventory, error) {
	inv := Inventory{CollectedAt: time.Now().UTC(), DeviceType: "physical", GPUs: []GPUInfo{}}
	rel := osRelease()
	inv.OSName = rel["PRETTY_NAME"]
	inv.OSVersion = rel["VERSION_ID"]
	inv.KernelVersion = readText("/proc/sys/kernel/osrelease")
	inv.Manufacturer = readText("/sys/class/dmi/id/sys_vendor")
	inv.Model = readText("/sys/class/dmi/id/product_name")
	inv.SerialNumber = readText("/sys/class/dmi/id/product_serial")
	inv.BIOSVersion = readText("/sys/class/dmi/id/bios_version")
	if strings.Contains(strings.ToLower(inv.Manufacturer+" "+inv.Model), "virtual") || strings.Contains(strings.ToLower(inv.Model), "kvm") {
		inv.DeviceType = "virtual"
	}
	data, _ := os.ReadFile("/proc/cpuinfo")
	cores := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
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
			cores[v] = true
		}
	}
	inv.CPULogicalProcessors = runtime.NumCPU()
	inv.CPUPhysicalCores = inv.CPULogicalProcessors
	if out, err := exec.Command("sh", "-c", "lscpu -p=CORE,SOCKET 2>/dev/null | grep -v '^#' | sort -u | wc -l").Output(); err == nil {
		if n, _ := strconv.Atoi(strings.TrimSpace(string(out))); n > 0 {
			inv.CPUPhysicalCores = n
		}
	}
	inv.TotalMemoryBytes, _, _, _ = readMemory()
	if out, err := exec.Command("sh", "-c", "lspci 2>/dev/null | grep -Ei 'VGA|3D controller|Display controller'").Output(); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if line != "" {
				inv.GPUs = append(inv.GPUs, GPUInfo{Model: line})
			}
		}
	}
	if data, err := os.ReadFile("/var/log/apt/history.log"); err == nil {
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		for i := len(lines) - 1; i >= 0; i-- {
			if strings.HasPrefix(lines[i], "End-Date:") {
				inv.LastOSUpdate = strings.TrimSpace(strings.TrimPrefix(lines[i], "End-Date:"))
				break
			}
		}
	}
	return inv, nil
}
