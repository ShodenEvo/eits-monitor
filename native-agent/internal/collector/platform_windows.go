//go:build windows

package collector

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	kernel32                 = syscall.NewLazyDLL("kernel32.dll")
	procGetSystemTimes       = kernel32.NewProc("GetSystemTimes")
	procGlobalMemoryStatusEx = kernel32.NewProc("GlobalMemoryStatusEx")
	procGetTickCount64       = kernel32.NewProc("GetTickCount64")
	procGetLogicalDrives     = kernel32.NewProc("GetLogicalDrives")
	procGetDriveTypeW        = kernel32.NewProc("GetDriveTypeW")
	procGetDiskFreeSpaceExW  = kernel32.NewProc("GetDiskFreeSpaceExW")
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
