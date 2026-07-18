//go:build windows

package embeddedmodels

import (
	"syscall"
	"unsafe"
)

// memoryStatusEx mirrors Win32 MEMORYSTATUSEX (golang.org/x/sys/windows does not export GlobalMemoryStatusEx).
type memoryStatusEx struct {
	length                uint32
	memoryLoad            uint32
	totalPhys             uint64
	availPhys             uint64
	totalPageFile         uint64
	availPageFile         uint64
	totalVirtual          uint64
	availVirtual          uint64
	availExtendedVirtual  uint64
}

var (
	modKernel32              = syscall.NewLazyDLL("kernel32.dll")
	procGlobalMemoryStatusEx = modKernel32.NewProc("GlobalMemoryStatusEx")
)

func detectPlatformRAMGb() float64 {
	var st memoryStatusEx
	st.length = uint32(unsafe.Sizeof(st))
	r, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&st)))
	if r == 0 {
		return 0
	}
	return float64(st.totalPhys) / (1024 * 1024 * 1024)
}
