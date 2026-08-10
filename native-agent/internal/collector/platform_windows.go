//go:build windows

package collector

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procGetSystemTimes           = kernel32.NewProc("GetSystemTimes")
	procGlobalMemoryStatusEx     = kernel32.NewProc("GlobalMemoryStatusEx")
	procGetTickCount64           = kernel32.NewProc("GetTickCount64")
	procGetLogicalDrives         = kernel32.NewProc("GetLogicalDrives")
	procGetDriveTypeW            = kernel32.NewProc("GetDriveTypeW")
	procGetDiskFreeSpaceExW      = kernel32.NewProc("GetDiskFreeSpaceExW")
	procCreateToolhelp32Snapshot = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32FirstW          = kernel32.NewProc("Process32FirstW")
	procProcess32NextW           = kernel32.NewProc("Process32NextW")
	procCloseHandle              = kernel32.NewProc("CloseHandle")
)

type filetime struct{ low, high uint32 }

func (f filetime) value() uint64 { return uint64(f.high)<<32 | uint64(f.low) }

type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

type processEntry32 struct {
	Size            uint32
	Usage           uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	Threads         uint32
	ParentProcessID uint32
	PriorityBase    int32
	Flags           uint32
	ExeFile         [260]uint16
}

func readCPUTimes() (cpuTimes, error) {
	var idle, kernel, user filetime
	r, _, err := procGetSystemTimes.Call(uintptr(unsafe.Pointer(&idle)), uintptr(unsafe.Pointer(&kernel)), uintptr(unsafe.Pointer(&user)))
	if r == 0 {
		return cpuTimes{}, err
	}
	return cpuTimes{idle: idle.value(), total: kernel.value() + user.value()}, nil
}

func readMemory() (uint64, uint64, float64, error) {
	status := memoryStatusEx{Length: uint32(unsafe.Sizeof(memoryStatusEx{}))}
	r, _, err := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&status)))
	if r == 0 {
		return 0, 0, 0, err
	}
	used := status.TotalPhys - status.AvailPhys
	percent := 0.0
	if status.TotalPhys > 0 {
		percent = 100 * float64(used) / float64(status.TotalPhys)
	}
	return status.TotalPhys, used, percent, nil
}

func readUptime() (uint64, error) {
	r, _, _ := procGetTickCount64.Call()
	return uint64(r) / 1000, nil
}

func readDisks() ([]DiskMetric, error) {
	mask, _, err := procGetLogicalDrives.Call()
	if mask == 0 {
		return nil, err
	}
	disks := []DiskMetric{}
	for i := 0; i < 26; i++ {
		if mask&(1<<i) == 0 {
			continue
		}
		root := fmt.Sprintf("%c:\\", 'A'+i)
		rootPtr, _ := syscall.UTF16PtrFromString(root)
		driveType, _, _ := procGetDriveTypeW.Call(uintptr(unsafe.Pointer(rootPtr)))
		// DRIVE_FIXED=3, DRIVE_REMOVABLE=2. Ignore CD/network/RAM drives.
		if driveType != 3 && driveType != 2 {
			continue
		}
		var freeAvailable, total, freeTotal uint64
		r, _, _ := procGetDiskFreeSpaceExW.Call(uintptr(unsafe.Pointer(rootPtr)), uintptr(unsafe.Pointer(&freeAvailable)), uintptr(unsafe.Pointer(&total)), uintptr(unsafe.Pointer(&freeTotal)))
		if r == 0 || total == 0 {
			continue
		}
		used := total - freeTotal
		disks = append(disks, DiskMetric{Mountpoint: root, Filesystem: "windows", Total: total, Used: used, Free: freeTotal, Percent: 100 * float64(used) / float64(total)})
	}
	return disks, nil
}

func readNetwork() (uint64, uint64) { return 0, 0 }

func readProcesses() []ProcessInfo {
	const th32csSnapProcess = 0x00000002
	handle, _, _ := procCreateToolhelp32Snapshot.Call(th32csSnapProcess, 0)
	if handle == 0 || handle == ^uintptr(0) {
		return []ProcessInfo{}
	}
	defer procCloseHandle.Call(handle)
	entry := processEntry32{Size: uint32(unsafe.Sizeof(processEntry32{}))}
	ok, _, _ := procProcess32FirstW.Call(handle, uintptr(unsafe.Pointer(&entry)))
	result := []ProcessInfo{}
	for ok != 0 {
		name := syscall.UTF16ToString(entry.ExeFile[:])
		if name != "" {
			result = append(result, ProcessInfo{PID: int(entry.ProcessID), Name: name})
		}
		entry.Size = uint32(unsafe.Sizeof(processEntry32{}))
		ok, _, _ = procProcess32NextW.Call(handle, uintptr(unsafe.Pointer(&entry)))
	}
	return result
}

func readInventory() (Inventory, error) {
	script := `$ErrorActionPreference='SilentlyContinue';$cs=Get-CimInstance Win32_ComputerSystem;$os=Get-CimInstance Win32_OperatingSystem;$cpu=@(Get-CimInstance Win32_Processor);$bios=Get-CimInstance Win32_BIOS;$gpu=@(Get-CimInstance Win32_VideoController|ForEach-Object{[pscustomobject]@{vendor='';model=$_.Name;memory_bytes=[uint64]$_.AdapterRAM;driver_version=$_.DriverVersion}});$hotfix=Get-HotFix|Sort-Object InstalledOn -Descending|Select-Object -First 1;[pscustomobject]@{manufacturer=$cs.Manufacturer;model=$cs.Model;serial_number=$bios.SerialNumber;device_type=if($cs.Model -match 'Virtual|VMware|VirtualBox|KVM'){ 'virtual' }else{'physical'};os_name=$os.Caption;os_version=$os.Version;os_build=$os.BuildNumber;kernel_version=$os.Version;last_os_update=if($hotfix){$hotfix.HotFixID+' '+$hotfix.InstalledOn}else{''};cpu_vendor=if($cpu.Count){$cpu[0].Manufacturer}else{''};cpu_model=if($cpu.Count){$cpu[0].Name}else{''};cpu_physical_cores=($cpu|Measure-Object NumberOfCores -Sum).Sum;cpu_logical_processors=($cpu|Measure-Object NumberOfLogicalProcessors -Sum).Sum;total_memory_bytes=[uint64]$cs.TotalPhysicalMemory;bios_version=($bios.SMBIOSBIOSVersion);gpus=$gpu}|ConvertTo-Json -Depth 5 -Compress`
	out, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script).Output()
	if err != nil {
		return Inventory{}, err
	}
	var inv Inventory
	if err := json.Unmarshal(out, &inv); err != nil {
		return Inventory{}, err
	}
	inv.CollectedAt = time.Now().UTC()
	if inv.CPULogicalProcessors == 0 {
		inv.CPULogicalProcessors = runtime.NumCPU()
	}
	if inv.CPUPhysicalCores == 0 {
		inv.CPUPhysicalCores = inv.CPULogicalProcessors
	}
	if inv.GPUs == nil {
		inv.GPUs = []GPUInfo{}
	}
	inv.Manufacturer = strings.TrimSpace(inv.Manufacturer)
	inv.Model = strings.TrimSpace(inv.Model)
	return inv, nil
}
